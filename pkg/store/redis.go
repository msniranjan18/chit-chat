package store

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/msniranjan18/chit-chat/pkg/models"
)

func InitRedis(url string) *redis.Client {
	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	// Enable TLS for secure connections
	opt.TLSConfig = &tls.Config{
		InsecureSkipVerify: false,
	}

	// Set connection pool settings
	opt.PoolSize = 100
	opt.MinIdleConns = 10
	opt.MaxRetries = 3
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 3 * time.Second
	opt.WriteTimeout = 3 * time.Second
	opt.PoolTimeout = 4 * time.Second

	client := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	log.Println("Redis connected successfully")
	return client
}

// Redis cache keys
func userPresenceKey(userID string) string {
	return fmt.Sprintf("presence:%s", userID)
}

func userChatsKey(userID string) string {
	return fmt.Sprintf("chats:%s", userID)
}

func chatMessagesKey(chatID string) string {
	return fmt.Sprintf("messages:%s", chatID)
}

func chatMembersKey(chatID string) string {
	return fmt.Sprintf("chat_members:%s", chatID)
}

func messageStatusKey(messageID string) string {
	return fmt.Sprintf("msg_status:%s", messageID)
}

// Cache helpers
func (s *Store) CacheUserPresence(userID string, presence models.UserPresence) error {
	data, err := json.Marshal(presence)
	if err != nil {
		return err
	}

	return s.RDB.Set(s.Ctx, userPresenceKey(userID), data, 5*time.Minute).Err()
}

func (s *Store) GetCachedUserPresence(userID string) (*models.UserPresence, error) {
	data, err := s.RDB.Get(s.Ctx, userPresenceKey(userID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var presence models.UserPresence
	if err := json.Unmarshal(data, &presence); err != nil {
		return nil, err
	}

	return &presence, nil
}

func (s *Store) CacheUserChats(userID string, chats []models.Chat) error {
	data, err := json.Marshal(chats)
	if err != nil {
		return err
	}

	return s.RDB.Set(s.Ctx, userChatsKey(userID), data, 10*time.Minute).Err()
}

func (s *Store) GetCachedUserChats(userID string) ([]models.Chat, error) {
	data, err := s.RDB.Get(s.Ctx, userChatsKey(userID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var chats []models.Chat
	if err := json.Unmarshal(data, &chats); err != nil {
		return nil, err
	}

	return chats, nil
}

func (s *Store) InvalidateUserChatsCache(userID string) error {
	return s.RDB.Del(s.Ctx, userChatsKey(userID)).Err()
}

func (s *Store) CacheChatMessages(chatID string, messages []models.Message) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	return s.RDB.Set(s.Ctx, chatMessagesKey(chatID), data, 5*time.Minute).Err()
}

func (s *Store) GetCachedChatMessages(chatID string) ([]models.Message, error) {
	data, err := s.RDB.Get(s.Ctx, chatMessagesKey(chatID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var messages []models.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *Store) InvalidateChatMessagesCache(chatID string) error {
	return s.RDB.Del(s.Ctx, chatMessagesKey(chatID)).Err()
}
