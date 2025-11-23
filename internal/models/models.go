package models

import (
	"sort"
	"strings"
	"time"
)

// User represents a user in the system
type User struct {
	Username    string
	Online      bool
	CurrentConn interface{} // *Client from server package
	LastSeen    time.Time
}

// Message represents a chat message
type Message struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // "sent", "delivered"
}

// Session represents an HTTP session
type Session struct {
	Username  string
	ExpiresAt time.Time
}

// Conversation represents a conversation between two users
type Conversation struct {
	PeerUsername      string    `json:"peerUsername"`
	LastMessagePreview string   `json:"lastMessagePreview"`
	LastMessageTime   time.Time `json:"lastMessageTime"`
	PeerOnline        bool      `json:"peerOnline"`
	UnreadCount       int       `json:"unreadCount"`
}

// WSMessage represents a WebSocket message envelope
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// InboundMessage represents a message from client to server
type InboundMessage struct {
	To      string `json:"to"`
	Content string `json:"content"`
	TempID  string `json:"tempId,omitempty"`
}

// OutboundMessage represents a message from server to client
type OutboundMessage struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
}

// TypingEvent represents a typing indicator event
type TypingEvent struct {
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	IsTyping bool   `json:"isTyping"`
}

// StatusEvent represents an online/offline status event
type StatusEvent struct {
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

// AckEvent represents a message acknowledgment
type AckEvent struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}

// ConvKey generates a normalized conversation key for two users
func ConvKey(a, b string) string {
	users := []string{a, b}
	sort.Strings(users)
	return strings.Join(users, "|")
}

