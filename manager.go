package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
)

type Manager struct {
	clients ClientList
	sync.Mutex
	handers map[Type]EventHandler
}

func NewManager() *Manager {
	return &Manager{
		clients: make(ClientList),
		handers: make(map[Type]EventHandler),
	}
}

func (m *Manager) RegisterEventHandler(eventType Type, handler EventHandler) {
	m.handers[eventType] = handler
}

func (m *Manager) routeEventHandler(event Event, c *Client) error {
	if handler, ok := m.handers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			log.Printf("error handling event: %v", err)
			return err
		}
		return nil
	} else {
		return fmt.Errorf("no handler for event type: %s", event.Type)
	}
}

func (m *Manager) serverWebsocket(w http.ResponseWriter, r *http.Request) {
	log.Println("new connection")

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	client := NewClient(conn, m)
	m.addclient(client)

	//start 2 goroutines for read and the other write messages
	go client.ReadMessages()
	go client.WriteMessages()
}

func (m *Manager) addclient(client *Client) {
	m.Lock()
	defer m.Unlock()
	m.clients[client] = true
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		client.connection.Close()
		delete(m.clients, client)
	}
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	switch origin {
	case "": //dev mode
		return true
	case "http://localhost:8080":
		return true
	default:
		return false
	}
}
