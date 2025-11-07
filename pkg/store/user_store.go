package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/msniranjan18/chit-chat/pkg/models"
)

func (s *Store) CreateUser(user *models.User) error {
	s.logger.Info("Creating user", "phone", user.Phone, "name", user.Name)

	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.LastSeen = time.Now()

	query := `
		INSERT INTO users (id, phone, name, status, last_seen, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	err := s.DB.QueryRow(
		query,
		user.ID, user.Phone, user.Name, user.Status,
		user.LastSeen, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		s.logger.Error("Failed to create user",
			"error", err, "phone", user.Phone, "name", user.Name)
		return err
	}

	s.logger.Info("User created successfully", "user_id", user.ID, "phone", user.Phone)
	return nil
}

func (s *Store) GetUserByID(userID string) (*models.User, error) {
	s.logger.Debug("Getting user by ID", "user_id", userID)

	query := `
		SELECT id, phone, name, status, avatar_url, last_seen, created_at, updated_at
		FROM users WHERE id = $1`

	user := &models.User{}
	err := s.DB.QueryRow(query, userID).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Status,
		&user.AvatarURL, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		s.logger.Debug("User not found by ID", "user_id", userID)
		return nil, nil
	}
	if err != nil {
		s.logger.Error("Failed to get user by ID", "error", err, "user_id", userID)
		return nil, err
	}

	s.logger.Debug("User retrieved by ID", "user_id", userID, "name", user.Name)
	return user, nil
}

func (s *Store) GetUserByPhone(phone string) (*models.User, error) {
	s.logger.Debug("Getting user by phone", "phone", phone)

	query := `
		SELECT id, phone, name, status, avatar_url, last_seen, created_at, updated_at
		FROM users WHERE phone = $1`

	user := &models.User{}
	err := s.DB.QueryRow(query, phone).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Status,
		&user.AvatarURL, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		s.logger.Debug("User not found by phone", "phone", phone)
		return nil, nil
	}
	if err != nil {
		s.logger.Error("Failed to get user by phone", "error", err, "phone", phone)
		return nil, err
	}

	s.logger.Debug("User retrieved by phone", "user_id", user.ID, "phone", phone)
	return user, nil
}

func (s *Store) UpdateUser(userID string, updates *models.UserUpdateRequest) error {
	s.logger.Info("Updating user", "user_id", userID, "updates", updates)

	query := `
		UPDATE users 
		SET name = COALESCE($2, name),
			status = COALESCE($3, status),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id`

	err := s.DB.QueryRow(query, userID, updates.Name, updates.Status).Scan(&userID)
	if err != nil {
		s.logger.Error("Failed to update user", "error", err, "user_id", userID)
		return err
	}

	s.logger.Info("User updated successfully", "user_id", userID)
	return nil
}

func (s *Store) UpdateUserLastSeen(userID string, lastSeen time.Time) error {
	s.logger.Debug("Updating user last seen", "user_id", userID, "last_seen", lastSeen)

	query := `UPDATE users SET last_seen = $1 WHERE id = $2`
	_, err := s.DB.Exec(query, lastSeen, userID)
	if err != nil {
		s.logger.Error("Failed to update user last seen",
			"error", err, "user_id", userID, "last_seen", lastSeen)
		return err
	}

	s.logger.Debug("User last seen updated", "user_id", userID, "last_seen", lastSeen)
	return nil
}

func (s *Store) SearchUsers(queryStr string, limit int) ([]models.User, error) {
	s.logger.Info("Searching users", "query", queryStr, "limit", limit)

	query := `
		SELECT id, phone, name, status, avatar_url, last_seen, created_at, updated_at
		FROM users 
		WHERE phone ILIKE $1 OR name ILIKE $1
		ORDER BY name
		LIMIT $2`

	rows, err := s.DB.Query(query, "%"+queryStr+"%", limit)
	if err != nil {
		s.logger.Error("Failed to search users", "error", err, "query", queryStr)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		// Use sql.NullString for nullable columns
		var avatarURL sql.NullString

		err := rows.Scan(
			&user.ID, &user.Phone, &user.Name, &user.Status,
			&avatarURL, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan user row", "error", err)
			return nil, err
		}

		// Convert NullString to regular string
		if avatarURL.Valid {
			user.AvatarURL = &avatarURL.String
		}

		users = append(users, user)
	}

	s.logger.Info("User search completed",
		"query", queryStr, "results", len(users), "limit", limit)
	return users, nil
}

func (s *Store) CreateUserSession(userID, sessionID, deviceInfo, ipAddress string) error {
	s.logger.Info("Creating user session",
		"user_id", userID, "session_id", sessionID,
		"device_info", deviceInfo[:min(len(deviceInfo), 50)])

	query := `
		INSERT INTO user_sessions (user_id, session_id, device_info, ip_address)
		VALUES ($1, $2, $3, $4)`

	_, err := s.DB.Exec(query, userID, sessionID, deviceInfo, ipAddress)
	if err != nil {
		s.logger.Error("Failed to create user session",
			"error", err, "user_id", userID, "session_id", sessionID)
		return err
	}

	s.logger.Info("User session created successfully",
		"user_id", userID, "session_id", sessionID)
	return nil
}

func (s *Store) GetUserSession(sessionID string) (*models.UserSession, error) {
	s.logger.Debug("Getting user session", "session_id", sessionID)

	query := `
		SELECT user_id, session_id, device_info, ip_address, last_active, created_at, is_active
		FROM user_sessions WHERE session_id = $1`

	session := &models.UserSession{}
	err := s.DB.QueryRow(query, sessionID).Scan(
		&session.UserID, &session.SessionID, &session.DeviceInfo,
		&session.IPAddress, &session.LastActive, &session.CreatedAt, &session.IsActive,
	)

	if err == sql.ErrNoRows {
		s.logger.Debug("User session not found", "session_id", sessionID)
		return nil, nil
	}
	if err != nil {
		s.logger.Error("Failed to get user session",
			"error", err, "session_id", sessionID)
		return nil, err
	}

	s.logger.Debug("User session retrieved",
		"session_id", sessionID, "user_id", session.UserID, "is_active", session.IsActive)
	return session, nil
}

func (s *Store) UpdateSessionActivity(sessionID string) error {
	s.logger.Debug("Updating session activity", "session_id", sessionID)

	query := `UPDATE user_sessions SET last_active = CURRENT_TIMESTAMP WHERE session_id = $1`
	_, err := s.DB.Exec(query, sessionID)
	if err != nil {
		s.logger.Error("Failed to update session activity",
			"error", err, "session_id", sessionID)
		return err
	}

	s.logger.Debug("Session activity updated", "session_id", sessionID)
	return nil
}

func (s *Store) DeleteSession(sessionID string) error {
	s.logger.Info("Deleting session", "session_id", sessionID)

	query := `DELETE FROM user_sessions WHERE session_id = $1`
	_, err := s.DB.Exec(query, sessionID)
	if err != nil {
		s.logger.Error("Failed to delete session", "error", err, "session_id", sessionID)
		return err
	}

	s.logger.Info("Session deleted successfully", "session_id", sessionID)
	return nil
}

func (s *Store) AddContact(userID, contactID, displayName string) error {
	s.logger.Info("Adding contact",
		"user_id", userID, "contact_id", contactID, "display_name", displayName)

	query := `
		INSERT INTO contacts (user_id, contact_id, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, contact_id) DO UPDATE
		SET display_name = EXCLUDED.display_name`

	_, err := s.DB.Exec(query, userID, contactID, displayName)
	if err != nil {
		s.logger.Error("Failed to add contact",
			"error", err, "user_id", userID, "contact_id", contactID)
		return err
	}

	s.logger.Info("Contact added successfully",
		"user_id", userID, "contact_id", contactID)
	return nil
}

func (s *Store) GetContacts(userID string) ([]models.User, error) {
	s.logger.Debug("Getting contacts", "user_id", userID)

	query := `
		SELECT u.id, u.phone, u.name, u.status, u.avatar_url, u.last_seen, u.created_at, u.updated_at
		FROM contacts c
		JOIN users u ON c.contact_id = u.id
		WHERE c.user_id = $1
		ORDER BY u.name`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		s.logger.Error("Failed to get contacts", "error", err, "user_id", userID)
		return nil, err
	}
	defer rows.Close()

	var contacts []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Phone, &user.Name, &user.Status,
			&user.AvatarURL, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan contact row", "error", err, "user_id", userID)
			return nil, err
		}
		contacts = append(contacts, user)
	}

	s.logger.Debug("Contacts retrieved", "user_id", userID, "contact_count", len(contacts))
	return contacts, nil
}

func (s *Store) RemoveContact(userID, contactID string) error {
	s.logger.Info("Removing contact", "user_id", userID, "contact_id", contactID)

	query := `DELETE FROM contacts WHERE user_id = $1 AND contact_id = $2`
	_, err := s.DB.Exec(query, userID, contactID)
	if err != nil {
		s.logger.Error("Failed to remove contact",
			"error", err, "user_id", userID, "contact_id", contactID)
		return err
	}

	s.logger.Info("Contact removed successfully",
		"user_id", userID, "contact_id", contactID)
	return nil
}

func (s *Store) GetUsersByIDs(userIDs []string) ([]models.User, error) {
	s.logger.Debug("Getting users by IDs", "user_count", len(userIDs))

	if len(userIDs) == 0 {
		s.logger.Debug("Empty user IDs list provided")
		return []models.User{}, nil
	}

	query := `
		SELECT id, phone, name, status, avatar_url, last_seen, created_at, updated_at
		FROM users 
		WHERE id = ANY($1)`

	rows, err := s.DB.Query(query, pq.Array(userIDs))
	if err != nil {
		s.logger.Error("Failed to get users by IDs",
			"error", err, "user_count", len(userIDs))
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Phone, &user.Name, &user.Status,
			&user.AvatarURL, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan user row in GetUsersByIDs", "error", err)
			return nil, err
		}
		users = append(users, user)
	}

	s.logger.Debug("Users retrieved by IDs", "requested", len(userIDs), "found", len(users))
	return users, nil
}

func (s *Store) GetOnlineUsers() ([]string, error) {
	s.logger.Debug("Getting online users")

	// Check Redis for online users
	pattern := "presence:*"
	keys, err := s.RDB.Keys(s.Ctx, pattern).Result()
	if err != nil {
		s.logger.Error("Failed to get online users from Redis", "error", err)
		return nil, err
	}

	var userIDs []string
	for _, key := range keys {
		// Extract user ID from key
		if len(key) > 9 { // "presence:".length = 9
			userID := key[9:]
			userIDs = append(userIDs, userID)
		}
	}

	s.logger.Debug("Online users retrieved", "count", len(userIDs))
	return userIDs, nil
}
