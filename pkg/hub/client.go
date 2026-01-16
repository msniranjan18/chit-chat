package hub

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Hub         *Hub
	UserID      string
	SessionID   string
	Conn        *websocket.Conn
	Send        chan []byte
	ActiveChats map[string]bool
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Warn("WebSocket read error",
					"error", err,
					"user_id", c.UserID,
					"session_id", c.SessionID)
			} else {
				c.Hub.logger.Debug("WebSocket connection closed normally",
					"user_id", c.UserID,
					"session_id", c.SessionID)
			}
			break
		}

		var wsMsg WsMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			c.Hub.logger.Error("Error unmarshaling WebSocket message",
				"error", err,
				"user_id", c.UserID,
				"session_id", c.SessionID)
			continue
		}

		// Set sender from client context
		wsMsg.Sender = c.UserID

		c.Hub.logger.Debug("Received WebSocket message",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"message_type", wsMsg.Type,
			"room_id", wsMsg.RoomID)

		// Handle message
		c.Hub.Broadcast <- wsMsg
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Hub.logger.Debug("Send channel closed, closing connection",
					"user_id", c.UserID,
					"session_id", c.SessionID)
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.Hub.logger.Warn("Failed to get WebSocket writer",
					"error", err,
					"user_id", c.UserID,
					"session_id", c.SessionID)
				return
			}

			bytesWritten, err := w.Write(message)
			if err != nil {
				c.Hub.logger.Warn("Failed to write to WebSocket",
					"error", err,
					"user_id", c.UserID,
					"session_id", c.SessionID)
				return
			}

			// Add queued messages to the current WebSocket message
			queuedMessages := 0
			n := len(c.Send)
			for i := 0; i < n; i++ {
				queuedMessage := <-c.Send
				bytesWrittenQueue, err := w.Write(queuedMessage)
				if err != nil {
					c.Hub.logger.Warn("Failed to write queued message",
						"error", err,
						"user_id", c.UserID,
						"session_id", c.SessionID)
					return
				}
				bytesWritten += bytesWrittenQueue
				queuedMessages++
			}

			if err := w.Close(); err != nil {
				c.Hub.logger.Warn("Failed to close WebSocket writer",
					"error", err,
					"user_id", c.UserID,
					"session_id", c.SessionID)
				return
			}

			c.Hub.logger.Debug("Message sent via WebSocket",
				"user_id", c.UserID,
				"session_id", c.SessionID,
				"bytes_written", bytesWritten,
				"queued_messages", queuedMessages)

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.Hub.logger.Debug("Failed to send ping, connection may be dead",
					"error", err,
					"user_id", c.UserID,
					"session_id", c.SessionID)
				return
			}
		}
	}
}

func (c *Client) JoinChat(chatID string) {
	c.Hub.mu.Lock()
	defer c.Hub.mu.Unlock()

	if c.Hub.ChatRooms[chatID] == nil {
		c.Hub.ChatRooms[chatID] = make(map[*Client]bool)
		c.Hub.logger.Debug("Created new chat room",
			"chat_id", chatID)
	}

	if !c.Hub.ChatRooms[chatID][c] {
		c.Hub.ChatRooms[chatID][c] = true
		c.ActiveChats[chatID] = true
		c.Hub.logger.Info("Client joined chat",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"chat_id", chatID,
			"room_members", len(c.Hub.ChatRooms[chatID]))
	}
}

func (c *Client) LeaveChat(chatID string) {
	c.Hub.mu.Lock()
	defer c.Hub.mu.Unlock()

	if room, ok := c.Hub.ChatRooms[chatID]; ok {
		if room[c] {
			delete(room, c)
			delete(c.ActiveChats, chatID)

			if len(room) == 0 {
				delete(c.Hub.ChatRooms, chatID)
				c.Hub.logger.Debug("Chat room empty, removed",
					"chat_id", chatID)
			}

			c.Hub.logger.Info("Client left chat",
				"user_id", c.UserID,
				"session_id", c.SessionID,
				"chat_id", chatID,
				"remaining_members", len(room))
		}
	}
}
