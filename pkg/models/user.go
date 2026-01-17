package models

import (
	"time"
)

type User struct {
	ID        string    `json:"id" db:"id"`
	Phone     string    `json:"phone" db:"phone"`
	Name      string    `json:"name" db:"name"`
	Status    string    `json:"status" db:"status"`
	AvatarURL *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	LastSeen  time.Time `json:"last_seen" db:"last_seen"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	IsOnline  bool      `json:"is_online,omitempty" db:"-"`
}

type UserSession struct {
	UserID     string    `json:"user_id" db:"user_id"`
	SessionID  string    `json:"session_id" db:"session_id"`
	DeviceInfo string    `json:"device_info,omitempty" db:"device_info"`
	IPAddress  string    `json:"ip_address,omitempty" db:"ip_address"`
	LastActive time.Time `json:"last_active" db:"last_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	IsActive   bool      `json:"is_active" db:"is_active"`
}

type Contact struct {
	UserID      string    `json:"user_id" db:"user_id"`
	ContactID   string    `json:"contact_id" db:"contact_id"`
	DisplayName string    `json:"display_name,omitempty" db:"display_name"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type UserPresence struct {
	UserID   string    `json:"user_id"`
	IsOnline bool      `json:"is_online"`
	LastSeen time.Time `json:"last_seen,omitempty"`
}

type AuthRequest struct {
	Phone    string `json:"phone"`
	Name     string `json:"name"`
	DeviceID string `json:"device_id,omitempty"`
}

type AuthResponse struct {
	Token     string    `json:"token"`
	User      User      `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

type UserUpdateRequest struct {
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

type SearchUserRequest struct {
	Phone string `json:"phone"`
	Name  string `json:"name,omitempty"`
}
