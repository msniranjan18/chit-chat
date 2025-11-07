package routes

import (
	"net/http"

	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/handlers"
	"github.com/msniranjan18/chit-chat/pkg/hub"
	"github.com/msniranjan18/chit-chat/pkg/store"
)

func NewRouter(h *hub.Hub, s store.Store) *http.ServeMux {
	mux := http.NewServeMux()

	// Create handlers
	authHandler := handlers.NewAuthHandler(s)
	userHandler := handlers.NewUserHandler(s)
	chatHandler := handlers.NewChatHandler(s)
	messageHandler := handlers.NewMessageHandler(s)

	// Static files
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Authentication endpoints (no auth required)
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/refresh", authHandler.RefreshToken)

	// WebSocket endpoint
	mux.HandleFunc("/ws", handlers.HandleWS(h))

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

	// Apply authentication middleware to API routes
	mux.Handle("/api/", auth.AuthMiddleware(apiRouter))

	// SPA catch-all route (must be last)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only serve index.html for non-API routes
		if r.URL.Path == "/" || !isAPIRoute(r.URL.Path) {
			http.ServeFile(w, r, "./static/index.html")
			return
		}
		// For undefined API routes, return 404
		http.NotFound(w, r)
	})

	return mux
}

// Helper function to check if path is an API route
func isAPIRoute(path string) bool {
	return len(path) > 4 && path[:4] == "/api"
}
