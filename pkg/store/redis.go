package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/msniranjan18/chit-chat/pkg/models"
)

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
	s.logger.Debug("Caching user presence",
		"user_id", userID,
		"presence", presence)

	data, err := json.Marshal(presence)
	if err != nil {
		s.logger.Error("Failed to marshal user presence for caching",
			"error", err,
			"user_id", userID)
		return err
	}

	key := userPresenceKey(userID)
	err = s.RDB.Set(s.Ctx, key, data, 5*time.Minute).Err()
	if err != nil {
		s.logger.Error("Failed to cache user presence in Redis",
			"error", err,
			"user_id", userID,
			"key", key,
			"ttl", "5m")
		return err
	}

	s.logger.Debug("User presence cached successfully",
		"user_id", userID,
		"key", key,
		"ttl", "5m")
	return nil
}

func (s *Store) GetCachedUserPresence(userID string) (*models.UserPresence, error) {
	s.logger.Debug("Getting cached user presence", "user_id", userID)

	key := userPresenceKey(userID)
	data, err := s.RDB.Get(s.Ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			s.logger.Debug("User presence not found in cache", "user_id", userID, "key", key)
			return nil, nil
		}
		s.logger.Error("Failed to get user presence from cache",
			"error", err,
			"user_id", userID,
			"key", key)
		return nil, err
	}

	var presence models.UserPresence
	if err := json.Unmarshal(data, &presence); err != nil {
		s.logger.Error("Failed to unmarshal user presence from cache",
			"error", err,
			"user_id", userID,
			"key", key,
			"data_length", len(data))
		return nil, err
	}

	s.logger.Debug("User presence retrieved from cache",
		"user_id", userID,
		"key", key,
		"presence", presence)
	return &presence, nil
}

func (s *Store) CacheUserChats(userID string, chats []models.Chat) error {
	s.logger.Debug("Caching user chats",
		"user_id", userID,
		"chat_count", len(chats))

	data, err := json.Marshal(chats)
	if err != nil {
		s.logger.Error("Failed to marshal user chats for caching",
			"error", err,
			"user_id", userID,
			"chat_count", len(chats))
		return err
	}

	key := userChatsKey(userID)
	err = s.RDB.Set(s.Ctx, key, data, 10*time.Minute).Err()
	if err != nil {
		s.logger.Error("Failed to cache user chats in Redis",
			"error", err,
			"user_id", userID,
			"key", key,
			"ttl", "10m",
			"chat_count", len(chats))
		return err
	}

	s.logger.Debug("User chats cached successfully",
		"user_id", userID,
		"key", key,
		"chat_count", len(chats),
		"ttl", "10m")
	return nil
}

func (s *Store) GetCachedUserChats(userID string) ([]models.Chat, error) {
	s.logger.Debug("Getting cached user chats", "user_id", userID)

	key := userChatsKey(userID)
	data, err := s.RDB.Get(s.Ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			s.logger.Debug("User chats not found in cache", "user_id", userID, "key", key)
			return nil, nil
		}
		s.logger.Error("Failed to get user chats from cache",
			"error", err,
			"user_id", userID,
			"key", key)
		return nil, err
	}

	var chats []models.Chat
	if err := json.Unmarshal(data, &chats); err != nil {
		s.logger.Error("Failed to unmarshal user chats from cache",
			"error", err,
			"user_id", userID,
			"key", key,
			"data_length", len(data))
		return nil, err
	}

	s.logger.Debug("User chats retrieved from cache",
		"user_id", userID,
		"key", key,
		"chat_count", len(chats))
	return chats, nil
}

func (s *Store) InvalidateUserChatsCache(userID string) error {
	s.logger.Debug("Invalidating user chats cache", "user_id", userID)

	key := userChatsKey(userID)
	result, err := s.RDB.Del(s.Ctx, key).Result()
	if err != nil {
		s.logger.Error("Failed to invalidate user chats cache",
			"error", err,
			"user_id", userID,
			"key", key)
		return err
	}

	s.logger.Debug("User chats cache invalidated",
		"user_id", userID,
		"key", key,
		"deleted_keys", result)
	return nil
}

func (s *Store) CacheChatMessages(chatID string, messages []models.Message) error {
	s.logger.Debug("Caching chat messages",
		"chat_id", chatID,
		"message_count", len(messages))

	data, err := json.Marshal(messages)
	if err != nil {
		s.logger.Error("Failed to marshal chat messages for caching",
			"error", err,
			"chat_id", chatID,
			"message_count", len(messages))
		return err
	}

	key := chatMessagesKey(chatID)
	err = s.RDB.Set(s.Ctx, key, data, 5*time.Minute).Err()
	if err != nil {
		s.logger.Error("Failed to cache chat messages in Redis",
			"error", err,
			"chat_id", chatID,
			"key", key,
			"ttl", "5m",
			"message_count", len(messages))
		return err
	}

	s.logger.Debug("Chat messages cached successfully",
		"chat_id", chatID,
		"key", key,
		"message_count", len(messages),
		"ttl", "5m")
	return nil
}

func (s *Store) GetCachedChatMessages(chatID string) ([]models.Message, error) {
	s.logger.Debug("Getting cached chat messages", "chat_id", chatID)

	key := chatMessagesKey(chatID)
	data, err := s.RDB.Get(s.Ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			s.logger.Debug("Chat messages not found in cache", "chat_id", chatID, "key", key)
			return nil, nil
		}
		s.logger.Error("Failed to get chat messages from cache",
			"error", err,
			"chat_id", chatID,
			"key", key)
		return nil, err
	}

	var messages []models.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		s.logger.Error("Failed to unmarshal chat messages from cache",
			"error", err,
			"chat_id", chatID,
			"key", key,
			"data_length", len(data))
		return nil, err
	}

	s.logger.Debug("Chat messages retrieved from cache",
		"chat_id", chatID,
		"key", key,
		"message_count", len(messages))
	return messages, nil
}

func (s *Store) InvalidateChatMessagesCache(chatID string) error {
	s.logger.Debug("Invalidating chat messages cache", "chat_id", chatID)

	key := chatMessagesKey(chatID)
	result, err := s.RDB.Del(s.Ctx, key).Result()
	if err != nil {
		s.logger.Error("Failed to invalidate chat messages cache",
			"error", err,
			"chat_id", chatID,
			"key", key)
		return err
	}

	s.logger.Debug("Chat messages cache invalidated",
		"chat_id", chatID,
		"key", key,
		"deleted_keys", result)
	return nil
}

// Cache chat members
func (s *Store) CacheChatMembers(chatID string, members []models.ChatMember) error {
	s.logger.Debug("Caching chat members",
		"chat_id", chatID,
		"member_count", len(members))

	data, err := json.Marshal(members)
	if err != nil {
		s.logger.Error("Failed to marshal chat members for caching",
			"error", err,
			"chat_id", chatID,
			"member_count", len(members))
		return err
	}

	key := chatMembersKey(chatID)
	err = s.RDB.Set(s.Ctx, key, data, 15*time.Minute).Err()
	if err != nil {
		s.logger.Error("Failed to cache chat members in Redis",
			"error", err,
			"chat_id", chatID,
			"key", key,
			"ttl", "15m",
			"member_count", len(members))
		return err
	}

	s.logger.Debug("Chat members cached successfully",
		"chat_id", chatID,
		"key", key,
		"member_count", len(members),
		"ttl", "15m")
	return nil
}

func (s *Store) GetCachedChatMembers(chatID string) ([]models.ChatMember, error) {
	s.logger.Debug("Getting cached chat members", "chat_id", chatID)

	key := chatMembersKey(chatID)
	data, err := s.RDB.Get(s.Ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			s.logger.Debug("Chat members not found in cache", "chat_id", chatID, "key", key)
			return nil, nil
		}
		s.logger.Error("Failed to get chat members from cache",
			"error", err,
			"chat_id", chatID,
			"key", key)
		return nil, err
	}

	var members []models.ChatMember
	if err := json.Unmarshal(data, &members); err != nil {
		s.logger.Error("Failed to unmarshal chat members from cache",
			"error", err,
			"chat_id", chatID,
			"key", key,
			"data_length", len(data))
		return nil, err
	}

	s.logger.Debug("Chat members retrieved from cache",
		"chat_id", chatID,
		"key", key,
		"member_count", len(members))
	return members, nil
}

func (s *Store) InvalidateChatMembersCache(chatID string) error {
	s.logger.Debug("Invalidating chat members cache", "chat_id", chatID)

	key := chatMembersKey(chatID)
	result, err := s.RDB.Del(s.Ctx, key).Result()
	if err != nil {
		s.logger.Error("Failed to invalidate chat members cache",
			"error", err,
			"chat_id", chatID,
			"key", key)
		return err
	}

	s.logger.Debug("Chat members cache invalidated",
		"chat_id", chatID,
		"key", key,
		"deleted_keys", result)
	return nil
}

// Cache message status
func (s *Store) CacheMessageStatus(messageID string, status map[string]string) error {
	s.logger.Debug("Caching message status",
		"message_id", messageID,
		"status_count", len(status))

	data, err := json.Marshal(status)
	if err != nil {
		s.logger.Error("Failed to marshal message status for caching",
			"error", err,
			"message_id", messageID,
			"status_count", len(status))
		return err
	}

	key := messageStatusKey(messageID)
	err = s.RDB.Set(s.Ctx, key, data, 2*time.Minute).Err()
	if err != nil {
		s.logger.Error("Failed to cache message status in Redis",
			"error", err,
			"message_id", messageID,
			"key", key,
			"ttl", "2m",
			"status_count", len(status))
		return err
	}

	s.logger.Debug("Message status cached successfully",
		"message_id", messageID,
		"key", key,
		"status_count", len(status),
		"ttl", "2m")
	return nil
}

func (s *Store) GetCachedMessageStatus(messageID string) (map[string]string, error) {
	s.logger.Debug("Getting cached message status", "message_id", messageID)

	key := messageStatusKey(messageID)
	data, err := s.RDB.Get(s.Ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			s.logger.Debug("Message status not found in cache", "message_id", messageID, "key", key)
			return nil, nil
		}
		s.logger.Error("Failed to get message status from cache",
			"error", err,
			"message_id", messageID,
			"key", key)
		return nil, err
	}

	var status map[string]string
	if err := json.Unmarshal(data, &status); err != nil {
		s.logger.Error("Failed to unmarshal message status from cache",
			"error", err,
			"message_id", messageID,
			"key", key,
			"data_length", len(data))
		return nil, err
	}

	s.logger.Debug("Message status retrieved from cache",
		"message_id", messageID,
		"key", key,
		"status_count", len(status))
	return status, nil
}

func (s *Store) InvalidateMessageStatusCache(messageID string) error {
	s.logger.Debug("Invalidating message status cache", "message_id", messageID)

	key := messageStatusKey(messageID)
	result, err := s.RDB.Del(s.Ctx, key).Result()
	if err != nil {
		s.logger.Error("Failed to invalidate message status cache",
			"error", err,
			"message_id", messageID,
			"key", key)
		return err
	}

	s.logger.Debug("Message status cache invalidated",
		"message_id", messageID,
		"key", key,
		"deleted_keys", result)
	return nil
}
