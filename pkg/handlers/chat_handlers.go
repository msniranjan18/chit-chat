package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/models"
	"github.com/msniranjan18/chit-chat/pkg/store"
)

type ChatHandler struct {
	store store.Store
}

func NewChatHandler(store store.Store) *ChatHandler {
	return &ChatHandler{store: store}
}

func (h *ChatHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's chats
	chats, err := h.store.GetUserChats(userID)
	if err != nil {
		http.Error(w, "Failed to get chats", http.StatusInternalServerError)
		return
	}

	response := models.ChatListResponse{
		Chats: chats,
		Total: len(chats),
		Page:  1,
		Limit: len(chats),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ChatHandler) GetChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract chat ID from URL
	chatID := r.PathValue("id")
	if chatID == "" {
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	// Verify user is a member of the chat
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// Get chat details
	chat, err := h.store.GetChat(chatID)
	if err != nil {
		http.Error(w, "Failed to get chat", http.StatusInternalServerError)
		return
	}

	if chat == nil {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// Get chat members
	members, err := h.store.GetChatMembers(chatID)
	if err != nil {
		http.Error(w, "Failed to get chat members", http.StatusInternalServerError)
		return
	}

	// Get member users
	var memberIDs []string
	for _, member := range members {
		memberIDs = append(memberIDs, member.UserID)
	}

	users, err := h.store.GetUsersByIDs(memberIDs)
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	response := models.ChatResponse{
		Chat:    *chat,
		Members: members,
		Users:   users,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Type == "" {
		http.Error(w, "Chat type is required", http.StatusBadRequest)
		return
	}

	if req.Type == models.ChatTypeGroup && (req.Name == nil || *req.Name == "") {
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	if len(req.UserIDs) == 0 && req.Type != models.ChatTypeDirect {
		http.Error(w, "At least one user is required", http.StatusBadRequest)
		return
	}

	// For direct chat, ensure exactly 2 users (creator + one other)
	if req.Type == models.ChatTypeDirect {
		if len(req.UserIDs) != 1 {
			http.Error(w, "Direct chat requires exactly one other user", http.StatusBadRequest)
			return
		}

		// Check if direct chat already exists
		existingChat, err := h.store.GetDirectChat(userID, req.UserIDs[0])
		if err != nil {
			http.Error(w, "Failed to check existing chat", http.StatusInternalServerError)
			return
		}

		if existingChat != nil {
			// Return existing chat
			response := models.ChatResponse{
				Chat: *existingChat,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Create chat
	chat, err := h.store.CreateChat(&req, userID)
	if err != nil {
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	// Get members for response
	members, err := h.store.GetChatMembers(chat.ID)
	if err != nil {
		http.Error(w, "Failed to get chat members", http.StatusInternalServerError)
		return
	}

	response := models.ChatResponse{
		Chat:    *chat,
		Members: members,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *ChatHandler) UpdateChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	// Verify user is an admin of the chat
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// TODO: Check if user has permission to update (admin/owner)

	var req models.ChatUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update chat
	if err := h.store.UpdateChat(chatID, &req); err != nil {
		http.Error(w, "Failed to update chat", http.StatusInternalServerError)
		return
	}

	// Get updated chat
	chat, err := h.store.GetChat(chatID)
	if err != nil {
		http.Error(w, "Failed to get updated chat", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	// Verify user is the creator or admin
	chat, err := h.store.GetChat(chatID)
	if err != nil || chat == nil {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// Only creator can delete (or admin in future)
	if chat.CreatedBy != userID {
		http.Error(w, "Only chat creator can delete", http.StatusForbidden)
		return
	}

	// Delete chat
	if err := h.store.DeleteChat(chatID); err != nil {
		http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChatHandler) GetChatMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	// Verify user is a member
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// Get members
	members, err := h.store.GetChatMembers(chatID)
	if err != nil {
		http.Error(w, "Failed to get chat members", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (h *ChatHandler) AddChatMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	// Verify user has permission to add members
	// TODO: Check if user is admin/owner

	var req models.ChatMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	// Add member
	role := models.ChatMemberRoleMember
	if req.Role != nil {
		role = models.ChatMemberRole(*req.Role)
	}

	if err := h.store.AddChatMember(chatID, req.UserID, role, req.DisplayName); err != nil {
		http.Error(w, "Failed to add chat member", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Member added successfully",
	})
}

func (h *ChatHandler) RemoveChatMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	memberID := r.PathValue("memberId")
	if memberID == "" {
		http.Error(w, "Member ID required", http.StatusBadRequest)
		return
	}

	// Verify user has permission to remove members
	// TODO: Check if user is admin/owner

	// Cannot remove yourself (use leave chat instead)
	if memberID == userID {
		http.Error(w, "Use leave endpoint to remove yourself", http.StatusBadRequest)
		return
	}

	// Remove member
	if err := h.store.RemoveChatMember(chatID, memberID); err != nil {
		http.Error(w, "Failed to remove chat member", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChatHandler) LeaveChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	// Get chat to check if user is the creator
	chat, err := h.store.GetChat(chatID)
	if err != nil || chat == nil {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// If user is the creator of a group, they can't leave (must delete or transfer ownership)
	if chat.Type == models.ChatTypeGroup && chat.CreatedBy == userID {
		http.Error(w, "Group creator must transfer ownership or delete group", http.StatusBadRequest)
		return
	}

	// Remove user from chat
	if err := h.store.RemoveChatMember(chatID, userID); err != nil {
		http.Error(w, "Failed to leave chat", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Left chat successfully",
	})
}

func (h *ChatHandler) MarkChatAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	// Verify user is a member
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// Mark all messages as read
	if err := h.store.MarkChatAsRead(chatID, userID); err != nil {
		http.Error(w, "Failed to mark chat as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Chat marked as read",
	})
}

func (h *ChatHandler) SearchChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	chatType := r.URL.Query().Get("type")
	var chatTypePtr *models.ChatType
	if chatType != "" {
		ct := models.ChatType(chatType)
		chatTypePtr = &ct
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Search chats
	chats, err := h.store.SearchChats(query, chatTypePtr, limit)
	if err != nil {
		http.Error(w, "Failed to search chats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}
