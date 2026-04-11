package database

import (
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"
)

// Create a new user
func (db *appdbimpl) CreateUser(username string) (User, error) {
	userID := "user_" + uuid.Must(uuid.NewV4()).String()

	_, err := db.c.Exec(
		`INSERT INTO users (id, username, photo) VALUES (?, ?, ?)`,
		userID,
		username,
		"",
	)
	if err != nil {
		return User{}, fmt.Errorf("error creating user: %w", err)
	}

	// New users start without a profile photo.
	return User{
		ID:       userID,
		Username: username,
		Photo:    "",
	}, nil
}

// GetUserByUsername loads a single user using the username as lookup key
func (db *appdbimpl) GetUserByUsername(username string) (User, error) {
	var user User

	err := db.c.QueryRow(
		`SELECT id, username, photo FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &user.Photo)

	if err != nil {
		return User{}, err
	}

	return user, nil
}

// GetUserByID loads a single user using the internal id
func (db *appdbimpl) GetUserByID(id string) (User, error) {
	var user User

	err := db.c.QueryRow(
		`SELECT id, username, photo FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Username, &user.Photo)

	if err != nil {
		return User{}, err
	}

	return user, nil
}

// Returns all the users in the database, optionally filtered by name
func (db *appdbimpl) ListUsers(usernameFilter string) ([]User, error) {
	var rows *sql.Rows
	var err error

	// Without a filter, return the full list ordered alphabetically.
	if usernameFilter == "" {
		rows, err = db.c.Query(
			`SELECT id, username, photo FROM users ORDER BY username ASC`,
		)
	} else {
		// Simple substring search on usernames.
		rows, err = db.c.Query(
			`SELECT id, username, photo FROM users WHERE username LIKE ? ORDER BY username ASC`,
			"%"+usernameFilter+"%",
		)
	}

	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Photo); err != nil {
			return nil, fmt.Errorf("error scanning user row: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// UpdateUserName changes the username and returns the updated record
func (db *appdbimpl) UpdateUserName(id string, newUsername string) (User, error) {
	result, err := db.c.Exec(
		`UPDATE users SET username = ? WHERE id = ?`,
		newUsername,
		id,
	)
	if err != nil {
		return User{}, fmt.Errorf("error updating username: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("error checking updated rows: %w", err)
	}
	// If no rows were touched, the target user does not exist.
	if rowsAffected == 0 {
		return User{}, sql.ErrNoRows
	}

	return db.GetUserByID(id)
}

// UpdateUserPhoto changes the profile photo and returns the updated record
func (db *appdbimpl) UpdateUserPhoto(id string, photo string) (User, error) {
	result, err := db.c.Exec(
		`UPDATE users SET photo = ? WHERE id = ?`,
		photo,
		id,
	)
	if err != nil {
		return User{}, fmt.Errorf("error updating user photo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("error checking updated rows: %w", err)
	}
	if rowsAffected == 0 {
		return User{}, sql.ErrNoRows
	}

	return db.GetUserByID(id)
}

// Returns true if the given user belongs to the given conversation.
func (db *appdbimpl) userBelongsToConversation(conversationID string, userID string) (bool, error) {
	var count int

	err := db.c.QueryRow(
		`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ?`,
		conversationID,
		userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking conversation membership: %w", err)
	}

	return count > 0, nil
}

// Returns true if the given user can access the given message
// because they belong to the conversation that owns it.
func (db *appdbimpl) userCanAccessMessage(messageID string, userID string) (bool, error) {
	conversationID, err := db.getConversationIDByMessageID(messageID)
	if err != nil {
		return false, err
	}

	return db.userBelongsToConversation(conversationID, userID)
}
