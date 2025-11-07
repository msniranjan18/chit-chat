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
		return nil, err
	}

	// Get chat members
	members, err := s.GetChatMembers(chatID)
	if err != nil {
		return nil, err
	}

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
		return nil, err
	}

	// Update chat last activity
	_, err = tx.Exec(`UPDATE chats SET last_activity = $1 WHERE id = $2`, now, chatID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	return message, nil
}

func (s *Store) GetMessage(messageID string) (*models.Message, error) {
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
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (s *Store) GetMessages(chatID string, offset, limit int) ([]models.Message, error) {
	// Try cache first
	if cached, err := s.GetCachedChatMessages(chatID); err == nil && cached != nil {
		if offset == 0 && len(cached) <= limit {
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
			return nil, err
		}
		messages = append(messages, message)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	// Cache first page
	if offset == 0 {
		go s.CacheChatMessages(chatID, messages)
	}

	return messages, nil
}

func (s *Store) UpdateMessageStatus(messageID, userID, status string) error {
	tx, err := s.DB.Begin()
	if err != nil {
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
			return err
		}
	}

	// Get message to update chat last activity
	var chatID string
	err = tx.QueryRow("SELECT chat_id FROM messages WHERE id = $1", messageID).Scan(&chatID)
	if err != nil {
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
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	return nil
}

func (s *Store) UpdateMessageContent(messageID, content string) error {
	query := `
		UPDATE messages 
		SET content = $1, is_edited = TRUE, edited_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING chat_id`

	var chatID string
	err := s.DB.QueryRow(query, content, messageID).Scan(&chatID)
	if err != nil {
		return err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	return nil
}

func (s *Store) DeleteMessage(messageID string) error {
	query := `
		UPDATE messages 
		SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING chat_id`

	var chatID string
	err := s.DB.QueryRow(query, messageID).Scan(&chatID)
	if err != nil {
		return err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	return nil
}

func (s *Store) GetMessageStatus(messageID, userID string) (string, error) {
	query := `SELECT status FROM message_status WHERE message_id = $1 AND user_id = $2`
	var status string
	err := s.DB.QueryRow(query, messageID, userID).Scan(&status)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return status, nil
}

func (s *Store) GetUnreadMessagesCount(chatID, userID string) (int, error) {
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
		return 0, err
	}
	return count, nil
}

func (s *Store) MarkChatAsRead(chatID, userID string) error {
	now := time.Now()

	tx, err := s.DB.Begin()
	if err != nil {
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
		return err
	}

	// Update message status for all unread messages
	_, err = tx.Exec(`
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
		return err
	}

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
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	// Invalidate cache
	s.InvalidateChatMessagesCache(chatID)

	return nil
}

func (s *Store) SearchMessages(chatID, queryStr string, limit int) ([]models.Message, error) {
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
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}
