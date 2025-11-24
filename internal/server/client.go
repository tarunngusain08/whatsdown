package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"whatsdown/internal/models"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demo
	},
}

// Client represents a WebSocket client connection
type Client struct {
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error for %s: %v", c.Username, err)
			}
			break
		}

		// Parse WebSocket message
		var wsMsg models.WSMessage
		if err := json.Unmarshal(messageBytes, &wsMsg); err != nil {
			log.Printf("Error unmarshaling WebSocket message: %v", err)
			continue
		}

		// Handle different message types
		switch wsMsg.Type {
		case "message":
			var inboundMsg models.InboundMessage
			payloadBytes, _ := json.Marshal(wsMsg.Payload)
			if err := json.Unmarshal(payloadBytes, &inboundMsg); err != nil {
				log.Printf("Error unmarshaling message payload: %v", err)
				continue
			}
			c.Hub.handleInboundMessageWithSender(c.Username, &inboundMsg)

		case "typing":
			var typingEvent models.TypingEvent
			payloadBytes, _ := json.Marshal(wsMsg.Payload)
			if err := json.Unmarshal(payloadBytes, &typingEvent); err != nil {
				log.Printf("Error unmarshaling typing payload: %v", err)
				continue
			}
			c.Hub.TypingEvents <- &TypingEventWrapper{
				From:     c.Username,
				To:       typingEvent.To,
				IsTyping: typingEvent.IsTyping,
			}
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write the message as a separate WebSocket frame
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error for %s: %v", c.Username, err)
				return
			}

			// Write any queued messages as separate frames
			n := len(c.Send)
			for i := 0; i < n; i++ {
				queuedMsg := <-c.Send
				if err := c.Conn.WriteMessage(websocket.TextMessage, queuedMsg); err != nil {
					log.Printf("WebSocket write queued message error for %s: %v", c.Username, err)
					return
				}
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("WebSocket ping error for %s: %v", c.Username, err)
				return
			}
		}
	}
}

