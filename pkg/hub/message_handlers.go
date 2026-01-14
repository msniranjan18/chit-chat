package hub

import (
	"encoding/json"
	"log"

	"github.com/msniranjan18/chit-chat/pkg/models"
)

type MessageHandler struct {
	hub *Hub
}

func NewMessageHandler(hub *Hub) *MessageHandler {
	return &MessageHandler{hub: hub}
}

func (h *MessageHandler) HandleMessage(data []byte) error {
	var msg WsMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	switch MessageType(msg.Type) {
	case MessageTypeMessage:
		return h.handleChatMessage(msg)
	case MessageTypeTyping:
		return h.handleTypingIndicator(msg)
	case MessageTypeStatus:
		return h.handleStatusUpdate(msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
		return nil
	}
}

func (h *MessageHandler) handleChatMessage(msg WsMessage) error {
	var messageReq models.MessageRequest
	if err := json.Unmarshal(msg.Payload, &messageReq); err != nil {
		return err
	}

	// Save message to database
	savedMsg, err := h.hub.Storage.SaveMessage(
		messageReq.ChatID,
		msg.Sender,
		messageReq.Content,
		messageReq.ContentType,
		messageReq.ReplyTo,
		messageReq.ForwardFrom,
		messageReq.Forwarded,
	)
	if err != nil {
		return err
	}

	// Prepare broadcast
	response := WsMessage{
		Type:   string(MessageTypeMessage),
		RoomID: messageReq.ChatID,
		Sender: msg.Sender,
		Payload: marshalPayload(models.MessageResponse{
			Message: *savedMsg,
		}),
	}

	// Broadcast to chat room
	h.hub.mu.RLock()
	if room, ok := h.hub.ChatRooms[messageReq.ChatID]; ok {
		payload := marshalMessage(response)
		for client := range room {
			// Mark as delivered if recipient is online
			if client.UserID != msg.Sender {
				go h.hub.Storage.UpdateMessageStatus(savedMsg.ID, client.UserID, "delivered")
			}

			select {
			case client.Send <- payload:
			default:
				close(client.Send)
				delete(room, client)
			}
		}
	}
	h.hub.mu.RUnlock()

	return nil
}

func (h *MessageHandler) handleTypingIndicator(msg WsMessage) error {
	var typing models.TypingIndicator
	if err := json.Unmarshal(msg.Payload, &typing); err != nil {
		return err
	}

	h.hub.mu.RLock()
	if room, ok := h.hub.ChatRooms[typing.ChatID]; ok {
		response := WsMessage{
			Type:    string(MessageTypeTyping),
			RoomID:  typing.ChatID,
			Sender:  msg.Sender,
			Payload: msg.Payload,
		}

		payload := marshalMessage(response)
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
	h.hub.mu.RUnlock()

	return nil
}

func (h *MessageHandler) handleStatusUpdate(msg WsMessage) error {
	var statusUpdate models.MessageStatusUpdate
	if err := json.Unmarshal(msg.Payload, &statusUpdate); err != nil {
		return err
	}

	// Update in database
	err := h.hub.Storage.UpdateMessageStatus(statusUpdate.MessageID, msg.Sender, statusUpdate.Status)
	if err != nil {
		return err
	}

	// Get message to find original sender
	message, err := h.hub.Storage.GetMessage(statusUpdate.MessageID)
	if err != nil {
		return err
	}

	// Notify sender
	h.hub.mu.RLock()
	if room, ok := h.hub.ChatRooms[message.ChatID]; ok {
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

		payload := marshalMessage(response)
		for client := range room {
			if client.UserID == message.SenderID {
				select {
				case client.Send <- payload:
				default:
					close(client.Send)
					delete(room, client)
				}
				break
			}
		}
	}
	h.hub.mu.RUnlock()

	return nil
}
