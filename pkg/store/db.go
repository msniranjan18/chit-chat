package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

type Store struct {
	DB  *sql.DB
	RDB *redis.Client
	Ctx context.Context
}

func NewStore(ctx context.Context, pgConnStr, redisAddr string) (*Store, error) {
	var db *sql.DB
	var err error

	// 1. Setup PostgreSQL
	// Retry Postgres connection 5 times
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", pgConnStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for Postgres... (attempt %d/5)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, err
	}
	// Test PostgreSQL connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Connect to Redis
	rdb := InitRedis(redisAddr)
	// Verify Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to PostgreSQL and Redis")

	return &Store{
		DB:  db,
		RDB: rdb,
		Ctx: ctx,
	}, nil
}

func (s *Store) InitSchema() error {
	schema := `
		-- Enable UUID extension
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

		-- Users table
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			phone VARCHAR(15) UNIQUE NOT NULL,
			name VARCHAR(100) NOT NULL,
			status TEXT DEFAULT 'Hey there! I am using ChitChat',
			avatar_url TEXT,
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		-- Indexes for users
		CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
		CREATE INDEX IF NOT EXISTS idx_users_last_seen ON users(last_seen);

		-- User sessions
		CREATE TABLE IF NOT EXISTS user_sessions (
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			device_info TEXT,
			ip_address INET,
			last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE
		);

		-- Contacts table
		CREATE TABLE IF NOT EXISTS contacts (
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			contact_id UUID REFERENCES users(id) ON DELETE CASCADE,
			display_name VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, contact_id)
		);

		-- Chats table
		CREATE TABLE IF NOT EXISTS chats (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			type VARCHAR(10) NOT NULL CHECK (type IN ('direct', 'group', 'channel')),
			name VARCHAR(100),
			description TEXT,
			avatar_url TEXT,
			created_by UUID REFERENCES users(id),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_archived BOOLEAN DEFAULT FALSE,
			is_muted BOOLEAN DEFAULT FALSE,
			is_pinned BOOLEAN DEFAULT FALSE
		);

		-- Indexes for chats
		CREATE INDEX IF NOT EXISTS idx_chats_type ON chats(type);
		CREATE INDEX IF NOT EXISTS idx_chats_last_activity ON chats(last_activity);
		CREATE INDEX IF NOT EXISTS idx_chats_created_by ON chats(created_by);

		-- Chat members
		CREATE TABLE IF NOT EXISTS chat_members (
			chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_read_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			role VARCHAR(10) DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
			is_admin BOOLEAN DEFAULT FALSE,
			display_name VARCHAR(100),
			is_banned BOOLEAN DEFAULT FALSE,
			banned_until TIMESTAMP,
			PRIMARY KEY (chat_id, user_id)
		);

		-- Indexes for chat members
		CREATE INDEX IF NOT EXISTS idx_chat_members_user_id ON chat_members(user_id);
		CREATE INDEX IF NOT EXISTS idx_chat_members_chat_id ON chat_members(chat_id);

		-- Messages table
		CREATE TABLE IF NOT EXISTS messages (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			sender_id UUID REFERENCES users(id),
			content TEXT NOT NULL,
			content_type VARCHAR(10) DEFAULT 'text',
			media_url TEXT,
			thumbnail_url TEXT,
			file_size BIGINT,
			duration INTEGER,
			status VARCHAR(10) DEFAULT 'sent',
			sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			delivered_at TIMESTAMP,
			read_at TIMESTAMP,
			reply_to UUID REFERENCES messages(id),
			forwarded BOOLEAN DEFAULT FALSE,
			forward_from UUID REFERENCES users(id),
			is_edited BOOLEAN DEFAULT FALSE,
			edited_at TIMESTAMP,
			is_deleted BOOLEAN DEFAULT FALSE,
			deleted_at TIMESTAMP
		);

		-- Indexes for messages
		CREATE INDEX IF NOT EXISTS idx_messages_chat_id_sent_at ON messages(chat_id, sent_at);
		CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);
		CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);

		-- Message status tracking (for group messages)
		CREATE TABLE IF NOT EXISTS message_status (
			message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			status VARCHAR(10) NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (message_id, user_id)
		);

		-- Group settings
		CREATE TABLE IF NOT EXISTS group_settings (
			chat_id UUID PRIMARY KEY REFERENCES chats(id) ON DELETE CASCADE,
			is_public BOOLEAN DEFAULT FALSE,
			join_link TEXT UNIQUE,
			join_link_expires_at TIMESTAMP,
			admins_can_edit BOOLEAN DEFAULT TRUE,
			members_can_invite BOOLEAN DEFAULT TRUE,
			send_media_allowed BOOLEAN DEFAULT TRUE,
			send_messages_allowed BOOLEAN DEFAULT TRUE,
			slow_mode_delay INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		-- Group invites
		CREATE TABLE IF NOT EXISTS group_invites (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			group_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			invited_by UUID REFERENCES users(id),
			invited_user UUID REFERENCES users(id),
			invite_token VARCHAR(100) UNIQUE NOT NULL,
			max_uses INTEGER,
			uses_count INTEGER DEFAULT 0,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE
		);

		-- Group join requests
		CREATE TABLE IF NOT EXISTS group_join_requests (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			group_id UUID REFERENCES chats(id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(id),
			message TEXT,
			status VARCHAR(10) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
			processed_by UUID REFERENCES users(id),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		-- Triggers for updated_at
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ language 'plpgsql';

		-- Apply triggers to tables
		DROP TRIGGER IF EXISTS update_users_updated_at ON users;
		CREATE TRIGGER update_users_updated_at
			BEFORE UPDATE ON users
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();

		DROP TRIGGER IF EXISTS update_chats_updated_at ON chats;
		CREATE TRIGGER update_chats_updated_at
			BEFORE UPDATE ON chats
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();

		DROP TRIGGER IF EXISTS update_group_settings_updated_at ON group_settings;
		CREATE TRIGGER update_group_settings_updated_at
			BEFORE UPDATE ON group_settings
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := s.DB.Exec(schema)
	return err
}

func (s *Store) Close() error {
	var errs []error

	if err := s.DB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("postgres close error: %w", err))
	}

	if err := s.RDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("redis close error: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing store: %v", errs)
	}
	return nil
}

func (s *Store) StartCleanupWorker(interval time.Duration, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting cleanup worker: interval=%v, maxAge=%v", interval, maxAge)

	for range ticker.C {
		// Delete expired sessions
		_, err := s.DB.Exec(`
			DELETE FROM user_sessions 
			WHERE last_active < NOW() - $1::interval
		`, maxAge.String())
		if err != nil {
			log.Printf("Error cleaning up sessions: %v", err)
		}

		// Delete expired group invites
		_, err = s.DB.Exec(`
			UPDATE group_invites 
			SET is_active = FALSE 
			WHERE expires_at < NOW()
		`)
		if err != nil {
			log.Printf("Error cleaning up group invites: %v", err)
		}

		// Archive inactive chats (no activity for 30 days)
		_, err = s.DB.Exec(`
			UPDATE chats 
			SET is_archived = TRUE 
			WHERE last_activity < NOW() - $1::interval 
			AND is_archived = FALSE
		`, (30 * 24 * time.Hour).String())
		if err != nil {
			log.Printf("Error archiving inactive chats: %v", err)
		}
	}
}
