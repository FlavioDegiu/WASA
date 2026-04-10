package database

import (
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
