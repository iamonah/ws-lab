package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait      = 10 * time.Second
	pinggInterval = (9 * pongWait) / 10
)

type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager
	chatroom   string

	// egress is used to avoid concurrent writes on the websocket connection
	egress chan Event
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		egress:     make(chan Event),
		manager:    manager,
		chatroom:   "general",
	}
}

func (c *Client) ReadMessages() {
	// cleanup connection
	defer func() {
		close(c.egress)
		c.manager.removeClient(c)
	}()

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("error setting read deadline: %v", err)
		return
	}

	c.connection.SetReadLimit(512) //hard limit on message size to prevent abuse

	c.connection.SetPongHandler(func(string) error {
		c.connection.SetReadDeadline(time.Now().Add(pongWait)) //reseting the timer
		return nil
	})

	for {
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("error reading message: %v", err)
			}
			return

		}

		var request Event
		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("error unmarshaling event: %v", err)
			return
		}

		if err := c.manager.routeEventHandler(request, c); err != nil {
			log.Printf("error routing event: %v", err)
			return
		}
	}

}

func (c *Client) WriteMessages() {
	ticker := time.NewTicker(pinggInterval)
	defer func() {
		// c.manager.removeClient(c)
		ticker.Stop()
	}()

	for {
		select {
		case msg, ok := <-c.egress:
			if !ok {
				// Channel closed meaning the client has disconnected
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Printf("connection closed: %v", err)
				}
				return
			}
			bytes, err := json.Marshal(msg)
			if err != nil {
				log.Printf("error marshaling message: %v", err)
				return
			}
			// Write the message to the websocket connection
			if err := c.connection.WriteMessage(websocket.TextMessage, bytes); err != nil {
				log.Printf("error writing message: %v", err)
				c.manager.removeClient(c)
				return
			}
		case <-ticker.C:
			// Send a ping message to check if connection alive
			if err := c.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("error sending ping: %v", err)
				return
			}

		}
	}
}
