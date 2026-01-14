package models

import (
	"time"
)

type Message struct {
	ID           string     `json:"id" db:"id"`
	ChatID       string     `json:"chat_id" db:"chat_id"`
	SenderID     string     `json:"sender_id" db:"sender_id"`
	SenderName   string     `json:"sender_name,omitempty" db:"-"`
	Content      string     `json:"content" db:"content"`
	ContentType  string     `json:"content_type" db:"content_type"`
	MediaURL     *string    `json:"media_url,omitempty" db:"media_url"`
	ThumbnailURL *string    `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	FileSize     *int64     `json:"file_size,omitempty" db:"file_size"`
	Duration     *int       `json:"duration,omitempty" db:"duration"` // For audio/video
	Status       string     `json:"status" db:"status"`
	SentAt       time.Time  `json:"sent_at" db:"sent_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty" db:"delivered_at"`
	ReadAt       *time.Time `json:"read_at,omitempty" db:"read_at"`
	ReplyTo      *string    `json:"reply_to,omitempty" db:"reply_to"`
	ReplyMessage *Message   `json:"reply_message,omitempty" db:"-"`
	Forwarded    bool       `json:"forwarded" db:"forwarded"`
	ForwardFrom  *string    `json:"forward_from,omitempty" db:"forward_from"`
	IsEdited     bool       `json:"is_edited" db:"is_edited"`
	EditedAt     *time.Time `json:"edited_at,omitempty" db:"edited_at"`
	IsDeleted    bool       `json:"is_deleted" db:"is_deleted"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeImage    ContentType = "image"
	ContentTypeVideo    ContentType = "video"
	ContentTypeAudio    ContentType = "audio"
	ContentTypeDocument ContentType = "document"
	ContentTypeLocation ContentType = "location"
	ContentTypeContact  ContentType = "contact"
	ContentTypeSticker  ContentType = "sticker"
)

type MessageRequest struct {
	ChatID      string  `json:"chat_id"`
	Content     string  `json:"content"`
	ContentType string  `json:"content_type"`
	ReplyTo     *string `json:"reply_to,omitempty"`
	ForwardFrom *string `json:"forward_from,omitempty"`
	Forwarded   bool    `json:"forwarded,omitempty"`
}

type MessageUpdateRequest struct {
	Content string `json:"content,omitempty"`
}

type MessageStatusUpdate struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	ChatID    string `json:"chat_id,omitempty"`
}

type TypingIndicator struct {
	ChatID   string `json:"chat_id"`
	UserID   string `json:"user_id"`
	IsTyping bool   `json:"is_typing"`
}

type MessageResponse struct {
	Message  Message `json:"message"`
	ChatInfo Chat    `json:"chat_info,omitempty"`
	Users    []User  `json:"users,omitempty"`
}

type BulkMessageStatusUpdate struct {
	MessageIDs []string `json:"message_ids"`
	Status     string   `json:"status"`
}
