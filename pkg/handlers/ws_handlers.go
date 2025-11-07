package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/hub"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, specify allowed origins
		return true
	},
}

func HandleWS(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from query parameters
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Token required", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := auth.ValidateJWT(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Upgrade to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// Create client
		client := &hub.Client{
			Hub:         h,
			UserID:      claims.UserID,
			SessionID:   claims.SessionID,
			Conn:        conn,
			Send:        make(chan []byte, 256),
			ActiveChats: make(map[string]bool),
		}

		// Register client
		h.Register <- client

		// Start read/write pumps
		go client.WritePump()
		go client.ReadPump()

		log.Printf("WebSocket connection established: user=%s, session=%s",
			claims.UserID, claims.SessionID)
	}
}
