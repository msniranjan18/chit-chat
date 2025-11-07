package auth

import (
	"context"
	"errors"
	"time"
)

type SessionStore interface {
	CreateSession(ctx context.Context, userID, sessionID, deviceInfo, ipAddress string) error
	GetSession(ctx context.Context, sessionID string) (Session, error)
	UpdateSessionActivity(ctx context.Context, sessionID string) error
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteUserSessions(ctx context.Context, userID string) error
	ListUserSessions(ctx context.Context, userID string) ([]Session, error)
}

type Session struct {
	UserID     string    `json:"user_id"`
	SessionID  string    `json:"session_id"`
	DeviceInfo string    `json:"device_info"`
	IPAddress  string    `json:"ip_address"`
	LastActive time.Time `json:"last_active"`
	CreatedAt  time.Time `json:"created_at"`
	IsActive   bool      `json:"is_active"`
}

// InMemorySessionStore is a simple in-memory implementation
// In production, use Redis or database-backed store
type InMemorySessionStore struct {
	sessions map[string]Session
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]Session),
	}
}

func (s *InMemorySessionStore) CreateSession(ctx context.Context, userID, sessionID, deviceInfo, ipAddress string) error {
	session := Session{
		UserID:     userID,
		SessionID:  sessionID,
		DeviceInfo: deviceInfo,
		IPAddress:  ipAddress,
		LastActive: time.Now(),
		CreatedAt:  time.Now(),
		IsActive:   true,
	}
	s.sessions[sessionID] = session
	return nil
}

func (s *InMemorySessionStore) GetSession(ctx context.Context, sessionID string) (Session, error) {
	session, exists := s.sessions[sessionID]
	if !exists {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (s *InMemorySessionStore) UpdateSessionActivity(ctx context.Context, sessionID string) error {
	session, exists := s.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}
	session.LastActive = time.Now()
	s.sessions[sessionID] = session
	return nil
}

func (s *InMemorySessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	delete(s.sessions, sessionID)
	return nil
}

func (s *InMemorySessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	for sessionID, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, sessionID)
		}
	}
	return nil
}

func (s *InMemorySessionStore) ListUserSessions(ctx context.Context, userID string) ([]Session, error) {
	var userSessions []Session
	for _, session := range s.sessions {
		if session.UserID == userID {
			userSessions = append(userSessions, session)
		}
	}
	return userSessions, nil
}

var (
	ErrSessionNotFound = errors.New("session not found")
)
