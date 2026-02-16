package main

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type ClientList map[*Client]bool

type Client struct { 
	connection *websocket.Conn
	manager    *Manager

	// egress is used to avoid concurrent writes on the websocket connection
	egress chan Event
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		egress:     make(chan Event),
		manager:    manager,
	}
}

func (c *Client) ReadMessages() {
	// cleanup connection
	defer func() {
		close(c.egress)
		c.manager.removeClient(c)
	}()

	for {
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("error reading message: %v", err)
			}
			print("hello")
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
		}
	}
}
