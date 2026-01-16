package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/msniranjan18/chit-chat/pkg/models"
)

func (s *Store) SaveMessage(
	chatID, senderID, content, contentType string,
	replyTo, forwardFrom *string,
	forwarded bool,
) (*models.Message, error) {
	s.logger.Info("Saving message",
		"chat_id", chatID, "sender_id", senderID, "content_type", contentType,
		"has_reply", replyTo != nil, "forwarded", forwarded)

	messageID := uuid.New().String()
	now := time.Now()

	message := &models.Message{
		ID:          messageID,
		ChatID:      chatID,
		SenderID:    senderID,
		Content:     content,
		ContentType: contentType,
		Status:      string(models.MessageStatusSent),
		SentAt:      now,
		ReplyTo:     replyTo,
		Forwarded:   forwarded,
		ForwardFrom: forwardFrom,
		IsEdited:    false,
		IsDeleted:   false,
	}

	// Start transaction
	tx, err := s.DB.Begin()
	if err != nil {
		s.logger.Error("Failed to begin transaction for SaveMessage", "error", err)
		return nil, err
	}
	defer tx.Rollback()

	// Save message
	query := `
		INSERT INTO messages (id, chat_id, sender_id, content, content_type, status, sent_at, reply_to, forwarded, forward_from)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	err = tx.QueryRow(
		query,
		message.ID, message.ChatID, message.SenderID,
		message.Content, message.ContentType, message.Status,
		message.SentAt, message.ReplyTo, message.Forwarded, message.ForwardFrom,
	).Scan(&message.ID)

	if err != nil {
		s.logger.Error("Failed to insert message",
			"error", err, "chat_id", chatID, "sender_id", senderID)
		return nil, err
	}

	s.logger.Debug("Message inserted in database", "message_id", messageID)

	// Get chat members
	members, err := s.GetChatMembers(chatID)
	if err != nil {
		s.logger.Error("Failed to get chat members for SaveMessage",
			"error", err, "chat_id", chatID)
		return nil, err
	}

	s.logger.Debug("Setting message status for members",
		"message_id", messageID, "member_count", len(members))

	// Set initial status for each member
	for _, member := range members {
		status := string(models.MessageStatusSent)
		if member.UserID == senderID {
			status = string(models.MessageStatusDelivered)
		}

		_, err = tx.Exec(`
			INSERT INTO message_status (message_id, user_id, status, updated_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (message_id, user_id) DO UPDATE
			SET status = EXCLUDED.status, updated_at = EXCLUDED.updated_at`,
			message.ID, member.UserID, status, now,
		)
		if err != nil {
			s.logger.Error("Failed to set message status for member",
				"error", err, "message_id", messageID, "user_id", member.UserID)
			return nil, err
		}
	}

	// Mark as delivered for sender
	_, err = tx.Exec(`
		UPDATE messages 
		SET delivered_at = $1
		WHERE id = $2 AND delivered_at IS NULL`,
		now, message.ID,
	)
	if err != nil {
		s.logger.Error("Failed to mark message as delivered",
			"error", err, "message_id", messageID)
		return nil, err
	}

	// Update chat last activity
	_, err = tx.Exec(`UPDATE chats SET last_activity = $1 WHERE id = $2`, now, chatID)
	if err != nil {
		s.logger.Error("Failed to update chat last activity",
			"error", err, "chat_id", chatID)
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction for SaveMessage", "error", err)
		return nil, err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	s.logger.Info("Message saved successfully",
		"message_id", messageID, "chat_id", chatID, "sender_id", senderID)
	return message, nil
}

func (s *Store) GetMessage(messageID string) (*models.Message, error) {
	s.logger.Debug("Getting message", "message_id", messageID)

	query := `
		SELECT id, chat_id, sender_id, content, content_type, media_url, thumbnail_url, file_size, duration,
		       status, sent_at, delivered_at, read_at, reply_to, forwarded, forward_from,
		       is_edited, edited_at, is_deleted, deleted_at
		FROM messages WHERE id = $1`

	message := &models.Message{}
	err := s.DB.QueryRow(query, messageID).Scan(
		&message.ID, &message.ChatID, &message.SenderID,
		&message.Content, &message.ContentType, &message.MediaURL,
		&message.ThumbnailURL, &message.FileSize, &message.Duration,
		&message.Status, &message.SentAt, &message.DeliveredAt,
		&message.ReadAt, &message.ReplyTo, &message.Forwarded,
		&message.ForwardFrom, &message.IsEdited, &message.EditedAt,
		&message.IsDeleted, &message.DeletedAt,
	)

	if err == sql.ErrNoRows {
		s.logger.Debug("Message not found", "message_id", messageID)
		return nil, nil
	}
	if err != nil {
		s.logger.Error("Failed to get message", "error", err, "message_id", messageID)
		return nil, err
	}

	s.logger.Debug("Message retrieved", "message_id", messageID, "chat_id", message.ChatID)
	return message, nil
}

func (s *Store) GetMessages(chatID string, offset, limit int) ([]models.Message, error) {
	s.logger.Debug("Getting messages",
		"chat_id", chatID, "offset", offset, "limit", limit)

	// Try cache first
	if cached, err := s.GetCachedChatMessages(chatID); err == nil && cached != nil {
		if offset == 0 && len(cached) <= limit {
			s.logger.Debug("Retrieved messages from cache",
				"chat_id", chatID, "message_count", len(cached))
			return cached, nil
		}
	}

	query := `
		SELECT id, chat_id, sender_id, content, content_type, media_url, thumbnail_url, file_size, duration,
		       status, sent_at, delivered_at, read_at, reply_to, forwarded, forward_from,
		       is_edited, edited_at, is_deleted, deleted_at
		FROM messages 
		WHERE chat_id = $1 AND is_deleted = FALSE
		ORDER BY sent_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.DB.Query(query, chatID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to query messages",
			"error", err, "chat_id", chatID, "offset", offset, "limit", limit)
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var message models.Message
		err := rows.Scan(
			&message.ID, &message.ChatID, &message.SenderID,
			&message.Content, &message.ContentType, &message.MediaURL,
			&message.ThumbnailURL, &message.FileSize, &message.Duration,
			&message.Status, &message.SentAt, &message.DeliveredAt,
			&message.ReadAt, &message.ReplyTo, &message.Forwarded,
			&message.ForwardFrom, &message.IsEdited, &message.EditedAt,
			&message.IsDeleted, &message.DeletedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan message row",
				"error", err, "chat_id", chatID)
			return nil, err
		}
		messages = append(messages, message)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	s.logger.Debug("Retrieved messages from database",
		"chat_id", chatID, "message_count", len(messages))

	// Cache first page
	if offset == 0 {
		go s.CacheChatMessages(chatID, messages)
	}

	return messages, nil
}

func (s *Store) UpdateMessageStatus(messageID, userID, status string) error {
	s.logger.Info("Updating message status",
		"message_id", messageID, "user_id", userID, "status", status)

	tx, err := s.DB.Begin()
	if err != nil {
		s.logger.Error("Failed to begin transaction for UpdateMessageStatus", "error", err)
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	// Update message_status table
	query := `
		INSERT INTO message_status (message_id, user_id, status, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id, user_id) DO UPDATE
		SET status = EXCLUDED.status, updated_at = EXCLUDED.updated_at`

	_, err = tx.Exec(query, messageID, userID, status, now)
	if err != nil {
		s.logger.Error("Failed to update message status in table",
			"error", err, "message_id", messageID, "user_id", userID)
		return err
	}

	// Update messages table timestamps
	if status == string(models.MessageStatusDelivered) {
		_, err = tx.Exec(`
			UPDATE messages 
			SET delivered_at = COALESCE(delivered_at, $1)
			WHERE id = $2`,
			now, messageID,
		)
		if err != nil {
			s.logger.Error("Failed to update delivered_at timestamp",
				"error", err, "message_id", messageID)
			return err
		}
	} else if status == string(models.MessageStatusRead) {
		_, err = tx.Exec(`
			UPDATE messages 
			SET read_at = COALESCE(read_at, $1)
			WHERE id = $2`,
			now, messageID,
		)
		if err != nil {
			s.logger.Error("Failed to update read_at timestamp",
				"error", err, "message_id", messageID)
			return err
		}
	}

	// Get message to update chat last activity
	var chatID string
	err = tx.QueryRow("SELECT chat_id FROM messages WHERE id = $1", messageID).Scan(&chatID)
	if err != nil {
		s.logger.Error("Failed to get chat_id for message",
			"error", err, "message_id", messageID)
		return err
	}

	// Update member's last read time
	_, err = tx.Exec(`
		UPDATE chat_members 
		SET last_read_at = $1 
		WHERE chat_id = $2 AND user_id = $3`,
		now, chatID, userID,
	)
	if err != nil {
		s.logger.Error("Failed to update member last read time",
			"error", err, "chat_id", chatID, "user_id", userID)
		return err
	}

	if err = tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction for UpdateMessageStatus", "error", err)
		return err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	s.logger.Info("Message status updated successfully",
		"message_id", messageID, "user_id", userID, "status", status)
	return nil
}

func (s *Store) UpdateMessageContent(messageID, content string) error {
	s.logger.Info("Updating message content", "message_id", messageID)

	query := `
		UPDATE messages 
		SET content = $1, is_edited = TRUE, edited_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING chat_id`

	var chatID string
	err := s.DB.QueryRow(query, content, messageID).Scan(&chatID)
	if err != nil {
		s.logger.Error("Failed to update message content",
			"error", err, "message_id", messageID)
		return err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	s.logger.Info("Message content updated successfully", "message_id", messageID)
	return nil
}

func (s *Store) DeleteMessage(messageID string) error {
	s.logger.Warn("Deleting message", "message_id", messageID)

	query := `
		UPDATE messages 
		SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING chat_id`

	var chatID string
	err := s.DB.QueryRow(query, messageID).Scan(&chatID)
	if err != nil {
		s.logger.Error("Failed to delete message", "error", err, "message_id", messageID)
		return err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	s.logger.Info("Message deleted successfully", "message_id", messageID)
	return nil
}

func (s *Store) GetMessageStatus(messageID, userID string) (string, error) {
	s.logger.Debug("Getting message status", "message_id", messageID, "user_id", userID)

	query := `SELECT status FROM message_status WHERE message_id = $1 AND user_id = $2`
	var status string
	err := s.DB.QueryRow(query, messageID, userID).Scan(&status)
	if err == sql.ErrNoRows {
		s.logger.Debug("Message status not found", "message_id", messageID, "user_id", userID)
		return "", nil
	}
	if err != nil {
		s.logger.Error("Failed to get message status",
			"error", err, "message_id", messageID, "user_id", userID)
		return "", err
	}

	s.logger.Debug("Message status retrieved",
		"message_id", messageID, "user_id", userID, "status", status)
	return status, nil
}

func (s *Store) GetUnreadMessagesCount(chatID, userID string) (int, error) {
	s.logger.Debug("Getting unread messages count", "chat_id", chatID, "user_id", userID)

	query := `
		SELECT COUNT(*) 
		FROM messages m
		LEFT JOIN message_status ms ON m.id = ms.message_id AND ms.user_id = $2
		WHERE m.chat_id = $1 
		AND m.sent_at > (SELECT last_read_at FROM chat_members WHERE chat_id = $1 AND user_id = $2)
		AND (ms.status IS NULL OR ms.status != 'read')`

	var count int
	err := s.DB.QueryRow(query, chatID, userID).Scan(&count)
	if err != nil {
		s.logger.Error("Failed to get unread messages count",
			"error", err, "chat_id", chatID, "user_id", userID)
		return 0, err
	}

	s.logger.Debug("Unread messages count retrieved",
		"chat_id", chatID, "user_id", userID, "count", count)
	return count, nil
}

func (s *Store) MarkChatAsRead(chatID, userID string) error {
	s.logger.Info("Marking chat as read", "chat_id", chatID, "user_id", userID)

	now := time.Now()

	tx, err := s.DB.Begin()
	if err != nil {
		s.logger.Error("Failed to begin transaction for MarkChatAsRead", "error", err)
		return err
	}
	defer tx.Rollback()

	// Update member's last read time
	_, err = tx.Exec(`
		UPDATE chat_members 
		SET last_read_at = $1 
		WHERE chat_id = $2 AND user_id = $3`,
		now, chatID, userID,
	)
	if err != nil {
		s.logger.Error("Failed to update member last read time",
			"error", err, "chat_id", chatID, "user_id", userID)
		return err
	}

	// Update message status for all unread messages
	result, err := tx.Exec(`
		INSERT INTO message_status (message_id, user_id, status, updated_at)
		SELECT m.id, $1, 'read', $2
		FROM messages m
		WHERE m.chat_id = $3 
		AND m.sent_at <= $2
		AND NOT EXISTS (
			SELECT 1 FROM message_status ms 
			WHERE ms.message_id = m.id AND ms.user_id = $1 AND ms.status = 'read'
		)
		ON CONFLICT (message_id, user_id) DO UPDATE
		SET status = 'read', updated_at = EXCLUDED.updated_at`,
		userID, now, chatID,
	)
	if err != nil {
		s.logger.Error("Failed to update message statuses",
			"error", err, "chat_id", chatID, "user_id", userID)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	s.logger.Debug("Updated message statuses", "rows_affected", rowsAffected)

	// Update messages read_at timestamp
	_, err = tx.Exec(`
		UPDATE messages m
		SET read_at = COALESCE(read_at, $1)
		FROM (
			SELECT DISTINCT ms.message_id
			FROM message_status ms
			WHERE ms.user_id = $2 
			AND ms.status = 'read'
		) AS read_msgs
		WHERE m.id = read_msgs.message_id
		AND m.read_at IS NULL`,
		now, userID,
	)
	if err != nil {
		s.logger.Error("Failed to update messages read_at timestamp",
			"error", err, "chat_id", chatID, "user_id", userID)
		return err
	}

	if err = tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction for MarkChatAsRead", "error", err)
		return err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	s.logger.Info("Chat marked as read successfully", "chat_id", chatID, "user_id", userID)
	return nil
}

func (s *Store) SearchMessages(chatID, queryStr string, limit int) ([]models.Message, error) {
	s.logger.Info("Searching messages",
		"chat_id", chatID, "query", queryStr, "limit", limit)

	searchQuery := `
		SELECT id, chat_id, sender_id, content, content_type, media_url, thumbnail_url, file_size, duration,
		       status, sent_at, delivered_at, read_at, reply_to, forwarded, forward_from,
		       is_edited, edited_at, is_deleted, deleted_at
		FROM messages 
		WHERE chat_id = $1 
		AND content ILIKE $2
		AND is_deleted = FALSE
		ORDER BY sent_at DESC
		LIMIT $3`

	rows, err := s.DB.Query(searchQuery, chatID, "%"+queryStr+"%", limit)
	if err != nil {
		s.logger.Error("Failed to search messages",
			"error", err, "chat_id", chatID, "query", queryStr)
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var message models.Message
		err := rows.Scan(
			&message.ID, &message.ChatID, &message.SenderID,
			&message.Content, &message.ContentType, &message.MediaURL,
			&message.ThumbnailURL, &message.FileSize, &message.Duration,
			&message.Status, &message.SentAt, &message.DeliveredAt,
			&message.ReadAt, &message.ReplyTo, &message.Forwarded,
			&message.ForwardFrom, &message.IsEdited, &message.EditedAt,
			&message.IsDeleted, &message.DeletedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan message row in search", "error", err)
			return nil, err
		}
		messages = append(messages, message)
	}

	s.logger.Info("Message search completed",
		"chat_id", chatID, "query", queryStr, "results", len(messages), "limit", limit)
	return messages, nil
}
