package database

import (
	"database/sql"
	"fmt"
)

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
