package main

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager

	// egress is used to avoid concurrent writes on the websocket connection
	egress chan []byte
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		egress:     make(chan []byte, 256),
		manager:    manager,
	}
}

func (c *Client) ReadMessages() {
	// cleanup connection
	defer func() {
		c.manager.removeClient(c)
	}()

	for {
		messageType, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break

		}

		for wsclient := range c.manager.clients {
			fmt.Println("hihihihhihihihi")
			wsclient.egress <- payload
		}

		log.Printf("received message type: %d", messageType)
		log.Printf("received message: %s", string(payload))
	}
}

func (c *Client) WriteMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	for {
		select {
		case msg, ok := <-c.egress:
			fmt.Printf("writing message: %s\n", string(msg))	
			if !ok {
				// Channel closed meaning the client has disconnected
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Printf("connection closed: %v", err)
				}
				return
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("error writing message: %v", err)
				c.manager.removeClient(c)
				return
			}
		}
	}
}
