package hub

import (
	"encoding/json"
	"log"
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
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		var wsMsg WsMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Error unmarshaling WebSocket message: %v", err)
			continue
		}

		// Set sender from client context
		wsMsg.Sender = c.UserID

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
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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
	}
	c.Hub.ChatRooms[chatID][c] = true
	c.ActiveChats[chatID] = true
}

func (c *Client) LeaveChat(chatID string) {
	c.Hub.mu.Lock()
	defer c.Hub.mu.Unlock()

	if room, ok := c.Hub.ChatRooms[chatID]; ok {
		delete(room, c)
		if len(room) == 0 {
			delete(c.Hub.ChatRooms, chatID)
		}
	}
	delete(c.ActiveChats, chatID)
}
