package main

import (
	"encoding/json"
	"fmt"
)

type Type string

type Event struct {
	Type    Type            `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, c *Client) error

const (
	EventSendMessage = "send_message"
)

func SendMessage(event Event, c *Client) error {
	fmt.Println(event)
	return nil
}
