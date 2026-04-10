package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// Creates a new message inside a conversation if the sender belongs to it.
func (db *appdbimpl) CreateMessage(conversationID string, senderID string, messageType string, content string, replyToMessageID string) (Message, error) {
	// First, verify that the sender belongs to the conversation.
	belongs, err := db.userBelongsToConversation(conversationID, senderID)
	if err != nil {
		return Message{}, err
	}
	if !belongs {
		return Message{}, sql.ErrNoRows
	}

	messageID := "msg_" + uuid.Must(uuid.NewV4()).String()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err = db.c.Exec(
		`INSERT INTO messages (
			id, conversation_id, sender_id, type, content, reply_to_message_id, forwarded_from_message_id, created_at, deleted
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		messageID,
		conversationID,
		senderID,
		messageType,
		content,
		replyToMessageID,
		"",
		createdAt,
		0,
	)
	if err != nil {
		return Message{}, fmt.Errorf("error creating message: %w", err)
	}

	var msg Message
	err = db.c.QueryRow(
		`SELECT m.id, m.conversation_id, m.sender_id, u.username, m.type, m.content,
		        m.reply_to_message_id, m.forwarded_from_message_id, m.created_at, m.deleted
		 FROM messages m
		 INNER JOIN users u ON u.id = m.sender_id
		 WHERE m.id = ?`,
		messageID,
	).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.SenderID,
		&msg.SenderUsername,
		&msg.Type,
		&msg.Content,
		&msg.ReplyToMessageID,
		&msg.ForwardedFromMessageID,
		&msg.CreatedAt,
		&msg.Deleted,
	)
	if err != nil {
		return Message{}, fmt.Errorf("error loading created message: %w", err)
	}

	return msg, nil
}

// Returns all messages of a conversation if the given user belongs to it.
func (db *appdbimpl) ListMessagesByConversationForUser(conversationID string, userID string) ([]Message, error) {
	// Do not expose messages from conversations the user does not belong to.
	belongs, err := db.userBelongsToConversation(conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !belongs {
		return nil, sql.ErrNoRows
	}

	rows, err := db.c.Query(
		`SELECT m.id, m.conversation_id, m.sender_id, u.username, m.type, m.content,
		        m.reply_to_message_id, m.forwarded_from_message_id, m.created_at, m.deleted
		 FROM messages m
		 INNER JOIN users u ON u.id = m.sender_id
		 WHERE m.conversation_id = ?
		 ORDER BY m.created_at DESC`,
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("error listing messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.SenderID,
			&msg.SenderUsername,
			&msg.Type,
			&msg.Content,
			&msg.ReplyToMessageID,
			&msg.ForwardedFromMessageID,
			&msg.CreatedAt,
			&msg.Deleted,
		); err != nil {
			return nil, fmt.Errorf("error scanning message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}
