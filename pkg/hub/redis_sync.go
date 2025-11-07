package hub

import (
	"encoding/json"
)

func (h *Hub) ListenToRedis() {
	// Subscribe to the global chat channel
	pubsub := h.Storage.RDB.Subscribe(h.Storage.Ctx, "chat_sync")
	defer pubsub.Close()

	ch := pubsub.Channel()
	h.logger.Info("Listening for Redis Pub/Sub messages")

	for msg := range ch {
		var incoming WsMessage
		if err := json.Unmarshal([]byte(msg.Payload), &incoming); err != nil {
			h.logger.Error("Error unmarshaling Redis message",
				"error", err,
				"channel", msg.Channel)
			continue
		}

		h.logger.Debug("Received Redis message",
			"channel", msg.Channel,
			"type", incoming.Type,
			"sender", incoming.Sender,
			"room_id", incoming.RoomID)

		// Process message based on type
		switch MessageType(incoming.Type) {
		case MessageTypeMessage:
			h.handleRedisChatMessage(incoming)
		case MessageTypeTyping:
			h.handleRedisTypingIndicator(incoming)
		case MessageTypeStatus:
			h.handleRedisStatusUpdate(incoming)
		case MessageTypePresence:
			h.handleRedisPresenceUpdate(incoming)
		default:
			h.logger.Warn("Unknown Redis message type",
				"type", incoming.Type,
				"sender", incoming.Sender)
		}
	}
}

func (h *Hub) handleRedisChatMessage(msg WsMessage) {
	h.logger.Debug("Forwarding Redis chat message to local clients",
		"sender", msg.Sender,
		"room_id", msg.RoomID)

	// Forward to local clients
	forwardedCount := 0
	h.mu.RLock()
	if room, ok := h.ChatRooms[msg.RoomID]; ok {
		payload := marshalMessage(msg)
		for client := range room {
			// Don't send back to sender
			if client.UserID != msg.Sender {
				select {
				case client.Send <- payload:
					forwardedCount++
				default:
					close(client.Send)
					delete(room, client)
					h.logger.Warn("Client buffer full during Redis forwarding",
						"user_id", client.UserID,
						"room_id", msg.RoomID)
				}
			}
		}
	}
	h.mu.RUnlock()

	h.logger.Debug("Redis chat message forwarded",
		"room_id", msg.RoomID,
		"forwarded_to", forwardedCount)
}

func (h *Hub) handleRedisTypingIndicator(msg WsMessage) {
	h.logger.Debug("Forwarding Redis typing indicator to local clients",
		"sender", msg.Sender,
		"room_id", msg.RoomID)

	// Forward to local clients
	forwardedCount := 0
	h.mu.RLock()
	if room, ok := h.ChatRooms[msg.RoomID]; ok {
		payload := marshalMessage(msg)
		for client := range room {
			if client.UserID != msg.Sender {
				select {
				case client.Send <- payload:
					forwardedCount++
				default:
					close(client.Send)
					delete(room, client)
					h.logger.Warn("Client buffer full during Redis typing forwarding",
						"user_id", client.UserID,
						"room_id", msg.RoomID)
				}
			}
		}
	}
	h.mu.RUnlock()

	h.logger.Debug("Redis typing indicator forwarded",
		"room_id", msg.RoomID,
		"forwarded_to", forwardedCount)
}

func (h *Hub) handleRedisStatusUpdate(msg WsMessage) {
	h.logger.Debug("Processing Redis status update")

	// Forward to original sender if they're connected to this instance
	var statusUpdate struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"`
	}

	if err := json.Unmarshal(msg.Payload, &statusUpdate); err != nil {
		h.logger.Error("Error unmarshaling Redis status update",
			"error", err,
			"raw_payload", string(msg.Payload))
		return
	}

	// Get message to find original sender
	message, err := h.Storage.GetMessage(statusUpdate.MessageID)
	if err != nil {
		h.logger.Error("Error getting message for Redis status update",
			"error", err,
			"message_id", statusUpdate.MessageID)
		return
	}

	h.logger.Debug("Forwarding Redis status update to original sender",
		"message_id", statusUpdate.MessageID,
		"original_sender", message.SenderID,
		"status", statusUpdate.Status)

	// Find and notify the sender
	forwardedCount := 0
	h.mu.RLock()
	if userClients, ok := h.Clients[message.SenderID]; ok {
		payload := marshalMessage(msg)
		for client := range userClients {
			select {
			case client.Send <- payload:
				forwardedCount++
			default:
				close(client.Send)
				delete(userClients, client)
				h.logger.Warn("Client buffer full during Redis status forwarding",
					"user_id", client.UserID,
					"message_id", statusUpdate.MessageID)
			}
		}
	}
	h.mu.RUnlock()

	h.logger.Debug("Redis status update forwarded",
		"message_id", statusUpdate.MessageID,
		"original_sender", message.SenderID,
		"forwarded_to", forwardedCount)
}

func (h *Hub) handleRedisPresenceUpdate(msg WsMessage) {
	h.logger.Debug("Processing Redis presence update")

	// Forward presence updates to all clients in relevant chats
	var presence struct {
		UserID   string `json:"user_id"`
		IsOnline bool   `json:"is_online"`
	}

	if err := json.Unmarshal(msg.Payload, &presence); err != nil {
		h.logger.Error("Error unmarshaling Redis presence update",
			"error", err,
			"raw_payload", string(msg.Payload))
		return
	}

	// Get user's chats
	chats, err := h.Storage.GetUserChats(presence.UserID)
	if err != nil {
		h.logger.Error("Error getting user chats for Redis presence",
			"error", err,
			"user_id", presence.UserID)
		return
	}

	totalForwarded := 0
	for _, chat := range chats {
		h.logger.Debug("Forwarding presence update in chat",
			"user_id", presence.UserID,
			"is_online", presence.IsOnline,
			"chat_id", chat.ID)

		forwardedInChat := 0
		h.mu.RLock()
		if room, ok := h.ChatRooms[chat.ID]; ok {
			payload := marshalMessage(msg)
			for client := range room {
				if client.UserID != presence.UserID {
					select {
					case client.Send <- payload:
						forwardedInChat++
						totalForwarded++
					default:
						close(client.Send)
						delete(room, client)
						h.logger.Warn("Client buffer full during Redis presence forwarding",
							"user_id", client.UserID,
							"chat_id", chat.ID)
					}
				}
			}
		}
		h.mu.RUnlock()

		h.logger.Debug("Presence update forwarded in chat",
			"chat_id", chat.ID,
			"forwarded_to", forwardedInChat)
	}

	h.logger.Info("Redis presence update completed",
		"user_id", presence.UserID,
		"is_online", presence.IsOnline,
		"total_chats", len(chats),
		"total_forwarded", totalForwarded)
}
