package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/models"
	"github.com/msniranjan18/chit-chat/pkg/store"
)

type AuthHandler struct {
	store store.Store
}

func NewAuthHandler(store store.Store) *AuthHandler {
	return &AuthHandler{store: store}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate phone number (Indian format)
	req.Phone = strings.TrimSpace(req.Phone)
	if len(req.Phone) != 10 {
		http.Error(w, "Invalid phone number", http.StatusBadRequest)
		return
	}

	// Validate name
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Check if user exists
	existingUser, err := h.store.GetUserByPhone(req.Phone)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var user *models.User
	if existingUser != nil {
		// Existing user - update last seen
		user = existingUser
		h.store.UpdateUserLastSeen(user.ID, time.Now())
	} else {
		// New user - create
		user = &models.User{
			Phone:  req.Phone,
			Name:   req.Name,
			Status: "Hey there! I am using ChitChat",
		}
		if err := h.store.CreateUser(user); err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}

	// Create session
	sessionID := uuid.New().String()
	deviceInfo := r.UserAgent()
	ipAddress := getIPAddress(r)

	if err := h.store.CreateUserSession(user.ID, sessionID, deviceInfo, ipAddress); err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, expiresAt, err := auth.GenerateJWT(user.ID, sessionID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := models.AuthResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user by phone
	user, err := h.store.GetUserByPhone(req.Phone)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update last seen
	h.store.UpdateUserLastSeen(user.ID, time.Now())

	// Create session
	sessionID := uuid.New().String()
	deviceInfo := r.UserAgent()
	ipAddress := getIPAddress(r)

	if err := h.store.CreateUserSession(user.ID, sessionID, deviceInfo, ipAddress); err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, expiresAt, err := auth.GenerateJWT(user.ID, sessionID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := models.AuthResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session ID from context
	sessionID := auth.GetSessionID(r.Context())
	if sessionID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Delete session
	if err := h.store.DeleteSession(sessionID); err != nil {
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Get user
	user, err := h.store.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Refresh token
	newToken, expiresAt, err := auth.RefreshJWT(token)
	if err != nil {
		http.Error(w, "Failed to refresh token", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"token":      newToken,
		"expires_at": expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to get IP address
func getIPAddress(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	// Remove port if present
	if idx := strings.Index(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
