package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"whatsdown/internal/models"

	"github.com/gorilla/websocket"
)

// SessionStore manages HTTP sessions
type SessionStore struct {
	sessions map[string]*models.Session
	mu       sync.RWMutex
}

var sessionStore = &SessionStore{
	sessions: make(map[string]*models.Session),
}

// CreateSession creates a new session for a username
func (s *SessionStore) CreateSession(username string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := generateSessionID()
	s.sessions[sessionID] = &models.Session{
		Username:  username,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return sessionID
}

// GetSession retrieves a session by ID
func (s *SessionStore) GetSession(sessionID string) (*models.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, sessionID)
		return nil, false
	}

	return session, true
}

// DeleteSession removes a session
func (s *SessionStore) DeleteSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// DeleteSessionByUsername removes all sessions for a username
func (s *SessionStore) DeleteSessionByUsername(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, session := range s.sessions {
		if session.Username == username {
			delete(s.sessions, id)
		}
	}
}

func generateSessionID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

// HTTPHandlers contains HTTP route handlers
type HTTPHandlers struct {
	Hub *Hub
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

// UserResponse represents a user in search results
type UserResponse struct {
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

// HandleLogin handles POST /api/login
func (h *HTTPHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate username
	username := strings.TrimSpace(req.Username)
	if len(username) == 0 || len(username) > 50 {
		http.Error(w, "Username must be between 1 and 50 characters", http.StatusBadRequest)
		return
	}

	// Check if username contains only alphanumeric and underscores
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
			(char >= '0' && char <= '9') || char == '_') {
			http.Error(w, "Username can only contain letters, numbers, and underscores", http.StatusBadRequest)
			return
		}
	}

	// Check if user already has an active connection
	h.Hub.mu.RLock()
	if user, exists := h.Hub.Users[username]; exists && user.CurrentConn != nil {
		h.Hub.mu.RUnlock()
		http.Error(w, "User already logged in from another device", http.StatusConflict)
		return
	}
	h.Hub.mu.RUnlock()

	// Create session
	sessionID := sessionStore.CreateSession(username)

	// Set cookie
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	}
	http.SetCookie(w, cookie)

	// Return response
	resp := LoginResponse{
		Username: username,
		Online:   false, // Will be true once WebSocket connects
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleLogout handles POST /api/logout
func (h *HTTPHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionIDFromRequest(r)
	if sessionID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	session, exists := sessionStore.GetSession(sessionID)
	if !exists {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	// Close WebSocket connection if exists
	h.Hub.mu.Lock()
	if client, exists := h.Hub.Clients[session.Username]; exists {
		h.Hub.mu.Unlock()
		h.Hub.Unregister <- client
	} else {
		h.Hub.mu.Unlock()
	}

	// Delete session
	sessionStore.DeleteSession(sessionID)

	// Clear cookie
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusOK)
}

// HandleMe handles GET /api/me
func (h *HTTPHandlers) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionIDFromRequest(r)
	if sessionID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	session, exists := sessionStore.GetSession(sessionID)
	if !exists {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	h.Hub.mu.RLock()
	online := false
	if user, exists := h.Hub.Users[session.Username]; exists {
		online = user.Online
	}
	h.Hub.mu.RUnlock()

	resp := LoginResponse{
		Username: session.Username,
		Online:   online,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleSearchUsers handles GET /api/users?search=<query>
func (h *HTTPHandlers) HandleSearchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user from session
	sessionID := getSessionIDFromRequest(r)
	if sessionID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	session, exists := sessionStore.GetSession(sessionID)
	if !exists {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("search")
	users := h.Hub.SearchUsers(query, session.Username)

	userResponses := make([]UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = UserResponse{
			Username: user.Username,
			Online:   user.Online,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userResponses)
}

// HandleGetConversations handles GET /api/conversations
func (h *HTTPHandlers) HandleGetConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionIDFromRequest(r)
	if sessionID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	session, exists := sessionStore.GetSession(sessionID)
	if !exists {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	conversations := h.Hub.GetConversations(session.Username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)
}

// HandleGetConversation handles GET /api/conversations/{peerUsername}
func (h *HTTPHandlers) HandleGetConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := getSessionIDFromRequest(r)
	if sessionID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	session, exists := sessionStore.GetSession(sessionID)
	if !exists {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	// Extract peer username from path
	path := strings.TrimPrefix(r.URL.Path, "/api/conversations/")
	peerUsername := strings.TrimSpace(path)

	if peerUsername == "" {
		http.Error(w, "Peer username required", http.StatusBadRequest)
		return
	}

	messages := h.Hub.GetConversationMessages(session.Username, peerUsername)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// getSessionIDFromRequest extracts session ID from cookie
func getSessionIDFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// requireAuth is a middleware to check authentication
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSessionIDFromRequest(r)
		if sessionID == "" {
			http.Error(w, "Not authenticated", http.StatusUnauthorized)
			return
		}

		_, exists := sessionStore.GetSession(sessionID)
		if !exists {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// HandleWebSocket handles WebSocket connections
func (h *HTTPHandlers) HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Get session
	sessionID := getSessionIDFromRequest(r)
	if sessionID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	session, exists := sessionStore.GetSession(sessionID)
	if !exists {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	username := session.Username

	// Check if user already has an active connection
	hub.mu.RLock()
	if user, exists := hub.Users[username]; exists && user.CurrentConn != nil {
		hub.mu.RUnlock()
		http.Error(w, "User already has an active connection", http.StatusConflict)
		return
	}
	hub.mu.RUnlock()

	// Upgrade connection
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for demo
		},
		EnableCompression: false, // Disable compression to avoid issues
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	
	log.Printf("WebSocket upgraded successfully for user: %s", username)

	// Create client
	client := &Client{
		Username: username,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}

	// Register client (non-blocking)
	select {
	case hub.Register <- client:
		// Registration will happen in hub's Run() goroutine
		// Start goroutines immediately - they will handle the connection
		// Note: If registration fails, the connection will be cleaned up
		go client.writePump()
		go client.readPump()
	default:
		// Hub is busy, close connection
		log.Printf("Failed to register client: hub register channel full")
		conn.Close()
	}
}

