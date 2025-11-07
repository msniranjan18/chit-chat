package models

import (
	"time"
)

type ChatType string

const (
	ChatTypeDirect  ChatType = "direct"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type Chat struct {
	ID           string    `json:"id" db:"id"`
	Type         ChatType  `json:"type" db:"type"`
	Name         *string   `json:"name,omitempty" db:"name"`
	Description  *string   `json:"description,omitempty" db:"description"`
	AvatarURL    *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedBy    string    `json:"created_by" db:"created_by"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	LastActivity time.Time `json:"last_activity" db:"last_activity"`
	LastMessage  *Message  `json:"last_message,omitempty" db:"-"`
	UnreadCount  int       `json:"unread_count,omitempty" db:"-"`
	IsArchived   bool      `json:"is_archived" db:"is_archived"`
	IsMuted      bool      `json:"is_muted" db:"is_muted"`
	IsPinned     bool      `json:"is_pinned" db:"is_pinned"`
}

type ChatMember struct {
	ChatID      string     `json:"chat_id" db:"chat_id"`
	UserID      string     `json:"user_id" db:"user_id"`
	JoinedAt    time.Time  `json:"joined_at" db:"joined_at"`
	LastReadAt  time.Time  `json:"last_read_at" db:"last_read_at"`
	Role        string     `json:"role" db:"role"`
	IsAdmin     bool       `json:"is_admin" db:"is_admin"`
	DisplayName *string    `json:"display_name,omitempty" db:"display_name"`
	IsBanned    bool       `json:"is_banned" db:"is_banned"`
	BannedUntil *time.Time `json:"banned_until,omitempty" db:"banned_until"`
}

type ChatMemberRole string

const (
	ChatMemberRoleOwner  ChatMemberRole = "owner"
	ChatMemberRoleAdmin  ChatMemberRole = "admin"
	ChatMemberRoleMember ChatMemberRole = "member"
	ChatMemberRoleViewer ChatMemberRole = "viewer"
)

type ChatRequest struct {
	Type        ChatType `json:"type"`
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	AvatarURL   *string  `json:"avatar_url,omitempty"`
	UserIDs     []string `json:"user_ids"` // For direct chat: [other_user_id], For group: all members
}

type ChatUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	IsArchived  *bool   `json:"is_archived,omitempty"`
	IsMuted     *bool   `json:"is_muted,omitempty"`
	IsPinned    *bool   `json:"is_pinned,omitempty"`
}

type ChatMemberRequest struct {
	UserID      string  `json:"user_id"`
	Role        *string `json:"role,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
}

type ChatResponse struct {
	Chat    Chat         `json:"chat"`
	Members []ChatMember `json:"members,omitempty"`
	Users   []User       `json:"users,omitempty"`
}

type ChatListResponse struct {
	Chats []Chat `json:"chats"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

type ChatSearchRequest struct {
	Query string    `json:"query"`
	Type  *ChatType `json:"type,omitempty"`
}
