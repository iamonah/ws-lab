# WebSocket Architecture & Design Notes (Go)

This repository is a learning and reference implementation of WebSockets in Go.
The goal is not just to "make it work", but to understand the lifecycle, concurrency model, and failure modes of WebSockets so this code can be reused confidently in the future.

The implementation follows best practices used with Gorilla WebSocket.

## 1. Core Mental Model

A WebSocket connection is:
- Long-lived
- Stateful
- Bidirectional
- Expected to stay open indefinitely

So by design:
- Read loops run forever
- Write loops run forever
- The only reason they stop is connection failure or intentional shutdown

## 2. One Connection = Two Goroutines

Each WebSocket connection has two independent loops:

### Read loop
- Reads messages sent by the client
- Blocks on `ReadMessage()`
- Exits only on error

### Write loop
- Sends messages to the client
- Reads from an internal send channel
- Exits only when the channel is closed or a write fails

These loops run concurrently and independently.

## 3. Why We Use a Manager (Hub)

The manager (or hub) exists to:
- Track all active client connections
- Broadcast messages safely
- Prevent writes to dead connections
- Ensure proper cleanup

The client list represents:
- All currently alive and usable WebSocket connections

Clients are not removed after reading or writing messages.
They are removed only when the connection is no longer usable.

## 4. Infinite Loops Are Intentional

Both read and write loops are written as:

```go
for {
    // block forever
}
```

This is intentional.

WebSockets are event-driven, not request/response.
A loop exits only when something goes wrong.

## 5. When and Why Clients Are Removed

A client is removed from the manager only when:
- `ReadMessage()` returns an error
- `WriteMessage()` returns an error
- The server intentionally shuts the connection down

Once a read or write fails:
- The connection is considered permanently invalid
- There is no retry on the same connection.
- Cleanup happens immediately.

## 6. Why Cleanup Exists in BOTH Read and Write Loops

Both loops include cleanup logic like:

```go
defer manager.unregister <- client
```

This is not redundancy — it is defensive design.

**Reason:**
- Either loop may be the first to detect failure
- WebSocket failures are asymmetric
- The network does not fail politely

**Rule:**
Whichever goroutine detects failure first is responsible for cleanup.

Cleanup must be:
- Idempotent
- Safe to call once
- Safe to attempt more than once

## 7. One Concurrent Writer Rule (Very Important)

In Gorilla WebSocket:
- A connection allows only ONE concurrent writer

Calling `WriteMessage()` from multiple goroutines will cause:
- Data corruption
- Panics
- Undefined behavior

This is by design.

## 8. The Correct Fix: Single Writer + Channel

Each client has:

```go
send chan []byte
```

**Pattern:**
- Only one goroutine ever calls `WriteMessage`
- All other goroutines send messages into the channel

This gives:
- Serialized writes
- Backpressure
- Spam protection
- Predictable behavior

This pattern scales safely under load.

## 9. Why We Don’t Just Use a Mutex

Using a mutex around `WriteMessage()` technically works, but:
- Hides backpressure
- Encourages unbounded memory usage
- Scales poorly
- Makes failure handling harder

Channels + single writer is the idiomatic Go solution.

## 10. WebSocket Close Semantics

WebSockets define a close handshake:
1. One side sends a Close frame
2. The other side responds with a Close frame
3. TCP connection closes

This is why you’ll see:

```go
conn.WriteMessage(websocket.CloseMessage, nil)
```

This sends a polite "goodbye".

## 11. Why Send a Close Frame If the Client Is Gone?

Because you don’t always know if the client is gone.

Possible states when a loop exits:
- Client truly disconnected (network/browser closed)
- Server is intentionally closing the connection
- Half-open connection (TCP looks alive, but client isn’t responding)

Sending a Close frame is:
- Best-effort
- Protocol-correct
- Harmless if the client is already gone

If the write fails, it’s ignored.

## 12. Close Codes You’ll Commonly See

| Code | Meaning | Interpretation |
|------|---------|----------------|
| 1000 | Normal closure | Clean, graceful shutdown |
| 1001 | Going away | Browser closed, page refresh |
| 1006 | Abnormal closure | Network drop, crash, proxy timeout |

**Important:**
- 1006 is never sent on the wire
- It’s reported locally by the implementation
- 1001 and 1006 are normal in real systems and should not crash the server.

## 13. Why Send Channels Closing ≠ Client Disconnect

When the write loop exits because:

```go
for msg := range c.send {
```

and the channel closes, it means:
- The server decided to stop sending messages

It does not necessarily mean:
- The client disconnected
- The network died

So the server must still perform a graceful shutdown.

## 14. Key Invariants (Rules to Remember)

- Read and write loops are infinite by design
- Connections are removed only on failure
- Either loop may initiate cleanup
- Only one writer per connection
- Close frames are best-effort, not guaranteed
- Cleanup must be safe and idempotent

## 15. Final One-Sentence Summary

A WebSocket connection lives forever until a read or write fails; at that point, whichever side notices first performs cleanup, notifies the peer if possible, and removes the connection from the manager.
