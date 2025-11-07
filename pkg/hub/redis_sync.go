package hub

import (
	"encoding/json"
	"log"
)

func (h *Hub) ListenToRedis() {
	// Subscribe to the global chat channel
	pubsub := h.Storage.RDB.Subscribe(h.Storage.Ctx, "chat_sync")
	defer pubsub.Close()

	ch := pubsub.Channel()
	log.Println("Listening for Redis Pub/Sub messages...")

	for msg := range ch {
		var incoming WsMessage
		if err := json.Unmarshal([]byte(msg.Payload), &incoming); err != nil {
			log.Printf("Error unmarshaling Redis message: %v", err)
			continue
		}

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
			log.Printf("Unknown Redis message type: %s", incoming.Type)
		}
	}
}

func (h *Hub) handleRedisChatMessage(msg WsMessage) {
	// Forward to local clients
	h.mu.RLock()
	if room, ok := h.ChatRooms[msg.RoomID]; ok {
		payload := marshalMessage(msg)
		for client := range room {
			// Don't send back to sender
			if client.UserID != msg.Sender {
				select {
				case client.Send <- payload:
				default:
					close(client.Send)
					delete(room, client)
				}
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) handleRedisTypingIndicator(msg WsMessage) {
	// Forward to local clients
	h.mu.RLock()
	if room, ok := h.ChatRooms[msg.RoomID]; ok {
		payload := marshalMessage(msg)
		for client := range room {
			if client.UserID != msg.Sender {
				select {
				case client.Send <- payload:
				default:
					close(client.Send)
					delete(room, client)
				}
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) handleRedisStatusUpdate(msg WsMessage) {
	// Forward to original sender if they're connected to this instance
	var statusUpdate struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"`
	}

	if err := json.Unmarshal(msg.Payload, &statusUpdate); err != nil {
		log.Printf("Error unmarshaling status update: %v", err)
		return
	}

	// Get message to find original sender
	message, err := h.Storage.GetMessage(statusUpdate.MessageID)
	if err != nil {
		log.Printf("Error getting message: %v", err)
		return
	}

	// Find and notify the sender
	h.mu.RLock()
	if userClients, ok := h.Clients[message.SenderID]; ok {
		payload := marshalMessage(msg)
		for client := range userClients {
			select {
			case client.Send <- payload:
			default:
				close(client.Send)
				delete(userClients, client)
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) handleRedisPresenceUpdate(msg WsMessage) {
	// Forward presence updates to all clients in relevant chats
	var presence struct {
		UserID   string `json:"user_id"`
		IsOnline bool   `json:"is_online"`
	}

	if err := json.Unmarshal(msg.Payload, &presence); err != nil {
		log.Printf("Error unmarshaling presence update: %v", err)
		return
	}

	// Get user's chats
	chats, err := h.Storage.GetUserChats(presence.UserID)
	if err != nil {
		log.Printf("Error getting user chats: %v", err)
		return
	}

	for _, chat := range chats {
		h.mu.RLock()
		if room, ok := h.ChatRooms[chat.ID]; ok {
			payload := marshalMessage(msg)
			for client := range room {
				if client.UserID != presence.UserID {
					select {
					case client.Send <- payload:
					default:
						close(client.Send)
						delete(room, client)
					}
				}
			}
		}
		h.mu.RUnlock()
	}
}
