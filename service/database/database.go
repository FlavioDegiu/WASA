/*
Package database is the middleware between the app database and the code. All data (de)serialization (save/load) from a
persistent database are handled here. Database specific logic should never escape this package.

To use this package you need to apply migrations to the database if needed/wanted, connect to it (using the database
data source name from config), and then initialize an instance of AppDatabase from the DB connection.

For example, this code adds a parameter in `webapi` executable for the database data source name (add it to the
main.WebAPIConfiguration structure):

	DB struct {
		Filename string `conf:""`
	}

This is an example on how to migrate the DB and connect to it:

	// Start Database
	logger.Println("initializing database support")
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		logger.WithError(err).Error("error opening SQLite DB")
		return fmt.Errorf("opening SQLite: %w", err)
	}
	defer func() {
		logger.Debug("database stopping")
		_ = db.Close()
	}()

Then you can initialize the AppDatabase and pass it to the api package.
*/
package database

import (
	"database/sql"
	"errors"
	"fmt"
)

// User element
type User struct {
	ID       string
	Username string
	Photo    string
}

// Conversation element, containing Users, ID, group boolean and a photo
type Conversation struct {
	ID      string
	IsGroup bool
	Name    string
	Photo   string
	Members []User
}

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	Ping() error

	CreateUser(username string) (User, error)
	GetUserByUsername(username string) (User, error)
	GetUserByID(id string) (User, error)
	ListUsers(usernameFilter string) ([]User, error)
	UpdateUserName(id string, newUsername string) (User, error)
	UpdateUserPhoto(id string, photo string) (User, error)

	CreateConversation(currentUserID string, memberIDs []string, isGroup bool, name string, photo string) (Conversation, error)
	ListConversationsByUser(userID string) ([]Conversation, error)
}

type appdbimpl struct {
	c *sql.DB
}

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building a AppDatabase")
	}

	// Check if table exists. If not, the database is empty, and we need to create the structure
	var tableName string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='users';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `
		CREATE TABLE users (
			id TEXT NOT NULL PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			photo TEXT NOT NULL DEFAULT ''
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating database structure: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error checking database structure: %w", err)
	}

	// Create the conversations table only once, the first time the app starts on an empty DB
	var conversationsTableName string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='conversations';`).Scan(&conversationsTableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `
		CREATE TABLE conversations (
			id TEXT NOT NULL PRIMARY KEY,
			is_group INTEGER NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			photo TEXT NOT NULL DEFAULT ''
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating conversations table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error checking conversations table: %w", err)
	}

	// Many-to-many relation between conversations and users
	var membersTableName string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='conversation_members';`).Scan(&membersTableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `
		CREATE TABLE conversation_members (
			conversation_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			PRIMARY KEY (conversation_id, user_id),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating conversation_members table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error checking conversation_members table: %w", err)
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}
