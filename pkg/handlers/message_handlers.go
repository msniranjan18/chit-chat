package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/models"
	"github.com/msniranjan18/chit-chat/pkg/store"
)

type MessageHandler struct {
	store *store.Store
}

func NewMessageHandler(store *store.Store) *MessageHandler {
	return &MessageHandler{store: store}
}

func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		http.Error(w, "Chat ID is required", http.StatusBadRequest)
		return
	}

	// Verify user is a member
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
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

	// Get messages
	messages, err := h.store.GetMessages(chatID, offset, limit)
	if err != nil {
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ChatID == "" {
		http.Error(w, "Chat ID is required", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Message content is required", http.StatusBadRequest)
		return
	}

	if req.ContentType == "" {
		req.ContentType = string(models.ContentTypeText)
	}

	// Verify user is a member
	isMember, err := h.store.IsChatMember(req.ChatID, userID)
	if err != nil || !isMember {
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
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	// Get chat info
	chat, err := h.store.GetChat(req.ChatID)
	if err != nil {
		http.Error(w, "Failed to get chat info", http.StatusInternalServerError)
		return
	}

	// Get chat members for response
	members, err := h.store.GetChatMembers(req.ChatID)
	if err != nil {
		http.Error(w, "Failed to get chat members", http.StatusInternalServerError)
		return
	}

	var memberIDs []string
	for _, member := range members {
		memberIDs = append(memberIDs, member.UserID)
	}

	users, err := h.store.GetUsersByIDs(memberIDs)
	if err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageID := r.PathValue("id")
	if messageID == "" {
		http.Error(w, "Message ID required", http.StatusBadRequest)
		return
	}

	// Get message to verify ownership
	message, err := h.store.GetMessage(messageID)
	if err != nil || message == nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Only sender can edit message
	if message.SenderID != userID {
		http.Error(w, "Only message sender can edit", http.StatusForbidden)
		return
	}

	// Cannot edit if message was forwarded
	if message.Forwarded {
		http.Error(w, "Cannot edit forwarded messages", http.StatusBadRequest)
		return
	}

	var req models.MessageUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Update message
	if err := h.store.UpdateMessageContent(messageID, req.Content); err != nil {
		http.Error(w, "Failed to update message", http.StatusInternalServerError)
		return
	}

	// Get updated message
	updatedMessage, err := h.store.GetMessage(messageID)
	if err != nil {
		http.Error(w, "Failed to get updated message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedMessage)
}

func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	messageID := r.PathValue("id")
	if messageID == "" {
		http.Error(w, "Message ID required", http.StatusBadRequest)
		return
	}

	// Get message to verify ownership
	message, err := h.store.GetMessage(messageID)
	if err != nil || message == nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Only sender can delete message
	if message.SenderID != userID {
		http.Error(w, "Only message sender can delete", http.StatusForbidden)
		return
	}

	// Delete message
	if err := h.store.DeleteMessage(messageID); err != nil {
		http.Error(w, "Failed to delete message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MessageHandler) UpdateMessageStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.MessageStatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MessageID == "" {
		http.Error(w, "Message ID is required", http.StatusBadRequest)
		return
	}

	if req.Status != string(models.MessageStatusDelivered) &&
		req.Status != string(models.MessageStatusRead) {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	// Update message status
	if err := h.store.UpdateMessageStatus(req.MessageID, userID, req.Status); err != nil {
		http.Error(w, "Failed to update message status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Message status updated",
	})
}

func (h *MessageHandler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		http.Error(w, "Chat ID is required", http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	// Verify user is a member
	isMember, err := h.store.IsChatMember(chatID, userID)
	if err != nil || !isMember {
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
		http.Error(w, "Failed to search messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
