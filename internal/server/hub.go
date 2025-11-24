package server

import (
	"encoding/json"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"whatsdown/internal/models"

	"github.com/google/uuid"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	Clients map[string]*Client

	// Registered users
	Users map[string]*models.User

	// Conversations: key is conversation key (e.g., "user1|user2"), value is messages
	Conversations map[string][]*models.Message

	// Register requests from clients
	Register chan *Client

	// Unregister requests from clients
	Unregister chan *Client

	// Inbound messages from clients
	InboundMessages chan *models.InboundMessage

	// Typing events
	TypingEvents chan *TypingEventWrapper

	// Mutex for thread-safe access
	mu sync.RWMutex
}

// TypingEventWrapper wraps typing event with sender username
type TypingEventWrapper struct {
	From      string
	To        string
	IsTyping  bool
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		Clients:         make(map[string]*Client),
		Users:           make(map[string]*models.User),
		Conversations:   make(map[string][]*models.Message),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		InboundMessages: make(chan *models.InboundMessage, 256),
		TypingEvents:    make(chan *TypingEventWrapper, 256),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			if !h.registerClient(client) {
				// Registration failed, connection should be closed by registerClient
			}

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case msg := <-h.InboundMessages:
			// Messages are handled directly in client.readPump via handleInboundMessageWithSender
			// This channel is kept for potential future use
			_ = msg

		case event := <-h.TypingEvents:
			h.handleTypingEvent(event)
		}
	}
}

func (h *Hub) registerClient(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	username := client.Username

	// Check if user already has an active connection
	if user, exists := h.Users[username]; exists && user.CurrentConn != nil {
		// Reject new connection - user already connected
		log.Printf("User %s already has an active connection, closing old connection", username)
		// Close the old connection's Send channel to trigger cleanup
		if oldClient, exists := h.Clients[username]; exists {
			close(oldClient.Send)
			delete(h.Clients, username)
		}
		// Continue with new registration
	}

	// Register client
	h.Clients[username] = client

	// Create or update user
	if user, exists := h.Users[username]; exists {
		user.Online = true
		user.CurrentConn = client
		user.LastSeen = time.Now()
	} else {
		h.Users[username] = &models.User{
			Username:    username,
			Online:      true,
			CurrentConn: client,
			LastSeen:    time.Now(),
		}
	}

	// Broadcast online status to all other users
	h.broadcastStatus(username, true)

	// Send online status of all existing users to the newly connected client
	// This ensures the new client knows who's online
	for uname, user := range h.Users {
		if uname != username && user.Online {
			statusEvent := &models.StatusEvent{
				Username: uname,
				Online:   true,
			}
			h.sendToClient(client, "status", statusEvent)
		}
	}

	log.Printf("Client registered: %s", username)
	return true
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	username := client.Username

	// Only unregister if this is still the active client
	if existingClient, exists := h.Clients[username]; exists && existingClient == client {
		delete(h.Clients, username)

		if user, exists := h.Users[username]; exists {
			user.Online = false
			user.CurrentConn = nil
			user.LastSeen = time.Now()
		}

		// Close Send channel safely (check if already closed)
		select {
		case <-client.Send:
			// Channel already closed or has messages, don't close again
		default:
			close(client.Send)
		}

		// Broadcast offline status
		h.broadcastStatus(username, false)

		log.Printf("Client unregistered: %s", username)
	} else {
		log.Printf("Client %s already replaced, skipping unregister", username)
	}
}

func (h *Hub) handleInboundMessageWithSender(from string, msg *models.InboundMessage) {
	h.mu.Lock()

	// Create message
	message := &models.Message{
		ID:        uuid.New().String(),
		From:      from,
		To:        msg.To,
		Content:   msg.Content,
		Timestamp: time.Now(),
		Status:    "sent",
	}

	// Store in conversation
	convKey := models.ConvKey(from, msg.To)
	h.Conversations[convKey] = append(h.Conversations[convKey], message)

	// Create outbound message for sender
	senderOutboundMsg := &models.OutboundMessage{
		ID:        message.ID,
		From:      message.From,
		To:        message.To,
		Content:   message.Content,
		Timestamp: message.Timestamp.Format(time.RFC3339),
		Status:    message.Status,
	}

	// Get clients while holding lock
	var senderClient *Client
	var recipientClient *Client
	var senderExists bool
	var recipientExists bool
	
	if client, exists := h.Clients[from]; exists {
		senderClient = client
		senderExists = true
	}
	if client, exists := h.Clients[msg.To]; exists {
		recipientClient = client
		recipientExists = true
	}

	h.mu.Unlock()

	// Send to sender (confirmation) - without lock
	if senderExists && senderClient != nil {
		log.Printf("Sending message to sender %s: %s -> %s", from, message.Content, msg.To)
		h.sendToClient(senderClient, "message", senderOutboundMsg)
	} else {
		log.Printf("Sender %s not found or not connected", from)
	}

	// Send to recipient if online - without lock
	if recipientExists && recipientClient != nil {
		// Create separate outbound message for recipient
		recipientOutboundMsg := &models.OutboundMessage{
			ID:        message.ID,
			From:      message.From,
			To:        message.To,
			Content:   message.Content,
			Timestamp: message.Timestamp.Format(time.RFC3339),
			Status:    "delivered",
		}
		log.Printf("Sending message to recipient %s: %s -> %s", msg.To, message.Content, from)
		h.sendToClient(recipientClient, "message", recipientOutboundMsg)
		
		// Mark as delivered in storage
		h.mu.Lock()
		message.Status = "delivered"
		h.mu.Unlock()

		// Send ack to sender
		if senderExists && senderClient != nil {
			ack := &models.AckEvent{
				MessageID: message.ID,
				Status:    "delivered",
			}
			h.sendToClient(senderClient, "ack", ack)
		}
	}
}

func (h *Hub) handleTypingEvent(event *TypingEventWrapper) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Send typing event to recipient
	if recipientClient, exists := h.Clients[event.To]; exists {
		typingEvent := &models.TypingEvent{
			From:     event.From,
			IsTyping: event.IsTyping,
		}
		log.Printf("Sending typing event: %s -> %s (typing: %v)", event.From, event.To, event.IsTyping)
		h.sendToClient(recipientClient, "typing", typingEvent)
	} else {
		log.Printf("Recipient %s not found for typing event from %s", event.To, event.From)
	}
}

func (h *Hub) broadcastStatus(username string, online bool) {
	statusEvent := &models.StatusEvent{
		Username: username,
		Online:   online,
	}

	// Broadcast to all connected clients except the user themselves
	for uname, client := range h.Clients {
		if uname != username {
			h.sendToClient(client, "status", statusEvent)
		}
	}
}

func (h *Hub) sendToClient(client *Client, msgType string, payload interface{}) {
	wsMsg := &models.WSMessage{
		Type:    msgType,
		Payload: payload,
	}

	data, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	select {
	case client.Send <- data:
		log.Printf("Message queued for client %s, type: %s", client.Username, msgType)
	default:
		log.Printf("Client %s send channel full, closing connection", client.Username)
		close(client.Send)
		delete(h.Clients, client.Username)
	}
}

// GetConversations returns all conversations for a user
func (h *Hub) GetConversations(username string) []*models.Conversation {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conversations := []*models.Conversation{}
	seenPeers := make(map[string]bool)

	for _, messages := range h.Conversations {
		if len(messages) == 0 {
			continue
		}

		// Check if this conversation involves the user
		lastMsg := messages[len(messages)-1]
		var peer string
		if lastMsg.From == username {
			peer = lastMsg.To
		} else if lastMsg.To == username {
			peer = lastMsg.From
		} else {
			continue
		}

		if seenPeers[peer] {
			continue
		}
		seenPeers[peer] = true

		peerOnline := false
		if user, exists := h.Users[peer]; exists {
			peerOnline = user.Online
		}

		conversations = append(conversations, &models.Conversation{
			PeerUsername:      peer,
			LastMessagePreview: lastMsg.Content,
			LastMessageTime:   lastMsg.Timestamp,
			PeerOnline:        peerOnline,
		})
	}

	// Sort conversations by last message time (most recent first)
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].LastMessageTime.After(conversations[j].LastMessageTime)
	})

	return conversations
}

// GetConversationMessages returns all messages for a conversation between two users
func (h *Hub) GetConversationMessages(username1, username2 string) []*models.Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	convKey := models.ConvKey(username1, username2)
	return h.Conversations[convKey]
}

// SearchUsers returns users matching the search query
func (h *Hub) SearchUsers(query string, excludeUsername string) []*models.User {
	h.mu.RLock()
	defer h.mu.RUnlock()

	results := []*models.User{}
	queryLower := query

	for _, user := range h.Users {
		if user.Username == excludeUsername {
			continue
		}
		// Simple substring search (case-insensitive)
		if len(queryLower) == 0 || contains(user.Username, queryLower) {
			results = append(results, &models.User{
				Username: user.Username,
				Online:   user.Online,
			})
		}
	}

	return results
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}

