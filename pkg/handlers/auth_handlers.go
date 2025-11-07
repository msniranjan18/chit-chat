package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/msniranjan18/common/jwt"
	"github.com/msniranjan18/common/middleware/auth"

	"github.com/msniranjan18/chit-chat/pkg/models"
	"github.com/msniranjan18/chit-chat/pkg/store"
)

type AuthHandler struct {
	store  *store.Store
	logger *slog.Logger
}

func NewAuthHandler(store *store.Store, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{store: store, logger: logger}
}

// Register godoc
// @Summary      Register or login a user
// @Description  Creates a new user profile or logs in an existing user based on their phone number. Returns a JWT token and user details.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      models.AuthRequest  true  "Registration/Login Details"
// @Success      200     {object}  models.AuthResponse "Successful login for existing user"
// @Success      201     {object}  models.AuthResponse "Successful registration for new user"
// @Failure      400     {object}  map[string]string "Invalid request body or phone number"
// @Failure      500     {object}  map[string]string "Internal server error"
// @Router       /api/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("Register: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Register: invalid request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("Register: processing registration", "phone", req.Phone, "name", req.Name)

	// Validate phone number (Indian format)
	req.Phone = strings.TrimSpace(req.Phone)
	if len(req.Phone) != 10 {
		h.logger.Warn("Register: invalid phone number length", "phone", req.Phone, "length", len(req.Phone))
		http.Error(w, "Invalid phone number", http.StatusBadRequest)
		return
	}

	// Validate name
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		h.logger.Warn("Register: missing name")
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Check if user exists
	existingUser, err := h.store.GetUserByPhone(req.Phone)
	if err != nil && err != sql.ErrNoRows {
		h.logger.Error("Register: failed to check existing user", "error", err, "phone", req.Phone)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var user *models.User
	if existingUser != nil {
		h.logger.Info("Register: existing user found", "user_id", existingUser.ID, "phone", req.Phone)
		// Existing user - update last seen
		user = existingUser
		h.store.UpdateUserLastSeen(user.ID, time.Now())
		h.logger.Debug("Register: updated last seen for existing user", "user_id", user.ID)
	} else {
		// New user - create
		user = &models.User{
			Phone:  req.Phone,
			Name:   req.Name,
			Status: "Hey there! I am using ChitChat",
		}
		if err := h.store.CreateUser(user); err != nil {
			h.logger.Error("Register: failed to create user", "error", err, "phone", req.Phone, "name", req.Name)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
		h.logger.Info("Register: new user created", "user_id", user.ID, "phone", req.Phone, "name", req.Name)
	}

	// Create session
	sessionID := uuid.New().String()
	deviceInfo := r.UserAgent()
	ipAddress := getIPAddress(r)

	h.logger.Debug("Register: creating user session",
		"user_id", user.ID, "session_id", sessionID, "device", deviceInfo, "ip", ipAddress)

	if err := h.store.CreateUserSession(user.ID, sessionID, deviceInfo, ipAddress); err != nil {
		h.logger.Error("Register: failed to create session",
			"error", err, "user_id", user.ID, "session_id", sessionID)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, expiresAt, err := jwt.GenerateJWT(user.ID, sessionID)
	if err != nil {
		h.logger.Error("Register: failed to generate JWT", "error", err, "user_id", user.ID)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Register: successful",
		"user_id", user.ID, "session_id", sessionID, "expires_at", expiresAt)

	// Prepare response
	response := models.AuthResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Login godoc
// @Summary      Login user
// @Description  Authenticates a user by phone number and returns a new session token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      models.AuthRequest  true  "Login Details"
// @Success      200     {object}  models.AuthResponse
// @Failure      404     {object}  map[string]string "User not found"
// @Failure      500     {object}  map[string]string "Internal server error"
// @Router       /api/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("Login: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Login: invalid request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("Login: processing login", "phone", req.Phone)

	// Get user by phone
	user, err := h.store.GetUserByPhone(req.Phone)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Warn("Login: user not found", "phone", req.Phone)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Login: failed to get user by phone", "error", err, "phone", req.Phone)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Login: user found", "user_id", user.ID, "name", user.Name)

	// Update last seen
	h.store.UpdateUserLastSeen(user.ID, time.Now())

	// Create session
	sessionID := uuid.New().String()
	deviceInfo := r.UserAgent()
	ipAddress := getIPAddress(r)

	h.logger.Debug("Login: creating user session",
		"user_id", user.ID, "session_id", sessionID, "device", deviceInfo, "ip", ipAddress)

	if err := h.store.CreateUserSession(user.ID, sessionID, deviceInfo, ipAddress); err != nil {
		h.logger.Error("Login: failed to create session",
			"error", err, "user_id", user.ID, "session_id", sessionID)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, expiresAt, err := jwt.GenerateJWT(user.ID, sessionID)
	if err != nil {
		h.logger.Error("Login: failed to generate JWT", "error", err, "user_id", user.ID)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Login: successful",
		"user_id", user.ID, "session_id", sessionID, "expires_at", expiresAt)

	// Prepare response
	response := models.AuthResponse{
		Token:     token,
		User:      *user,
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Logout godoc
// @Summary      Logout user
// @Description  Deletes the current user session and invalidates the session ID.
// @Tags         auth
// @Success      200     {object}  map[string]string "Logged out successfully"
// @Failure      401     {object}  map[string]string "Not authenticated"
// @Router       /api/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session ID from context
	sessionID := auth.GetSessionID(r.Context())
	if sessionID == "" {
		h.logger.Warn("Logout: no session ID in context")
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	h.logger.Info("Logout: processing logout", "session_id", sessionID)

	// Delete session
	if err := h.store.DeleteSession(sessionID); err != nil {
		h.logger.Error("Logout: failed to delete session", "error", err, "session_id", sessionID)
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Logout: successful", "session_id", sessionID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

// Verify godoc
// @Summary      Verify authentication
// @Description  Checks if the current JWT token is valid and returns the authenticated user's profile.
// @Tags         auth
// @Produce      json
// @Success      200     {object}  models.User
// @Failure      401     {object}  map[string]string "Not authenticated"
// @Router       /api/auth/verify [get]
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("Verify: no user ID in context")
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	h.logger.Debug("Verify: verifying user", "user_id", userID)

	// Get user
	user, err := h.store.GetUserByID(userID)
	if err != nil {
		h.logger.Error("Verify: failed to get user", "error", err, "user_id", userID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	h.logger.Debug("Verify: user verified", "user_id", userID, "name", user.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// RefreshToken godoc
// @Summary      Refresh session token
// @Description  Takes an existing valid JWT and returns a new token with an extended expiration time.
// @Tags         auth
// @Produce      json
// @Param        Authorization  header    string  true  "Insert your Bearer token" default(Bearer <token>)
// @Success      200            {object}  map[string]interface{} "Returns new token and expires_at"
// @Failure      401            {object}  map[string]string "Invalid or missing token"
// @Router       /api/auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.logger.Warn("RefreshToken: missing authorization header")
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		h.logger.Warn("RefreshToken: empty token")
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	h.logger.Debug("RefreshToken: refreshing token")

	// Refresh token
	newToken, expiresAt, err := jwt.RefreshJWT(token)
	if err != nil {
		h.logger.Error("RefreshToken: failed to refresh token", "error", err)
		http.Error(w, "Failed to refresh token", http.StatusUnauthorized)
		return
	}

	h.logger.Info("RefreshToken: token refreshed successfully", "expires_at", expiresAt)

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
