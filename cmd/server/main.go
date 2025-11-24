package main

import (
	"embed"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"whatsdown/internal/server"
)

//go:embed web
var webFiles embed.FS

func main() {
	hub := server.NewHub()
	go hub.Run()

	handlers := &server.HTTPHandlers{Hub: hub}

	// API routes
	http.HandleFunc("/api/login", handlers.HandleLogin)
	http.HandleFunc("/api/logout", handlers.HandleLogout)
	http.HandleFunc("/api/me", handlers.HandleMe)
	http.HandleFunc("/api/users", handlers.HandleSearchUsers)
	http.HandleFunc("/api/conversations", handlers.HandleGetConversations)
	http.HandleFunc("/api/conversations/", handlers.HandleGetConversation)

	// WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleWebSocket(hub, w, r)
	})

	// Serve static files (SPA)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Remove leading slash for embed
		path = strings.TrimPrefix(path, "/")
		// For embed, files are stored with their path relative to embed directive
		// Since we embedded web/, the path in embed FS is web/path
		embedPath := filepath.Join("web", path)
		if path == "" || path == "/" {
			embedPath = "web/index.html"
		}

		data, err := webFiles.ReadFile(embedPath)
		if err != nil {
			// If file not found, serve index.html for SPA routing
			if embedPath != "web/index.html" {
				data, err = webFiles.ReadFile("web/index.html")
			}
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}

		// Set content type based on file extension
		if strings.HasSuffix(path, ".html") {
			w.Header().Set("Content-Type", "text/html")
		} else if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(path, ".json") {
			w.Header().Set("Content-Type", "application/json")
		}

		w.Write(data)
	})

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
