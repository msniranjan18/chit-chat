package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/models"
	"github.com/msniranjan18/chit-chat/pkg/store"
)

type ChatHandler struct {
	store  *store.Store
	logger *slog.Logger
}

func NewChatHandler(store *store.Store, logger *slog.Logger) *ChatHandler {
	return &ChatHandler{store: store, logger: logger}
}

func (h *ChatHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("GetChats: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.Info("GetChats: fetching user chats", "user_id", userID)

	// Get user's chats
	chats, err := h.store.GetUserChats(userID)
	if err != nil {
		h.logger.Error("GetChats: failed to get chats", "error", err, "user_id", userID)
		http.Error(w, "Failed to get chats", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("GetChats: retrieved chats", "user_id", userID, "chat_count", len(chats))

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
		h.logger.Warn("GetChat: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract chat ID from URL
	chatID := r.PathValue("id")
	if chatID == "" {
		h.logger.Warn("GetChat: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("GetChat: fetching chat details", "user_id", userID, "chat_id", chatID)

	// Verify user is a member of the chat
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		h.logger.Warn("GetChat: user is not a member or chat not found",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// Get chat details
	chat, err := h.store.GetChat(chatID)
	if err != nil {
		h.logger.Error("GetChat: failed to get chat", "error", err, "chat_id", chatID, "user_id", userID)
		http.Error(w, "Failed to get chat", http.StatusInternalServerError)
		return
	}

	if chat == nil {
		h.logger.Warn("GetChat: chat not found", "chat_id", chatID, "user_id", userID)
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// Get chat members
	members, err := h.store.GetChatMembers(chatID)
	if err != nil {
		h.logger.Error("GetChat: failed to get chat members",
			"error", err, "chat_id", chatID, "user_id", userID)
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
		h.logger.Error("GetChat: failed to get users",
			"error", err, "chat_id", chatID, "user_id", userID, "member_count", len(memberIDs))
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("GetChat: retrieved chat details",
		"chat_id", chatID, "user_id", userID, "member_count", len(members))

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
		h.logger.Warn("CreateChat: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("CreateChat: invalid request body", "user_id", userID, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("CreateChat: creating new chat",
		"user_id", userID, "type", req.Type, "name", req.Name, "user_count", len(req.UserIDs))

	// Validate request
	if req.Type == "" {
		h.logger.Warn("CreateChat: missing chat type", "user_id", userID)
		http.Error(w, "Chat type is required", http.StatusBadRequest)
		return
	}

	if req.Type == models.ChatTypeGroup && (req.Name == nil || *req.Name == "") {
		h.logger.Warn("CreateChat: missing group name", "user_id", userID)
		http.Error(w, "Group name is required", http.StatusBadRequest)
		return
	}

	if len(req.UserIDs) == 0 && req.Type != models.ChatTypeDirect {
		h.logger.Warn("CreateChat: no users specified", "user_id", userID, "type", req.Type)
		http.Error(w, "At least one user is required", http.StatusBadRequest)
		return
	}

	// For direct chat, ensure exactly 2 users (creator + one other)
	if req.Type == models.ChatTypeDirect {
		if len(req.UserIDs) != 1 {
			h.logger.Warn("CreateChat: direct chat requires exactly one other user",
				"user_id", userID, "provided_user_count", len(req.UserIDs))
			http.Error(w, "Direct chat requires exactly one other user", http.StatusBadRequest)
			return
		}

		// Check if direct chat already exists
		existingChat, err := h.store.GetDirectChat(userID, req.UserIDs[0])
		if err != nil {
			h.logger.Error("CreateChat: failed to check existing direct chat",
				"error", err, "user_id", userID, "other_user_id", req.UserIDs[0])
			http.Error(w, "Failed to check existing chat", http.StatusInternalServerError)
			return
		}

		if existingChat != nil {
			h.logger.Info("CreateChat: returning existing direct chat",
				"user_id", userID, "other_user_id", req.UserIDs[0], "chat_id", existingChat.ID)
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
		h.logger.Error("CreateChat: failed to create chat",
			"error", err, "user_id", userID, "type", req.Type)
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	h.logger.Info("CreateChat: chat created successfully",
		"chat_id", chat.ID, "user_id", userID, "type", chat.Type)

	// Get members for response
	members, err := h.store.GetChatMembers(chat.ID)
	if err != nil {
		h.logger.Error("CreateChat: failed to get chat members",
			"error", err, "chat_id", chat.ID, "user_id", userID)
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
		h.logger.Warn("UpdateChat: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		h.logger.Warn("UpdateChat: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("UpdateChat: updating chat", "user_id", userID, "chat_id", chatID)

	// Verify user is an admin of the chat
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		h.logger.Warn("UpdateChat: user is not a member or chat not found",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// TODO: Check if user has permission to update (admin/owner)

	var req models.ChatUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("UpdateChat: invalid request body",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Debug("UpdateChat: update request",
		"user_id", userID, "chat_id", chatID, "update_fields", req)

	// Update chat
	if err := h.store.UpdateChat(chatID, &req); err != nil {
		h.logger.Error("UpdateChat: failed to update chat",
			"error", err, "user_id", userID, "chat_id", chatID)
		http.Error(w, "Failed to update chat", http.StatusInternalServerError)
		return
	}

	// Get updated chat
	chat, err := h.store.GetChat(chatID)
	if err != nil {
		h.logger.Error("UpdateChat: failed to get updated chat",
			"error", err, "user_id", userID, "chat_id", chatID)
		http.Error(w, "Failed to get updated chat", http.StatusInternalServerError)
		return
	}

	h.logger.Info("UpdateChat: chat updated successfully",
		"user_id", userID, "chat_id", chatID)

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
		h.logger.Warn("DeleteChat: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		h.logger.Warn("DeleteChat: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("DeleteChat: attempting to delete chat", "user_id", userID, "chat_id", chatID)

	// Verify user is the creator or admin
	chat, err := h.store.GetChat(chatID)
	if err != nil || chat == nil {
		h.logger.Warn("DeleteChat: chat not found",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// Only creator can delete (or admin in future)
	if chat.CreatedBy != userID {
		h.logger.Warn("DeleteChat: user is not the creator",
			"user_id", userID, "chat_id", chatID, "creator", chat.CreatedBy)
		http.Error(w, "Only chat creator can delete", http.StatusForbidden)
		return
	}

	// Delete chat
	if err := h.store.DeleteChat(chatID); err != nil {
		h.logger.Error("DeleteChat: failed to delete chat",
			"error", err, "user_id", userID, "chat_id", chatID)
		http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
		return
	}

	h.logger.Info("DeleteChat: chat deleted successfully", "user_id", userID, "chat_id", chatID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChatHandler) GetChatMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("GetChatMembers: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		h.logger.Warn("GetChatMembers: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	h.logger.Debug("GetChatMembers: fetching members", "user_id", userID, "chat_id", chatID)

	// Verify user is a member
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		h.logger.Warn("GetChatMembers: user is not a member or chat not found",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// Get members
	members, err := h.store.GetChatMembers(chatID)
	if err != nil {
		h.logger.Error("GetChatMembers: failed to get chat members",
			"error", err, "user_id", userID, "chat_id", chatID)
		http.Error(w, "Failed to get chat members", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("GetChatMembers: retrieved members",
		"chat_id", chatID, "user_id", userID, "member_count", len(members))

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
		h.logger.Warn("AddChatMember: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		h.logger.Warn("AddChatMember: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("AddChatMember: adding member to chat", "requester_id", userID, "chat_id", chatID)

	// Verify user has permission to add members
	// TODO: Check if user is admin/owner

	var req models.ChatMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("AddChatMember: invalid request body",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		h.logger.Warn("AddChatMember: missing user ID", "requester_id", userID, "chat_id", chatID)
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	h.logger.Debug("AddChatMember: adding user to chat",
		"requester_id", userID, "chat_id", chatID, "target_user_id", req.UserID, "role", req.Role)

	// Add member
	role := models.ChatMemberRoleMember
	if req.Role != nil {
		role = models.ChatMemberRole(*req.Role)
	}

	displayName := ""
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}

	if err := h.store.AddChatMember(chatID, req.UserID, role, displayName); err != nil {
		h.logger.Error("AddChatMember: failed to add chat member",
			"error", err, "requester_id", userID, "chat_id", chatID, "target_user_id", req.UserID)
		http.Error(w, "Failed to add chat member", http.StatusInternalServerError)
		return
	}

	h.logger.Info("AddChatMember: member added successfully",
		"requester_id", userID, "chat_id", chatID, "target_user_id", req.UserID, "role", role)

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
		h.logger.Warn("RemoveChatMember: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		h.logger.Warn("RemoveChatMember: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	memberID := r.PathValue("memberId")
	if memberID == "" {
		h.logger.Warn("RemoveChatMember: missing member ID", "user_id", userID, "chat_id", chatID)
		http.Error(w, "Member ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("RemoveChatMember: removing member from chat",
		"requester_id", userID, "chat_id", chatID, "member_id", memberID)

	// Verify user has permission to remove members
	// TODO: Check if user is admin/owner

	// Cannot remove yourself (use leave chat instead)
	if memberID == userID {
		h.logger.Warn("RemoveChatMember: cannot remove self",
			"user_id", userID, "chat_id", chatID)
		http.Error(w, "Use leave endpoint to remove yourself", http.StatusBadRequest)
		return
	}

	// Remove member
	if err := h.store.RemoveChatMember(chatID, memberID); err != nil {
		h.logger.Error("RemoveChatMember: failed to remove chat member",
			"error", err, "requester_id", userID, "chat_id", chatID, "member_id", memberID)
		http.Error(w, "Failed to remove chat member", http.StatusInternalServerError)
		return
	}

	h.logger.Info("RemoveChatMember: member removed successfully",
		"requester_id", userID, "chat_id", chatID, "member_id", memberID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *ChatHandler) LeaveChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("LeaveChat: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		h.logger.Warn("LeaveChat: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("LeaveChat: user attempting to leave chat", "user_id", userID, "chat_id", chatID)

	// Get chat to check if user is the creator
	chat, err := h.store.GetChat(chatID)
	if err != nil || chat == nil {
		h.logger.Warn("LeaveChat: chat not found",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// If user is the creator of a group, they can't leave (must delete or transfer ownership)
	if chat.Type == models.ChatTypeGroup && chat.CreatedBy == userID {
		h.logger.Warn("LeaveChat: group creator cannot leave",
			"user_id", userID, "chat_id", chatID, "chat_type", chat.Type)
		http.Error(w, "Group creator must transfer ownership or delete group", http.StatusBadRequest)
		return
	}

	// Remove user from chat
	if err := h.store.RemoveChatMember(chatID, userID); err != nil {
		h.logger.Error("LeaveChat: failed to remove user from chat",
			"error", err, "user_id", userID, "chat_id", chatID)
		http.Error(w, "Failed to leave chat", http.StatusInternalServerError)
		return
	}

	h.logger.Info("LeaveChat: user left chat successfully", "user_id", userID, "chat_id", chatID)

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
		h.logger.Warn("MarkChatAsRead: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		h.logger.Warn("MarkChatAsRead: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID required", http.StatusBadRequest)
		return
	}

	h.logger.Debug("MarkChatAsRead: marking chat as read", "user_id", userID, "chat_id", chatID)

	// Verify user is a member
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		h.logger.Warn("MarkChatAsRead: user is not a member or chat not found",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// Mark all messages as read
	if err := h.store.MarkChatAsRead(chatID, userID); err != nil {
		h.logger.Error("MarkChatAsRead: failed to mark chat as read",
			"error", err, "user_id", userID, "chat_id", chatID)
		http.Error(w, "Failed to mark chat as read", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("MarkChatAsRead: chat marked as read successfully",
		"user_id", userID, "chat_id", chatID)

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
		h.logger.Warn("SearchChats: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		h.logger.Warn("SearchChats: missing search query", "user_id", userID)
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

	h.logger.Info("SearchChats: searching chats",
		"user_id", userID, "query", query, "type", chatType, "limit", limit)

	// Search chats
	chats, err := h.store.SearchChats(query, chatTypePtr, limit)
	if err != nil {
		h.logger.Error("SearchChats: failed to search chats",
			"error", err, "user_id", userID, "query", query)
		http.Error(w, "Failed to search chats", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("SearchChats: search completed",
		"user_id", userID, "query", query, "result_count", len(chats))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}
