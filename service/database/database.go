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

// Domain errors
var ErrGroupNotFound = errors.New("group not found (or requester is not a member of the group)")
var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyInGroup = errors.New("user already belongs to the group")
var ErrNotGroupMember = errors.New("requester is not a member of the group")
var ErrConversationNotGroup = errors.New("target conversation is not a group")
var ErrSourceMessageNotFound = errors.New("source message not found")
var ErrDestinationConversationNotFound = errors.New("destination conversation not found")
var ErrUserAlreadyReacted = errors.New("user already reacted to this message")
var ErrMessageNotOwned = errors.New("message does not belong to the authenticated user")
var ErrCommentNotOwned = errors.New("comment does not belong to the authenticated user")

/*
Core backend data models used by the DB
*/

// User element
type User struct {
	ID       string
	Username string
	Photo    string
}

// Conversation element, containing Users, ID, group boolean and a photo
type Conversation struct {
	ID             string
	IsGroup        bool
	Name           string
	Photo          string
	Members        []User
	Messages       []Message
	LastMessage    *Message
	LastActivityAt string
}

// Message represents a message stored in the database.
type Message struct {
	ID                     string
	ConversationID         string
	SenderID               string
	SenderUsername         string
	Type                   string
	Content                string
	ReplyToMessageID       string
	ForwardedFromMessageID string
	CreatedAt              string
	Deleted                bool
	DeliveredToAll         bool
	ReadByAll              bool
	Comments               []Comment
}

// Comment represents a reaction/comment attached to a message.
type Comment struct {
	ID             string
	MessageID      string
	AuthorID       string
	AuthorUsername string
	Content        string
	CreatedAt      string
}

/*
AppDatabase is the high level interface for the DB

It defines all operations the rest of the backend is allowed to perform on persistence.
*/
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
	GetConversationByIDForUser(conversationID string, userID string) (Conversation, error)
	AddUserToGroup(groupID string, requesterID string, userIDToAdd string) (Conversation, error)
	LeaveGroup(groupID string, userID string) error
	SetGroupName(groupID string, requesterID string, newName string) (Conversation, error)
	SetGroupPhoto(groupID string, requesterID string, newPhoto string) (Conversation, error)

	CreateMessage(conversationID string, senderID string, messageType string, content string, replyToMessageID string) (Message, error)
	ListMessagesByConversationForUser(conversationID string, userID string) ([]Message, error)
	MarkMessageAsRead(messageID string, userID string) error
	IsMessageReadByAll(messageID string) (bool, error)
	MarkConversationsAsDelivered(userID string) error
	IsMessageDeliveredToAll(messageID string) (bool, error)
	DeleteMessage(messageID string, userID string) error
	ForwardMessage(messageID string, senderID string, destinationConversationID string) (Message, error)

	CreateComment(messageID string, authorID string, content string) (Comment, error)
	ListCommentsByMessageID(messageID string) ([]Comment, error)
	DeleteComment(messageID string, commentID string, userID string) error
}

type appdbimpl struct {
	c *sql.DB
}

/*
New returns a new instance of AppDatabase based on the SQLite connection `db`.
`db` is required - an error will be returned if `db` is `nil`.

Schema initialization:
- for each table
- query sqlite_master
- if the table does not exist
- create it
*/
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

	var messagesTableName string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='messages';`).Scan(&messagesTableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `
		CREATE TABLE messages (
			id TEXT NOT NULL PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			reply_to_message_id TEXT NOT NULL DEFAULT '',
			forwarded_from_message_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			deleted INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
			FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating messages table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error checking messages table: %w", err)
	}

	var messageReadsTableName string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='message_reads';`).Scan(&messageReadsTableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `
		CREATE TABLE message_reads (
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			read_at TEXT NOT NULL,
			PRIMARY KEY (message_id, user_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating message_reads table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error checking message_reads table: %w", err)
	}

	var messageDeliveriesTableName string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='message_deliveries';`).Scan(&messageDeliveriesTableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `
		CREATE TABLE message_deliveries (
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			delivered_at TEXT NOT NULL,
			PRIMARY KEY (message_id, user_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating message_deliveries table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error checking message_deliveries table: %w", err)
	}

	var commentsTableName string
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='comments';`).Scan(&commentsTableName)
	if errors.Is(err, sql.ErrNoRows) {
		sqlStmt := `
		CREATE TABLE comments (
			id TEXT NOT NULL PRIMARY KEY,
			message_id TEXT NOT NULL,
			author_id TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE (message_id, author_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating comments table: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error checking comments table: %w", err)
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}
