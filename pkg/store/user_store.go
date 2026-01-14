package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/msniranjan18/chit-chat/pkg/models"
)

func (s *Store) CreateUser(user *models.User) error {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.LastSeen = time.Now()

	query := `
		INSERT INTO users (id, phone, name, status, last_seen, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	return s.DB.QueryRow(
		query,
		user.ID, user.Phone, user.Name, user.Status,
		user.LastSeen, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)
}

func (s *Store) GetUserByID(userID string) (*models.User, error) {
	query := `
		SELECT id, phone, name, status, avatar_url, last_seen, created_at, updated_at
		FROM users WHERE id = $1`

	user := &models.User{}
	err := s.DB.QueryRow(query, userID).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Status,
		&user.AvatarURL, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Store) GetUserByPhone(phone string) (*models.User, error) {
	query := `
		SELECT id, phone, name, status, avatar_url, last_seen, created_at, updated_at
		FROM users WHERE phone = $1`

	user := &models.User{}
	err := s.DB.QueryRow(query, phone).Scan(
		&user.ID, &user.Phone, &user.Name, &user.Status,
		&user.AvatarURL, &user.LastSeen, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Store) UpdateUser(userID string, updates *models.UserUpdateRequest) error {
	query := `
		UPDATE users 
		SET name = COALESCE($2, name),
			status = COALESCE($3, status),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id`

	return s.DB.QueryRow(query, userID, updates.Name, updates.Status).Scan(&userID)
}

func (s *Store) UpdateUserLastSeen(userID string, lastSeen time.Time) error {
	query := `UPDATE users SET last_seen = $1 WHERE id = $2`
	_, err := s.DB.Exec(query, lastSeen, userID)
	return err
}

func (s *Store) SearchUsers(queryStr string, limit int) ([]models.User, error) {
	query := `
		SELECT id, phone, name, status, avatar_url, last_seen, created_at, updated_at
		FROM users 
		WHERE phone ILIKE $1 OR name ILIKE $1
		ORDER BY name
		LIMIT $2`

	rows, err := s.DB.Query(query, "%"+queryStr+"%", limit)
	if err != nil {
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
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (s *Store) CreateUserSession(userID, sessionID, deviceInfo, ipAddress string) error {
	query := `
		INSERT INTO user_sessions (user_id, session_id, device_info, ip_address)
		VALUES ($1, $2, $3, $4)`

	_, err := s.DB.Exec(query, userID, sessionID, deviceInfo, ipAddress)
	return err
}

func (s *Store) GetUserSession(sessionID string) (*models.UserSession, error) {
	query := `
		SELECT user_id, session_id, device_info, ip_address, last_active, created_at, is_active
		FROM user_sessions WHERE session_id = $1`

	session := &models.UserSession{}
	err := s.DB.QueryRow(query, sessionID).Scan(
		&session.UserID, &session.SessionID, &session.DeviceInfo,
		&session.IPAddress, &session.LastActive, &session.CreatedAt, &session.IsActive,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Store) UpdateSessionActivity(sessionID string) error {
	query := `UPDATE user_sessions SET last_active = CURRENT_TIMESTAMP WHERE session_id = $1`
	_, err := s.DB.Exec(query, sessionID)
	return err
}

func (s *Store) DeleteSession(sessionID string) error {
	query := `DELETE FROM user_sessions WHERE session_id = $1`
	_, err := s.DB.Exec(query, sessionID)
	return err
}

func (s *Store) AddContact(userID, contactID, displayName string) error {
	query := `
		INSERT INTO contacts (user_id, contact_id, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, contact_id) DO UPDATE
		SET display_name = EXCLUDED.display_name`

	_, err := s.DB.Exec(query, userID, contactID, displayName)
	return err
}

func (s *Store) GetContacts(userID string) ([]models.User, error) {
	query := `
		SELECT u.id, u.phone, u.name, u.status, u.avatar_url, u.last_seen, u.created_at, u.updated_at
		FROM contacts c
		JOIN users u ON c.contact_id = u.id
		WHERE c.user_id = $1
		ORDER BY u.name`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
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
			return nil, err
		}
		contacts = append(contacts, user)
	}

	return contacts, nil
}

func (s *Store) RemoveContact(userID, contactID string) error {
	query := `DELETE FROM contacts WHERE user_id = $1 AND contact_id = $2`
	_, err := s.DB.Exec(query, userID, contactID)
	return err
}

func (s *Store) GetUsersByIDs(userIDs []string) ([]models.User, error) {
	if len(userIDs) == 0 {
		return []models.User{}, nil
	}

	query := `
		SELECT id, phone, name, status, avatar_url, last_seen, created_at, updated_at
		FROM users 
		WHERE id = ANY($1)`

	rows, err := s.DB.Query(query, pq.Array(userIDs))
	if err != nil {
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
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (s *Store) GetOnlineUsers() ([]string, error) {
	// Check Redis for online users
	pattern := "presence:*"
	keys, err := s.RDB.Keys(s.Ctx, pattern).Result()
	if err != nil {
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

	return userIDs, nil
}
