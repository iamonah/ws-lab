package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type Type string

type Event struct {
	Type    Type            `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, c *Client) error

const (
	EventSendMessage    = "send_message"
	EventNewMessage     = "new_message"
	EventUserJoined     = "user_joined"
	EventUserLeft       = "user_left"
	EventTyping         = "typing"
	EventStopTyping     = "stop_typing"
	EventUserList       = "user_list"
	EventChangeChatroom = "change_chatroom"
)

type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
}

type NewMessageEvent struct {
	SendMessageEvent
	SentAt time.Time `json:"sent_at"`
}

func SendMessage(event Event, c *Client) error {
	chatEvent := SendMessageEvent{}
	if err := json.Unmarshal(event.Payload, &chatEvent); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	newMessage := NewMessageEvent{
		SendMessageEvent: chatEvent,
		SentAt:           time.Now(),
	}

	data, err := json.Marshal(newMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal new message event: %w", err)
	}

	outgoingEvent := Event{
		Type:    EventNewMessage,
		Payload: data,
	}
	for Client := range c.manager.clients {
		if Client.chatroom == c.chatroom {
			Client.egress <- outgoingEvent
		}
	}
	return nil
}

type ChangeRoomEvent struct {
	RoomName string `json:"room_name"`
}

func ChatRoomHandler(event Event, c *Client) error {
	changeRoomEvent := ChangeRoomEvent{}
	if err := json.Unmarshal(event.Payload, &changeRoomEvent); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Printf("client requested to change room to: %s", changeRoomEvent.RoomName)
	c.chatroom = changeRoomEvent.RoomName
	return nil
}
