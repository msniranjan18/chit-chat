package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/msniranjan18/chit-chat/pkg/models"
)

func (s *Store) CreateChat(chatReq *models.ChatRequest, createdBy string) (*models.Chat, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create chat
	chatID := uuid.New().String()
	now := time.Now()

	chat := &models.Chat{
		ID:           chatID,
		Type:         chatReq.Type,
		Name:         chatReq.Name,
		Description:  chatReq.Description,
		AvatarURL:    chatReq.AvatarURL,
		CreatedBy:    createdBy,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastActivity: now,
	}

	query := `
		INSERT INTO chats (id, type, name, description, avatar_url, created_by, created_at, updated_at, last_activity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	err = tx.QueryRow(
		query,
		chat.ID, chat.Type, chat.Name, chat.Description,
		chat.AvatarURL, chat.CreatedBy, chat.CreatedAt,
		chat.UpdatedAt, chat.LastActivity,
	).Scan(&chat.ID)

	if err != nil {
		return nil, err
	}

	// Add creator as member with owner role
	_, err = tx.Exec(`
		INSERT INTO chat_members (chat_id, user_id, joined_at, role, is_admin)
		VALUES ($1, $2, $3, 'owner', TRUE)`,
		chatID, createdBy, now,
	)
	if err != nil {
		return nil, err
	}

	// Add other members
	for _, userID := range chatReq.UserIDs {
		if userID == createdBy {
			continue
		}

		_, err = tx.Exec(`
			INSERT INTO chat_members (chat_id, user_id, joined_at, role)
			VALUES ($1, $2, $3, 'member')`,
			chatID, userID, now,
		)
		if err != nil {
			return nil, err
		}
	}

	// For groups, create group settings
	if chatReq.Type == models.ChatTypeGroup {
		_, err = tx.Exec(`
			INSERT INTO group_settings (chat_id, is_public)
			VALUES ($1, FALSE)`,
			chatID,
		)
		if err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.InvalidateUserChatsCache(createdBy)
	for _, userID := range chatReq.UserIDs {
		s.InvalidateUserChatsCache(userID)
	}

	return chat, nil
}

func (s *Store) GetChat(chatID string) (*models.Chat, error) {
	query := `
		SELECT id, type, name, description, avatar_url, created_by, created_at, updated_at, last_activity,
		       is_archived, is_muted, is_pinned
		FROM chats WHERE id = $1`

	chat := &models.Chat{}
	err := s.DB.QueryRow(query, chatID).Scan(
		&chat.ID, &chat.Type, &chat.Name, &chat.Description,
		&chat.AvatarURL, &chat.CreatedBy, &chat.CreatedAt,
		&chat.UpdatedAt, &chat.LastActivity, &chat.IsArchived,
		&chat.IsMuted, &chat.IsPinned,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return chat, nil
}

func (s *Store) GetDirectChat(user1ID, user2ID string) (*models.Chat, error) {
	query := `
		SELECT c.id, c.type, c.name, c.description, c.avatar_url, c.created_by, 
		       c.created_at, c.updated_at, c.last_activity,
		       c.is_archived, c.is_muted, c.is_pinned
		FROM chats c
		JOIN chat_members cm1 ON c.id = cm1.chat_id
		JOIN chat_members cm2 ON c.id = cm2.chat_id
		WHERE c.type = 'direct'
		AND cm1.user_id = $1 AND cm2.user_id = $2
		LIMIT 1`

	chat := &models.Chat{}
	err := s.DB.QueryRow(query, user1ID, user2ID).Scan(
		&chat.ID, &chat.Type, &chat.Name, &chat.Description,
		&chat.AvatarURL, &chat.CreatedBy, &chat.CreatedAt,
		&chat.UpdatedAt, &chat.LastActivity, &chat.IsArchived,
		&chat.IsMuted, &chat.IsPinned,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return chat, nil
}

func (s *Store) GetUserChats(userID string) ([]models.Chat, error) {
	// Try cache first
	if cached, err := s.GetCachedUserChats(userID); err == nil && cached != nil {
		return cached, nil
	}

	query := `
		SELECT c.id, c.type, c.name, c.description, c.avatar_url, c.created_by,
		       c.created_at, c.updated_at, c.last_activity,
		       c.is_archived, c.is_muted, c.is_pinned,
		       (SELECT COUNT(*) FROM messages m WHERE m.chat_id = c.id AND m.sent_at > cm.last_read_at) as unread_count,
		       (SELECT content FROM messages WHERE chat_id = c.id ORDER BY sent_at DESC LIMIT 1) as last_message_content,
		       (SELECT sent_at FROM messages WHERE chat_id = c.id ORDER BY sent_at DESC LIMIT 1) as last_message_time
		FROM chats c
		JOIN chat_members cm ON c.id = cm.chat_id
		WHERE cm.user_id = $1 AND c.is_archived = FALSE
		ORDER BY c.last_activity DESC`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var chat models.Chat
		var lastMessageContent sql.NullString
		var lastMessageTime sql.NullTime
		var unreadCount int

		err := rows.Scan(
			&chat.ID, &chat.Type, &chat.Name, &chat.Description,
			&chat.AvatarURL, &chat.CreatedBy, &chat.CreatedAt,
			&chat.UpdatedAt, &chat.LastActivity, &chat.IsArchived,
			&chat.IsMuted, &chat.IsPinned, &unreadCount,
			&lastMessageContent, &lastMessageTime,
		)
		if err != nil {
			return nil, err
		}

		if lastMessageContent.Valid && lastMessageTime.Valid {
			chat.LastMessage = &models.Message{
				Content: lastMessageContent.String,
				SentAt:  lastMessageTime.Time,
			}
		}
		chat.UnreadCount = unreadCount

		chats = append(chats, chat)
	}

	// Cache the result
	go s.CacheUserChats(userID, chats)

	return chats, nil
}

func (s *Store) UpdateChat(chatID string, updates *models.ChatUpdateRequest) error {
	query := `
		UPDATE chats 
		SET name = COALESCE($2, name),
			description = COALESCE($3, description),
			avatar_url = COALESCE($4, avatar_url),
			is_archived = COALESCE($5, is_archived),
			is_muted = COALESCE($6, is_muted),
			is_pinned = COALESCE($7, is_pinned),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id`

	return s.DB.QueryRow(
		query, chatID, updates.Name, updates.Description,
		updates.AvatarURL, updates.IsArchived, updates.IsMuted, updates.IsPinned,
	).Scan(&chatID)
}

func (s *Store) UpdateChatLastActivity(chatID string) error {
	query := `UPDATE chats SET last_activity = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := s.DB.Exec(query, chatID)
	return err
}

func (s *Store) DeleteChat(chatID string) error {
	query := `DELETE FROM chats WHERE id = $1`
	_, err := s.DB.Exec(query, chatID)
	return err
}

func (s *Store) GetChatMembers(chatID string) ([]models.ChatMember, error) {
	query := `
		SELECT chat_id, user_id, joined_at, last_read_at, role, is_admin, display_name, is_banned, banned_until
		FROM chat_members 
		WHERE chat_id = $1 AND is_banned = FALSE
		ORDER BY joined_at`

	rows, err := s.DB.Query(query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.ChatMember
	for rows.Next() {
		var member models.ChatMember
		err := rows.Scan(
			&member.ChatID, &member.UserID, &member.JoinedAt,
			&member.LastReadAt, &member.Role, &member.IsAdmin,
			&member.DisplayName, &member.IsBanned, &member.BannedUntil,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

func (s *Store) AddChatMember(chatID, userID string, role models.ChatMemberRole, displayName string) error {
	query := `
		INSERT INTO chat_members (chat_id, user_id, joined_at, role, display_name)
		VALUES ($1, $2, CURRENT_TIMESTAMP, $3, $4)
		ON CONFLICT (chat_id, user_id) DO UPDATE
		SET role = EXCLUDED.role,
			display_name = COALESCE(EXCLUDED.display_name, chat_members.display_name),
			is_banned = FALSE,
			banned_until = NULL`

	_, err := s.DB.Exec(query, chatID, userID, role, displayName)
	if err != nil {
		return err
	}

	// Invalidate user's chat cache
	s.InvalidateUserChatsCache(userID)

	return nil
}

func (s *Store) RemoveChatMember(chatID, userID string) error {
	query := `DELETE FROM chat_members WHERE chat_id = $1 AND user_id = $2`
	_, err := s.DB.Exec(query, chatID, userID)
	if err != nil {
		return err
	}

	// Invalidate user's chat cache
	s.InvalidateUserChatsCache(userID)

	return nil
}

func (s *Store) UpdateChatMemberRole(chatID, userID string, role models.ChatMemberRole) error {
	query := `UPDATE chat_members SET role = $3 WHERE chat_id = $1 AND user_id = $2`
	_, err := s.DB.Exec(query, chatID, userID, role)
	return err
}

func (s *Store) UpdateMemberLastRead(chatID, userID string) error {
	query := `UPDATE chat_members SET last_read_at = CURRENT_TIMESTAMP WHERE chat_id = $1 AND user_id = $2`
	_, err := s.DB.Exec(query, chatID, userID)
	return err
}

func (s *Store) IsChatMember(chatID, userID string) (bool, error) {
	query := `SELECT 1 FROM chat_members WHERE chat_id = $1 AND user_id = $2 AND is_banned = FALSE`
	var exists int
	err := s.DB.QueryRow(query, chatID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) SearchChats(queryStr string, chatType *models.ChatType, limit int) ([]models.Chat, error) {
	baseQuery := `
		SELECT id, type, name, description, avatar_url, created_by, created_at, updated_at, last_activity,
		       is_archived, is_muted, is_pinned
		FROM chats 
		WHERE (name ILIKE $1 OR description ILIKE $1) 
		AND is_archived = FALSE`

	var query string
	var args []interface{}

	if chatType != nil {
		query = baseQuery + " AND type = $2 ORDER BY last_activity DESC LIMIT $3"
		args = []interface{}{"%" + queryStr + "%", *chatType, limit}
	} else {
		query = baseQuery + " ORDER BY last_activity DESC LIMIT $2"
		args = []interface{}{"%" + queryStr + "%", limit}
	}

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var chat models.Chat
		err := rows.Scan(
			&chat.ID, &chat.Type, &chat.Name, &chat.Description,
			&chat.AvatarURL, &chat.CreatedBy, &chat.CreatedAt,
			&chat.UpdatedAt, &chat.LastActivity, &chat.IsArchived,
			&chat.IsMuted, &chat.IsPinned,
		)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}

	return chats, nil
}
