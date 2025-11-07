package routes

import (
	"log/slog"
	"net/http"

	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/handlers"
	"github.com/msniranjan18/chit-chat/pkg/hub"
	"github.com/msniranjan18/chit-chat/pkg/store"

	_ "github.com/swaggo/files"
	httpSwagger "github.com/swaggo/http-swagger"
)

// RouterHandler wraps the router with the logger for dependency injection
type RouterHandler struct {
	router *http.ServeMux
	logger *slog.Logger
}

// NewRouter creates a new HTTP router with all routes configured
func NewRouter(h *hub.Hub, s *store.Store, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	// Create handlers with logger
	authHandler := handlers.NewAuthHandler(s, logger)
	userHandler := handlers.NewUserHandler(s, logger)
	chatHandler := handlers.NewChatHandler(s, logger)
	messageHandler := handlers.NewMessageHandler(s, logger)
	wsHandler := handlers.NewWSHandler(h, logger)

	// Static files
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	logger.Debug("Static file server configured", "path", "/static/")

	// Swagger UI
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
	logger.Debug("Swagger UI configured", "path", "/swagger/")

	// WebSocket endpoint
	mux.HandleFunc("/ws", wsHandler.HandleWS)
	logger.Debug("WebSocket endpoint configured", "path", "/ws")

	// Authentication endpoints (no auth required)
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/refresh", authHandler.RefreshToken)
	logger.Debug("Public authentication endpoints configured",
		"endpoints", []string{"/api/auth/register", "/api/auth/login", "/api/auth/refresh"})

	// API endpoints with authentication middleware
	apiRouter := http.NewServeMux()

	// Auth endpoints (require auth)
	apiRouter.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	apiRouter.HandleFunc("GET /api/auth/verify", authHandler.Verify)

	// User endpoints
	apiRouter.HandleFunc("GET /api/users/me", userHandler.GetCurrentUser)
	apiRouter.HandleFunc("PUT /api/users/me", userHandler.UpdateUser)
	apiRouter.HandleFunc("PATCH /api/users/me", userHandler.UpdateUser)
	apiRouter.HandleFunc("GET /api/users/search", userHandler.SearchUsers)
	apiRouter.HandleFunc("GET /api/users/{id}", userHandler.GetUser)
	apiRouter.HandleFunc("GET /api/users/online", userHandler.GetOnlineUsers)
	apiRouter.HandleFunc("GET /api/users/sessions", userHandler.GetUserSessions)

	// Contact endpoints
	apiRouter.HandleFunc("GET /api/contacts", userHandler.GetContacts)
	apiRouter.HandleFunc("POST /api/contacts", userHandler.AddContact)
	apiRouter.HandleFunc("DELETE /api/contacts/{id}", userHandler.RemoveContact)

	// Chat endpoints
	apiRouter.HandleFunc("GET /api/chats", chatHandler.GetChats)
	apiRouter.HandleFunc("POST /api/chats", chatHandler.CreateChat)
	apiRouter.HandleFunc("GET /api/chats/search", chatHandler.SearchChats)
	apiRouter.HandleFunc("GET /api/chats/{id}", chatHandler.GetChat)
	apiRouter.HandleFunc("PUT /api/chats/{id}", chatHandler.UpdateChat)
	apiRouter.HandleFunc("PATCH /api/chats/{id}", chatHandler.UpdateChat)
	apiRouter.HandleFunc("DELETE /api/chats/{id}", chatHandler.DeleteChat)
	apiRouter.HandleFunc("GET /api/chats/{id}/members", chatHandler.GetChatMembers)
	apiRouter.HandleFunc("POST /api/chats/{id}/members", chatHandler.AddChatMember)
	apiRouter.HandleFunc("DELETE /api/chats/{id}/members/{memberId}", chatHandler.RemoveChatMember)
	apiRouter.HandleFunc("POST /api/chats/{id}/leave", chatHandler.LeaveChat)
	apiRouter.HandleFunc("POST /api/chats/{id}/read", chatHandler.MarkChatAsRead)

	// Message endpoints
	apiRouter.HandleFunc("GET /api/messages", messageHandler.GetMessages)
	apiRouter.HandleFunc("POST /api/messages", messageHandler.SendMessage)
	apiRouter.HandleFunc("GET /api/messages/search", messageHandler.SearchMessages)
	apiRouter.HandleFunc("PUT /api/messages/{id}", messageHandler.UpdateMessage)
	apiRouter.HandleFunc("PATCH /api/messages/{id}", messageHandler.UpdateMessage)
	apiRouter.HandleFunc("DELETE /api/messages/{id}", messageHandler.DeleteMessage)
	apiRouter.HandleFunc("POST /api/messages/status", messageHandler.UpdateMessageStatus)

	// Apply authentication middleware to API routes with logging
	authenticatedAPI := auth.AuthMiddleware(apiRouter)

	// Wrap the authenticated API with route logging
	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("API request received",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"content_type", r.Header.Get("Content-Type"))

		// Pass through to authenticated handler
		authenticatedAPI.ServeHTTP(w, r)
	}))

	logger.Info("API routes configured",
		"auth_endpoints", 2,
		"user_endpoints", 8,
		"contact_endpoints", 3,
		"chat_endpoints", 14,
		"message_endpoints", 7)

	// SPA catch-all route (must be last)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only serve index.html for non-API routes
		if r.URL.Path == "/" || !isAPIRoute(r.URL.Path) {
			logger.Debug("Serving SPA index", "path", r.URL.Path, "method", r.Method)
			http.ServeFile(w, r, "./static/index.html")
			return
		}

		// For undefined API routes, return 404 with logging
		logger.Warn("API route not found",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr)
		http.NotFound(w, r)
	})

	logger.Debug("SPA catch-all route configured", "serve_from", "./static/index.html")

	return mux
}

// Helper function to check if path is an API route
func isAPIRoute(path string) bool {
	return len(path) > 4 && path[:4] == "/api"
}
