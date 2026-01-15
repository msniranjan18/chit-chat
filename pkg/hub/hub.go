package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/msniranjan18/chit-chat/pkg/models"
	"github.com/msniranjan18/chit-chat/pkg/store"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 10 * 1024 * 1024 // 10MB
)

type Hub struct {
	Storage *store.Store

	// Registered clients by userID (multiple devices per user)
	Clients map[string]map[*Client]bool

	// Chat rooms for broadcasting
	ChatRooms map[string]map[*Client]bool

	// Broadcast channel for all messages
	Broadcast chan WsMessage

	// Channels for client management
	Register   chan *Client
	Unregister chan *Client

	mu sync.RWMutex
}

type WsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	RoomID  string          `json:"room_id"`
	Sender  string          `json:"sender"`
}

type MessageType string

const (
	MessageTypeMessage    MessageType = "message"
	MessageTypeTyping     MessageType = "typing"
	MessageTypePresence   MessageType = "presence"
	MessageTypeStatus     MessageType = "status_update"
	MessageTypeChatUpdate MessageType = "chat_update"
	MessageTypeError      MessageType = "error"
)

func NewHub(s *store.Store) *Hub {
	return &Hub{
		Storage:    s,
		Clients:    make(map[string]map[*Client]bool),
		ChatRooms:  make(map[string]map[*Client]bool),
		Broadcast:  make(chan WsMessage),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	log.Println("WebSocket hub started")
	for {
		select {
		case client := <-h.Register:
			h.handleRegister(client)

		case client := <-h.Unregister:
			h.handleUnregister(client)

		case message := <-h.Broadcast:
			h.handleBroadcast(message)
		}
	}
}

func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Register client under user
	if h.Clients[client.UserID] == nil {
		h.Clients[client.UserID] = make(map[*Client]bool)
	}
	h.Clients[client.UserID][client] = true

	// Join all active chats for this user
	chats, err := h.Storage.GetUserChats(client.UserID)
	if err == nil {
		for _, chat := range chats {
			if h.ChatRooms[chat.ID] == nil {
				h.ChatRooms[chat.ID] = make(map[*Client]bool)
			}
			h.ChatRooms[chat.ID][client] = true
			client.ActiveChats[chat.ID] = true
		}
	}

	// Update user status
	h.Storage.UpdateUserLastSeen(client.UserID, time.Now())

	// Notify contacts that user is online
	go h.notifyPresence(client.UserID, "online")
	log.Printf("Client registered: user=%s, session=%s", client.UserID, client.SessionID)
}

func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove from user clients
	if userClients, ok := h.Clients[client.UserID]; ok {
		delete(userClients, client)
		if len(userClients) == 0 {
			delete(h.Clients, client.UserID)
			// User went offline
			go h.notifyPresence(client.UserID, "offline")
		}
	}

	// Remove from all chat rooms
	for chatID := range client.ActiveChats {
		if room, ok := h.ChatRooms[chatID]; ok {
			delete(room, client)
			if len(room) == 0 {
				delete(h.ChatRooms, chatID)
			}
		}
	}

	close(client.Send)
	log.Printf("Client unregistered: user=%s, session=%s", client.UserID, client.SessionID)
}

func (h *Hub) handleBroadcast(message WsMessage) {
	switch MessageType(message.Type) {
	case MessageTypeMessage:
		h.handleChatMessage(message)
	case MessageTypeTyping:
		h.handleTypingIndicator(message)
	case MessageTypeStatus:
		h.handleStatusUpdate(message)
	default:
		log.Printf("Unknown message type: %s", message.Type)
	}
}

func (h *Hub) handleChatMessage(msg WsMessage) {
	var messageReq models.MessageRequest
	if err := json.Unmarshal(msg.Payload, &messageReq); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	// Save message to database
	savedMsg, err := h.Storage.SaveMessage(
		messageReq.ChatID,
		msg.Sender,
		messageReq.Content,
		messageReq.ContentType,
		messageReq.ReplyTo,
		messageReq.ForwardFrom,
		messageReq.Forwarded,
	)
	if err != nil {
		log.Printf("Error saving message: %v", err)
		return
	}

	// Get chat members
	members, err := h.Storage.GetChatMembers(messageReq.ChatID)
	if err != nil {
		log.Printf("Error getting chat members: %v", err)
		return
	}

	// Track which members are online vs offline
	var onlineMembers []string
	var offlineMembers []string

	h.mu.RLock()
	// Check which members are currently online
	for _, member := range members {
		if member.UserID == msg.Sender {
			// Skip sender for delivery status (they already have it)
			continue
		}
		
		// Check if member has active WebSocket connections
		if userClients, ok := h.Clients[member.UserID]; ok && len(userClients) > 0 {
			onlineMembers = append(onlineMembers, member.UserID)
		} else {
			offlineMembers = append(offlineMembers, member.UserID)
		}
	}
	h.mu.RUnlock()

	// Prepare response for online members
	response := WsMessage{
		Type:   string(MessageTypeMessage),
		RoomID: messageReq.ChatID,
		Sender: msg.Sender,
		Payload: marshalPayload(models.MessageResponse{
			Message: *savedMsg,
			Users:   []models.User{}, // Will be populated per user
		}),
	}

	// Broadcast to all online clients in the chat room
	h.mu.RLock()
	if room, ok := h.ChatRooms[messageReq.ChatID]; ok {
		for client := range room {
			// Skip sender (they already sent the message)
			if client.UserID == msg.Sender {
				continue
			}

			// Mark as delivered for online recipients
			go h.Storage.UpdateMessageStatus(savedMsg.ID, client.UserID, "delivered")

			// Send message to client
			select {
			case client.Send <- marshalMessage(response):
				// Message sent successfully
			default:
				// Client buffer full, disconnect
				close(client.Send)
				delete(room, client)
			}
		}
	}
	h.mu.RUnlock()

	// Update message status for offline members (they'll see it when they come online)
	for _, offlineMemberID := range offlineMembers {
		go h.Storage.UpdateMessageStatus(savedMsg.ID, offlineMemberID, "sent")
	}

	// Publish to Redis for other instances
	go func() {
		payload, _ := json.Marshal(msg)
		h.Storage.RDB.Publish(h.Storage.Ctx, "chat_sync", payload)
	}()

	// Update chat last activity
	go h.Storage.UpdateChatLastActivity(messageReq.ChatID)

	// Log delivery status
	log.Printf("Message delivered: chat=%s, sender=%s, online=%d, offline=%d", 
		messageReq.ChatID, msg.Sender, len(onlineMembers), len(offlineMembers))
}

func (h *Hub) handleTypingIndicator(msg WsMessage) {
	var typing models.TypingIndicator
	if err := json.Unmarshal(msg.Payload, &typing); err != nil {
		log.Printf("Error unmarshaling typing indicator: %v", err)
		return
	}

	// Broadcast typing indicator to all in chat except sender
	h.mu.RLock()
	if room, ok := h.ChatRooms[typing.ChatID]; ok {
		response := WsMessage{
			Type:    string(MessageTypeTyping),
			RoomID:  typing.ChatID,
			Sender:  msg.Sender,
			Payload: msg.Payload,
		}

		for client := range room {
			if client.UserID != msg.Sender {
				select {
				case client.Send <- marshalMessage(response):
				default:
					close(client.Send)
					delete(room, client)
				}
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) handleStatusUpdate(msg WsMessage) {
	var statusUpdate models.MessageStatusUpdate
	if err := json.Unmarshal(msg.Payload, &statusUpdate); err != nil {
		log.Printf("Error unmarshaling status update: %v", err)
		return
	}

	// Update in database
	err := h.Storage.UpdateMessageStatus(statusUpdate.MessageID, msg.Sender, statusUpdate.Status)
	if err != nil {
		log.Printf("Error updating message status: %v", err)
		return
	}

	// Get message to find original sender
	message, err := h.Storage.GetMessage(statusUpdate.MessageID)
	if err != nil {
		log.Printf("Error getting message: %v", err)
		return
	}

	// Notify sender that their message was read/delivered
	h.mu.RLock()
	if room, ok := h.ChatRooms[message.ChatID]; ok {
		response := WsMessage{
			Type:   string(MessageTypeStatus),
			RoomID: message.ChatID,
			Sender: msg.Sender,
			Payload: marshalPayload(models.MessageStatusUpdate{
				MessageID: statusUpdate.MessageID,
				Status:    statusUpdate.Status,
				ChatID:    message.ChatID,
			}),
		}

		for client := range room {
			// Find the original sender
			if client.UserID == message.SenderID {
				select {
				case client.Send <- marshalMessage(response):
				default:
					close(client.Send)
					delete(room, client)
				}
				break
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) notifyPresence(userID, status string) {
	// Get user's chats
	chats, err := h.Storage.GetUserChats(userID)
	if err != nil {
		log.Printf("Error getting user chats: %v", err)
		return
	}

	for _, chat := range chats {
		h.mu.RLock()
		if room, ok := h.ChatRooms[chat.ID]; ok {
			presenceMsg := WsMessage{
				Type:   string(MessageTypePresence),
				RoomID: chat.ID,
				Sender: userID,
				Payload: marshalPayload(models.UserPresence{
					UserID:   userID,
					IsOnline: status == "online",
					LastSeen: time.Now(),
				}),
			}

			payload := marshalMessage(presenceMsg)
			for client := range room {
				if client.UserID != userID {
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

// Helper functions
func marshalMessage(msg WsMessage) []byte {
	data, _ := json.Marshal(msg)
	return data
}

func marshalPayload(payload interface{}) json.RawMessage {
	data, _ := json.Marshal(payload)
	return data
}
