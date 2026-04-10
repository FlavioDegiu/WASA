package database

import (
	"database/sql"
	"fmt"
)

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
