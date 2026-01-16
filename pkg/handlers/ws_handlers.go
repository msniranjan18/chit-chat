package handlers

import (
	"log/slog"
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

type WSHandler struct {
	hub    *hub.Hub
	logger *slog.Logger
}

func NewWSHandler(hub *hub.Hub, logger *slog.Logger) *WSHandler {
	return &WSHandler{hub: hub, logger: logger}
}

func (h *WSHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	// Extract token from query parameters
	token := r.URL.Query().Get("token")
	if token == "" {
		h.logger.Warn("HandleWS: missing token in query")
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	h.logger.Debug("HandleWS: validating token")

	// Validate token
	claims, err := auth.ValidateJWT(token)
	if err != nil {
		h.logger.Warn("HandleWS: invalid token", "error", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	h.logger.Info("HandleWS: upgrading to WebSocket",
		"user_id", claims.UserID, "session_id", claims.SessionID)

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("HandleWS: WebSocket upgrade error",
			"error", err, "user_id", claims.UserID)
		return
	}

	// Create client
	client := &hub.Client{
		Hub:         h.hub,
		UserID:      claims.UserID,
		SessionID:   claims.SessionID,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		ActiveChats: make(map[string]bool),
	}

	h.logger.Debug("HandleWS: registering client",
		"user_id", claims.UserID, "session_id", claims.SessionID)

	// Register client
	h.hub.Register <- client

	// Start read/write pumps
	go client.WritePump()
	go client.ReadPump()

	h.logger.Info("HandleWS: WebSocket connection established",
		"user_id", claims.UserID, "session_id", claims.SessionID)
}
