package models

import (
	"time"
)

type GroupSettings struct {
	ChatID              string     `json:"chat_id" db:"chat_id"`
	IsPublic            bool       `json:"is_public" db:"is_public"`
	JoinLink            *string    `json:"join_link,omitempty" db:"join_link"`
	JoinLinkExpiresAt   *time.Time `json:"join_link_expires_at,omitempty" db:"join_link_expires_at"`
	AdminsCanEdit       bool       `json:"admins_can_edit" db:"admins_can_edit"`
	MembersCanInvite    bool       `json:"members_can_invite" db:"members_can_invite"`
	SendMediaAllowed    bool       `json:"send_media_allowed" db:"send_media_allowed"`
	SendMessagesAllowed bool       `json:"send_messages_allowed" db:"send_messages_allowed"`
	SlowModeDelay       int        `json:"slow_mode_delay" db:"slow_mode_delay"` // seconds
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

type GroupInvite struct {
	ID          string     `json:"id" db:"id"`
	GroupID     string     `json:"group_id" db:"group_id"`
	InvitedBy   string     `json:"invited_by" db:"invited_by"`
	InvitedUser *string    `json:"invited_user,omitempty" db:"invited_user"`
	InviteToken string     `json:"invite_token" db:"invite_token"`
	MaxUses     *int       `json:"max_uses,omitempty" db:"max_uses"`
	UsesCount   int        `json:"uses_count" db:"uses_count"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	IsActive    bool       `json:"is_active" db:"is_active"`
}

type GroupJoinRequest struct {
	ID          string    `json:"id" db:"id"`
	GroupID     string    `json:"group_id" db:"group_id"`
	UserID      string    `json:"user_id" db:"user_id"`
	Message     *string   `json:"message,omitempty" db:"message"`
	Status      string    `json:"status" db:"status"` // pending, approved, rejected
	ProcessedBy *string   `json:"processed_by,omitempty" db:"processed_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type GroupCreateRequest struct {
	Name        string                `json:"name"`
	Description *string               `json:"description,omitempty"`
	AvatarURL   *string               `json:"avatar_url,omitempty"`
	UserIDs     []string              `json:"user_ids"`
	IsPublic    bool                  `json:"is_public,omitempty"`
	Settings    *GroupSettingsRequest `json:"settings,omitempty"`
}

type GroupSettingsRequest struct {
	IsPublic            *bool `json:"is_public,omitempty"`
	AdminsCanEdit       *bool `json:"admins_can_edit,omitempty"`
	MembersCanInvite    *bool `json:"members_can_invite,omitempty"`
	SendMediaAllowed    *bool `json:"send_media_allowed,omitempty"`
	SendMessagesAllowed *bool `json:"send_messages_allowed,omitempty"`
	SlowModeDelay       *int  `json:"slow_mode_delay,omitempty"`
}

type GroupUpdateRequest struct {
	Name        *string               `json:"name,omitempty"`
	Description *string               `json:"description,omitempty"`
	AvatarURL   *string               `json:"avatar_url,omitempty"`
	Settings    *GroupSettingsRequest `json:"settings,omitempty"`
}

type GroupInviteRequest struct {
	GroupID   string   `json:"group_id"`
	UserIDs   []string `json:"user_ids,omitempty"`
	MaxUses   *int     `json:"max_uses,omitempty"`
	ExpiresIn *int     `json:"expires_in,omitempty"` // hours
}

type GroupJoinRequestResponse struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id"`
	Action  string `json:"action"` // approve, reject
}

type GroupStats struct {
	GroupID       string    `json:"group_id"`
	TotalMembers  int       `json:"total_members"`
	ActiveMembers int       `json:"active_members"`
	MessagesToday int       `json:"messages_today"`
	MessagesWeek  int       `json:"messages_week"`
	MessagesMonth int       `json:"messages_month"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type GroupResponse struct {
	Group    Chat          `json:"group"`
	Members  []ChatMember  `json:"members"`
	Settings GroupSettings `json:"settings"`
	Stats    *GroupStats   `json:"stats,omitempty"`
}
