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

type MessageHandler struct {
	store  *store.Store
	logger *slog.Logger
}

func NewMessageHandler(store *store.Store, logger *slog.Logger) *MessageHandler {
	return &MessageHandler{store: store, logger: logger}
}

func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("GetMessages: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("GetMessages: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		h.logger.Warn("GetMessages: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID is required", http.StatusBadRequest)
		return
	}

	h.logger.Info("GetMessages: fetching messages",
		"user_id", userID, "chat_id", chatID)

	// Verify user is a member
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		h.logger.Warn("GetMessages: user is not a member or chat not found",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// Get pagination parameters
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	h.logger.Debug("GetMessages: pagination",
		"chat_id", chatID, "user_id", userID, "offset", offset, "limit", limit)

	// Get messages
	messages, err := h.store.GetMessages(chatID, offset, limit)
	if err != nil {
		h.logger.Error("GetMessages: failed to get messages",
			"error", err, "user_id", userID, "chat_id", chatID)
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	// Get sender details
	var senderIDs []string
	for _, msg := range messages {
		senderIDs = append(senderIDs, msg.SenderID)
	}

	senders, err := h.store.GetUsersByIDs(senderIDs)
	if err != nil {
		h.logger.Error("GetMessages: failed to get sender details",
			"error", err, "user_id", userID, "chat_id", chatID, "sender_count", len(senderIDs))
		http.Error(w, "Failed to get sender details", http.StatusInternalServerError)
		return
	}

	// Create sender map
	senderMap := make(map[string]models.User)
	for _, sender := range senders {
		senderMap[sender.ID] = sender
	}

	// Add sender names to messages
	for i := range messages {
		if sender, ok := senderMap[messages[i].SenderID]; ok {
			messages[i].SenderName = sender.Name
		}
	}

	h.logger.Debug("GetMessages: retrieved messages",
		"chat_id", chatID, "user_id", userID, "message_count", len(messages))

	response := struct {
		Messages []models.Message `json:"messages"`
		Offset   int              `json:"offset"`
		Limit    int              `json:"limit"`
		Total    int              `json:"total,omitempty"`
	}{
		Messages: messages,
		Offset:   offset,
		Limit:    limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("SendMessage: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("SendMessage: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("SendMessage: invalid request body", "user_id", userID, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.logger.Info("SendMessage: sending message",
		"user_id", userID, "chat_id", req.ChatID, "content_type", req.ContentType)

	// Validate request
	if req.ChatID == "" {
		h.logger.Warn("SendMessage: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID is required", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		h.logger.Warn("SendMessage: empty content", "user_id", userID, "chat_id", req.ChatID)
		http.Error(w, "Message content is required", http.StatusBadRequest)
		return
	}

	if req.ContentType == "" {
		req.ContentType = string(models.ContentTypeText)
	}

	// Verify user is a member
	isMember, err := h.store.IsChatMember(req.ChatID, userID)
	if err != nil || !isMember {
		h.logger.Warn("SendMessage: user is not a member or chat not found",
			"user_id", userID, "chat_id", req.ChatID, "error", err)
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	// Save message
	message, err := h.store.SaveMessage(
		req.ChatID,
		userID,
		req.Content,
		req.ContentType,
		req.ReplyTo,
		req.ForwardFrom,
		req.Forwarded,
	)
	if err != nil {
		h.logger.Error("SendMessage: failed to save message",
			"error", err, "user_id", userID, "chat_id", req.ChatID)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	// Get chat info
	chat, err := h.store.GetChat(req.ChatID)
	if err != nil {
		h.logger.Error("SendMessage: failed to get chat info",
			"error", err, "user_id", userID, "chat_id", req.ChatID)
		http.Error(w, "Failed to get chat info", http.StatusInternalServerError)
		return
	}

	// Get chat members for response
	members, err := h.store.GetChatMembers(req.ChatID)
	if err != nil {
		h.logger.Error("SendMessage: failed to get chat members",
			"error", err, "user_id", userID, "chat_id", req.ChatID)
		http.Error(w, "Failed to get chat members", http.StatusInternalServerError)
		return
	}

	var memberIDs []string
	for _, member := range members {
		memberIDs = append(memberIDs, member.UserID)
	}

	users, err := h.store.GetUsersByIDs(memberIDs)
	if err != nil {
		h.logger.Error("SendMessage: failed to get users",
			"error", err, "user_id", userID, "chat_id", req.ChatID, "member_count", len(memberIDs))
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	h.logger.Info("SendMessage: message sent successfully",
		"user_id", userID, "chat_id", req.ChatID, "message_id", message.ID)

	response := models.MessageResponse{
		Message:  *message,
		ChatInfo: *chat,
		Users:    users,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *MessageHandler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		h.logger.Warn("UpdateMessage: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("UpdateMessage: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageID := r.PathValue("id")
	if messageID == "" {
		h.logger.Warn("UpdateMessage: missing message ID", "user_id", userID)
		http.Error(w, "Message ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("UpdateMessage: updating message", "user_id", userID, "message_id", messageID)

	// Get message to verify ownership
	message, err := h.store.GetMessage(messageID)
	if err != nil || message == nil {
		h.logger.Warn("UpdateMessage: message not found",
			"user_id", userID, "message_id", messageID, "error", err)
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Only sender can edit message
	if message.SenderID != userID {
		h.logger.Warn("UpdateMessage: user is not the sender",
			"user_id", userID, "message_id", messageID, "sender_id", message.SenderID)
		http.Error(w, "Only message sender can edit", http.StatusForbidden)
		return
	}

	// Cannot edit if message was forwarded
	if message.Forwarded {
		h.logger.Warn("UpdateMessage: cannot edit forwarded message",
			"user_id", userID, "message_id", messageID)
		http.Error(w, "Cannot edit forwarded messages", http.StatusBadRequest)
		return
	}

	var req models.MessageUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("UpdateMessage: invalid request body",
			"user_id", userID, "message_id", messageID, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		h.logger.Warn("UpdateMessage: empty content",
			"user_id", userID, "message_id", messageID)
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Update message
	if err := h.store.UpdateMessageContent(messageID, req.Content); err != nil {
		h.logger.Error("UpdateMessage: failed to update message",
			"error", err, "user_id", userID, "message_id", messageID)
		http.Error(w, "Failed to update message", http.StatusInternalServerError)
		return
	}

	// Get updated message
	updatedMessage, err := h.store.GetMessage(messageID)
	if err != nil {
		h.logger.Error("UpdateMessage: failed to get updated message",
			"error", err, "user_id", userID, "message_id", messageID)
		http.Error(w, "Failed to get updated message", http.StatusInternalServerError)
		return
	}

	h.logger.Info("UpdateMessage: message updated successfully",
		"user_id", userID, "message_id", messageID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedMessage)
}

func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.logger.Warn("DeleteMessage: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("DeleteMessage: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageID := r.PathValue("id")
	if messageID == "" {
		h.logger.Warn("DeleteMessage: missing message ID", "user_id", userID)
		http.Error(w, "Message ID required", http.StatusBadRequest)
		return
	}

	h.logger.Info("DeleteMessage: deleting message", "user_id", userID, "message_id", messageID)

	// Get message to verify ownership
	message, err := h.store.GetMessage(messageID)
	if err != nil || message == nil {
		h.logger.Warn("DeleteMessage: message not found",
			"user_id", userID, "message_id", messageID, "error", err)
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Only sender can delete message
	if message.SenderID != userID {
		h.logger.Warn("DeleteMessage: user is not the sender",
			"user_id", userID, "message_id", messageID, "sender_id", message.SenderID)
		http.Error(w, "Only message sender can delete", http.StatusForbidden)
		return
	}

	// Delete message
	if err := h.store.DeleteMessage(messageID); err != nil {
		h.logger.Error("DeleteMessage: failed to delete message",
			"error", err, "user_id", userID, "message_id", messageID)
		http.Error(w, "Failed to delete message", http.StatusInternalServerError)
		return
	}

	h.logger.Info("DeleteMessage: message deleted successfully",
		"user_id", userID, "message_id", messageID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *MessageHandler) UpdateMessageStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("UpdateMessageStatus: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("UpdateMessageStatus: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.MessageStatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("UpdateMessageStatus: invalid request body", "user_id", userID, "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MessageID == "" {
		h.logger.Warn("UpdateMessageStatus: missing message ID", "user_id", userID)
		http.Error(w, "Message ID is required", http.StatusBadRequest)
		return
	}

	if req.Status != string(models.MessageStatusDelivered) &&
		req.Status != string(models.MessageStatusRead) {
		h.logger.Warn("UpdateMessageStatus: invalid status",
			"user_id", userID, "message_id", req.MessageID, "status", req.Status)
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	h.logger.Debug("UpdateMessageStatus: updating status",
		"user_id", userID, "message_id", req.MessageID, "status", req.Status)

	// Update message status
	if err := h.store.UpdateMessageStatus(req.MessageID, userID, req.Status); err != nil {
		h.logger.Error("UpdateMessageStatus: failed to update message status",
			"error", err, "user_id", userID, "message_id", req.MessageID)
		http.Error(w, "Failed to update message status", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("UpdateMessageStatus: status updated successfully",
		"user_id", userID, "message_id", req.MessageID, "status", req.Status)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Message status updated",
	})
}

func (h *MessageHandler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("SearchMessages: method not allowed", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		h.logger.Warn("SearchMessages: unauthorized request", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		h.logger.Warn("SearchMessages: missing chat ID", "user_id", userID)
		http.Error(w, "Chat ID is required", http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		h.logger.Warn("SearchMessages: missing search query", "user_id", userID, "chat_id", chatID)
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	h.logger.Info("SearchMessages: searching messages",
		"user_id", userID, "chat_id", chatID, "query", query)

	// Verify user is a member
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
		h.logger.Warn("SearchMessages: user is not a member or chat not found",
			"user_id", userID, "chat_id", chatID, "error", err)
		http.Error(w, "Chat not found or access denied", http.StatusNotFound)
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	// Search messages
	messages, err := h.store.SearchMessages(chatID, query, limit)
	if err != nil {
		h.logger.Error("SearchMessages: failed to search messages",
			"error", err, "user_id", userID, "chat_id", chatID, "query", query)
		http.Error(w, "Failed to search messages", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("SearchMessages: search completed",
		"user_id", userID, "chat_id", chatID, "query", query, "result_count", len(messages))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
