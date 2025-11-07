package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/models"
	"github.com/msniranjan18/chit-chat/pkg/store"
)

type UserHandler struct {
	store  *store.Store
	logger *slog.Logger
}

func NewUserHandler(store *store.Store, logger *slog.Logger) *UserHandler {
	return &UserHandler{store: store, logger: logger}
}

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("GetCurrentUser: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("GetCurrentUser: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.Debug("GetCurrentUser: fetching user", "user_id", userID)

	user, err := h.store.GetUserByID(userID)
	if err != nil {
		h.logger.Error("GetCurrentUser: failed to get user", "error", err, "user_id", userID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if user == nil {
		h.logger.Warn("GetCurrentUser: user not found", "user_id", userID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	h.logger.Debug("GetCurrentUser: retrieved user", "user_id", userID, "name", user.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		h.logger.Warn("UpdateUser: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("UpdateUser: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.Info("UpdateUser: updating user", "user_id", userID)

	var req models.UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("UpdateUser: invalid request body", "user_id", userID, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Debug("UpdateUser: update request", "user_id", userID, "request", req)

	// Update user
	if err := h.store.UpdateUser(userID, &req); err != nil {
		h.logger.Error("UpdateUser: failed to update user", "error", err, "user_id", userID)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	// Get updated user
	user, err := h.store.GetUserByID(userID)
	if err != nil {
		h.logger.Error("UpdateUser: failed to get updated user",
			"error", err, "user_id", userID)
		http.Error(w, "Failed to get updated user", http.StatusInternalServerError)
		return
	}

	h.logger.Info("UpdateUser: user updated successfully", "user_id", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("SearchUsers: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("SearchUsers: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		h.logger.Warn("SearchUsers: missing search query", "user_id", userID)
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	h.logger.Info("SearchUsers: searching users", "user_id", userID, "query", query)

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	h.logger.Debug("SearchUsers: search parameters", "user_id", userID, "query", query, "limit", limit)

	// Search users
	users, err := h.store.SearchUsers(query, limit)
	if err != nil {
		h.logger.Error("SearchUsers: failed to search users",
			"error", err, "user_id", userID, "query", query)
		http.Error(w, "Failed to search users", http.StatusInternalServerError)
		return
	}

	// Filter out current user
	var filteredUsers []models.User
	for _, user := range users {
		if user.ID != userID {
			filteredUsers = append(filteredUsers, user)
		} else {
			h.logger.Debug("SearchUsers: filtered out current user", "user_id", userID)
		}
	}

	h.logger.Debug("SearchUsers: search completed",
		"user_id", userID, "query", query, "found", len(users), "filtered", len(filteredUsers))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredUsers)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("GetUser: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("GetUser: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetUserID := r.PathValue("id")
	if targetUserID == "" {
		h.logger.Warn("GetUser: missing target user ID", "requester_id", userID)
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	h.logger.Debug("GetUser: fetching user info", "requester_id", userID, "target_user_id", targetUserID)

	// Get user
	user, err := h.store.GetUserByID(targetUserID)
	if err != nil {
		h.logger.Error("GetUser: failed to get user",
			"error", err, "requester_id", userID, "target_user_id", targetUserID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if user == nil {
		h.logger.Warn("GetUser: user not found",
			"requester_id", userID, "target_user_id", targetUserID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	h.logger.Debug("GetUser: retrieved user",
		"requester_id", userID, "target_user_id", targetUserID, "name", user.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("GetContacts: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("GetContacts: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.Debug("GetContacts: fetching contacts", "user_id", userID)

	// Get contacts
	contacts, err := h.store.GetContacts(userID)
	if err != nil {
		h.logger.Error("GetContacts: failed to get contacts",
			"error", err, "user_id", userID)
		http.Error(w, "Failed to get contacts", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("GetContacts: retrieved contacts", "user_id", userID, "contact_count", len(contacts))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}

func (h *UserHandler) AddContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("AddContact: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("AddContact: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.Info("AddContact: adding contact", "requester_id", userID)

	var req struct {
		UserID      string  `json:"user_id"`
		DisplayName *string `json:"display_name,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("AddContact: invalid request body", "requester_id", userID, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		h.logger.Warn("AddContact: missing target user ID", "requester_id", userID)
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	h.logger.Debug("AddContact: contact request",
		"requester_id", userID, "target_user_id", req.UserID, "display_name", req.DisplayName)

	// Check if target user exists
	targetUser, err := h.store.GetUserByID(req.UserID)
	if err != nil {
		h.logger.Error("AddContact: failed to check target user",
			"error", err, "requester_id", userID, "target_user_id", req.UserID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if targetUser == nil {
		h.logger.Warn("AddContact: target user not found",
			"requester_id", userID, "target_user_id", req.UserID)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Add contact
	displayName := ""
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}

	if err := h.store.AddContact(userID, req.UserID, displayName); err != nil {
		h.logger.Error("AddContact: failed to add contact",
			"error", err, "requester_id", userID, "target_user_id", req.UserID)
		http.Error(w, "Failed to add contact", http.StatusInternalServerError)
		return
	}

	h.logger.Info("AddContact: contact added successfully",
		"requester_id", userID, "target_user_id", req.UserID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Contact added successfully",
	})
}

func (h *UserHandler) RemoveContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.logger.Warn("RemoveContact: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("RemoveContact: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	contactID := r.PathValue("id")
	if contactID == "" {
		h.logger.Warn("RemoveContact: missing contact ID", "user_id", userID)
		http.Error(w, "Contact ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("RemoveContact: removing contact", "user_id", userID, "contact_id", contactID)

	// Remove contact
	if err := h.store.RemoveContact(userID, contactID); err != nil {
		h.logger.Error("RemoveContact: failed to remove contact",
			"error", err, "user_id", userID, "contact_id", contactID)
		http.Error(w, "Failed to remove contact", http.StatusInternalServerError)
		return
	}

	h.logger.Info("RemoveContact: contact removed successfully",
		"user_id", userID, "contact_id", contactID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) GetOnlineUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("GetOnlineUsers: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("GetOnlineUsers: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.Debug("GetOnlineUsers: fetching online users", "requester_id", userID)

	// Get online users
	onlineUserIDs, err := h.store.GetOnlineUsers()
	if err != nil {
		h.logger.Error("GetOnlineUsers: failed to get online users", "error", err, "requester_id", userID)
		http.Error(w, "Failed to get online users", http.StatusInternalServerError)
		return
	}

	// Get user details
	users, err := h.store.GetUsersByIDs(onlineUserIDs)
	if err != nil {
		h.logger.Error("GetOnlineUsers: failed to get user details",
			"error", err, "requester_id", userID, "online_count", len(onlineUserIDs))
		http.Error(w, "Failed to get user details", http.StatusInternalServerError)
		return
	}

	// Mark as online
	for i := range users {
		users[i].IsOnline = true
	}

	h.logger.Debug("GetOnlineUsers: retrieved online users",
		"requester_id", userID, "online_count", len(users))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *UserHandler) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("GetUserSessions: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("GetUserSessions: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.Debug("GetUserSessions: fetching user sessions", "user_id", userID)

	// Get sessions (simplified - using in-memory store)
	// In production, implement in store
	sessions := []models.UserSession{
		{
			UserID:     userID,
			SessionID:  "current",
			DeviceInfo: "Web Browser",
			LastActive: time.Now(),
			CreatedAt:  time.Now(),
			IsActive:   true,
		},
	}

	h.logger.Debug("GetUserSessions: returning sessions", "user_id", userID, "session_count", len(sessions))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}
