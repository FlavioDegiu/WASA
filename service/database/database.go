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

	"github.com/gofrs/uuid"
)

type User struct {
	ID       string
	Username string
	Photo    string
}

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	Ping() error

	CreateUser(username string) (User, error)
	GetUserByUsername(username string) (User, error)
	GetUserByID(id string) (User, error)
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

	return &appdbimpl{
		c: db,
	}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}

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

	return User{
		ID:       userID,
		Username: username,
		Photo:    "",
	}, nil
}

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
