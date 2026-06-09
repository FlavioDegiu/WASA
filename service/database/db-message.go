package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

/*
Creates a new message inside a conversation if the sender belongs to it
Delegates the logic to createMessageWithForward with the forward field empty
*/
func (db *appdbimpl) CreateMessage(conversationID string, senderID string, messageType string, content string, replyToMessageID string) (Message, error) {
	return db.createMessageWithForward(conversationID, senderID, messageType, content, replyToMessageID, "")
}

// Returns one message by ID with sender username included.
func (db *appdbimpl) getMessageByID(messageID string) (Message, error) {

	var msg Message

	// Base query. Join with 'users' because senderUsername is needed
	err := db.c.QueryRow(
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
		return Message{}, err
	}

	// Enrich the message data with data from the database using helper functions
	msg.DeliveredToAll, err = db.IsMessageDeliveredToAll(msg.ID)
	if err != nil {
		return Message{}, fmt.Errorf("error computing message delivery status: %w", err)
	}

	msg.ReadByAll, err = db.IsMessageReadByAll(msg.ID)
	if err != nil {
		return Message{}, fmt.Errorf("error computing message read status: %w", err)
	}

	msg.Comments, err = db.ListCommentsByMessageID(msg.ID)
	if err != nil {
		return Message{}, fmt.Errorf("error loading message comments: %w", err)
	}

	return msg, nil
}

/*
Creates a new message row, optionally linking it to a forwarded source message.
*/
func (db *appdbimpl) createMessageWithForward(
	conversationID string,
	senderID string,
	messageType string,
	content string,
	replyToMessageID string,
	forwardedFromMessageID string,
) (Message, error) {

	// Membership check
	belongs, err := db.userBelongsToConversation(conversationID, senderID)
	if err != nil {
		return Message{}, err
	}
	if !belongs {
		return Message{}, sql.ErrNoRows
	}

	// Generate ID and timestamp
	messageID := "msg_" + uuid.Must(uuid.NewV4()).String()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Message insertion
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
		forwardedFromMessageID,
		createdAt,
		0,
	)
	if err != nil {
		return Message{}, fmt.Errorf("error creating message: %w", err)
	}

	msg, err := db.getMessageByID(messageID)
	if err != nil {
		return Message{}, fmt.Errorf("error loading created message: %w", err)
	}

	return msg, nil
}

/*
Returns all messages of a conversation if the given user belongs to it
*/
func (db *appdbimpl) ListMessagesByConversationForUser(conversationID string, userID string) ([]Message, error) {

	// Membership check
	belongs, err := db.userBelongsToConversation(conversationID, userID)
	if err != nil {
		return nil, err
	}
	if !belongs {
		return nil, sql.ErrNoRows
	}

	// SQL query - newest first
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

		// Compute whether all recipients have received this message in their conversation list.
		msg.DeliveredToAll, err = db.IsMessageDeliveredToAll(msg.ID)
		if err != nil {
			return nil, fmt.Errorf("error computing message delivery status: %w", err)
		}

		// Compute whether all recipients have read this message.
		msg.ReadByAll, err = db.IsMessageReadByAll(msg.ID)
		if err != nil {
			return nil, fmt.Errorf("error computing message read status: %w", err)
		}

		// Load all comments attached to this message.
		msg.Comments, err = db.ListCommentsByMessageID(msg.ID)
		if err != nil {
			return nil, fmt.Errorf("error loading message comments: %w", err)
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// Returns the conversation ID that owns the given message.
func (db *appdbimpl) getConversationIDByMessageID(messageID string) (string, error) {

	var conversationID string

	// Little SQL query to find the conversation that owns a message
	err := db.c.QueryRow(
		`SELECT conversation_id FROM messages WHERE id = ?`,
		messageID,
	).Scan(&conversationID)
	if err != nil {
		return "", err
	}

	return conversationID, nil
}

// Marks a message as read by the given user
func (db *appdbimpl) MarkMessageAsRead(messageID string, userID string) error {

	// Find which conversation this message belongs to.
	conversationID, err := db.getConversationIDByMessageID(messageID)
	if err != nil {
		return err
	}

	// Only members of the conversation can mark the message as read.
	belongs, err := db.userBelongsToConversation(conversationID, userID)
	if err != nil {
		return err
	}
	if !belongs {
		return sql.ErrNoRows
	}

	// Read timestamp
	readAt := time.Now().UTC().Format(time.RFC3339)

	// INSERT OR REPLACE keeps the operation idempotent:
	// if the user marks it as read twice, we still end in a valid state
	_, err = db.c.Exec(
		`INSERT OR REPLACE INTO message_reads (message_id, user_id, read_at) VALUES (?, ?, ?)`,
		messageID,
		userID,
		readAt,
	)
	if err != nil {
		return fmt.Errorf("error marking message as read: %w", err)
	}

	return nil
}

// Returns true if every recipient of a message has read it.
func (db *appdbimpl) IsMessageReadByAll(messageID string) (bool, error) {

	var conversationID string
	var senderID string

	// Load the conversation and sender for this message.
	err := db.c.QueryRow(
		`SELECT conversation_id, sender_id FROM messages WHERE id = ?`,
		messageID,
	).Scan(&conversationID, &senderID)
	if err != nil {
		return false, err
	}

	// Conta i riceventi
	var recipientCount int
	err = db.c.QueryRow(
		`SELECT COUNT(*) FROM conversation_members
		 WHERE conversation_id = ? AND user_id <> ?`,
		conversationID,
		senderID,
	).Scan(&recipientCount)
	if err != nil {
		return false, fmt.Errorf("error counting recipients: %w", err)
	}

	// Conta quanti lo hanno letto
	var readCount int
	err = db.c.QueryRow(
		`SELECT COUNT(*) FROM message_reads
		 WHERE message_id = ? AND user_id <> ?`,
		messageID,
		senderID,
	).Scan(&readCount)
	if err != nil {
		return false, fmt.Errorf("error counting message reads: %w", err)
	}

	return recipientCount > 0 && readCount == recipientCount, nil
}

// Marks as delivered all messages that belong to conversations visible to the given user.
func (db *appdbimpl) MarkConversationsAsDelivered(userID string) error {

	// Deliver timestamp
	deliveredAt := time.Now().UTC().Format(time.RFC3339)

	// SQL insertion
	_, err := db.c.Exec(
		`INSERT OR IGNORE INTO message_deliveries (message_id, user_id, delivered_at)
		 SELECT m.id, ?, ?
		 FROM messages m
		 INNER JOIN conversation_members cm ON cm.conversation_id = m.conversation_id
		 WHERE cm.user_id = ?
		   AND m.sender_id <> ?`,
		userID,
		deliveredAt,
		userID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("error marking conversations as delivered: %w", err)
	}

	return nil
}

/*
Returns true if every recipient of a message has received it in their conversation list.

- load conversation and sender
- count recipients excluding sender
- count deliveries excluding sender
- compare counts
*/
func (db *appdbimpl) IsMessageDeliveredToAll(messageID string) (bool, error) {

	var conversationID string
	var senderID string

	// Load conversation and sender
	err := db.c.QueryRow(
		`SELECT conversation_id, sender_id FROM messages WHERE id = ?`,
		messageID,
	).Scan(&conversationID, &senderID)
	if err != nil {
		return false, err
	}

	// Cound recipients
	var recipientCount int
	err = db.c.QueryRow(
		`SELECT COUNT(*) FROM conversation_members
		 WHERE conversation_id = ? AND user_id <> ?`,
		conversationID,
		senderID,
	).Scan(&recipientCount)
	if err != nil {
		return false, fmt.Errorf("error counting message recipients: %w", err)
	}

	// Count delivered
	var deliveredCount int
	err = db.c.QueryRow(
		`SELECT COUNT(*) FROM message_deliveries
		 WHERE message_id = ? AND user_id <> ?`,
		messageID,
		senderID,
	).Scan(&deliveredCount)
	if err != nil {
		return false, fmt.Errorf("error counting delivered recipients: %w", err)
	}

	return recipientCount > 0 && deliveredCount == recipientCount, nil
}

/*
Marks a message as deleted if it belongs to the given sender.
*/
func (db *appdbimpl) DeleteMessage(messageID string, userID string) error {
	var senderID string

	// Load the sender of the message so we can verify ownership.
	err := db.c.QueryRow(
		`SELECT sender_id FROM messages WHERE id = ?`,
		messageID,
	).Scan(&senderID)
	if err != nil {
		return err
	}

	// Only the original sender can delete the message.
	if senderID != userID {
		return ErrMessageNotOwned
	}

	// Soft delete: keep the message row, but mark it as deleted.
	result, err := db.c.Exec(
		`UPDATE messages SET deleted = 1 WHERE id = ?`,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("error deleting message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking deleted rows: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Creates a comment on a message if the author belongs to the conversation.
func (db *appdbimpl) CreateComment(messageID string, authorID string, content string) (Comment, error) {

	// Only users in the conversation can comment the message.
	canAccess, err := db.userCanAccessMessage(messageID, authorID)
	if err != nil {
		return Comment{}, err
	}
	if !canAccess {
		return Comment{}, sql.ErrNoRows
	}

	// A user can add at most one reaction/comment to the same message.
	var existingCount int
	err = db.c.QueryRow(
		`SELECT COUNT(*) FROM comments WHERE message_id = ? AND author_id = ?`,
		messageID,
		authorID,
	).Scan(&existingCount)
	if err != nil {
		return Comment{}, fmt.Errorf("error checking existing comment: %w", err)
	}
	if existingCount > 0 {
		return Comment{}, ErrUserAlreadyReacted
	}

	// ID and timestamp
	commentID := "comm_" + uuid.Must(uuid.NewV4()).String()
	createdAt := time.Now().UTC().Format(time.RFC3339)

	// SQL insertion of the comment
	_, err = db.c.Exec(
		`INSERT INTO comments (id, message_id, author_id, content, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		commentID,
		messageID,
		authorID,
		content,
		createdAt,
	)
	if err != nil {
		return Comment{}, fmt.Errorf("error creating comment: %w", err)
	}

	// Return the comment by querying from the DB
	var comment Comment
	err = db.c.QueryRow(
		`SELECT c.id, c.message_id, c.author_id, u.username, c.content, c.created_at
		 FROM comments c
		 INNER JOIN users u ON u.id = c.author_id
		 WHERE c.id = ?`,
		commentID,
	).Scan(
		&comment.ID,
		&comment.MessageID,
		&comment.AuthorID,
		&comment.AuthorUsername,
		&comment.Content,
		&comment.CreatedAt,
	)
	if err != nil {
		return Comment{}, fmt.Errorf("error loading created comment: %w", err)
	}

	return comment, nil
}

// Returns all comments attached to a message.
func (db *appdbimpl) ListCommentsByMessageID(messageID string) ([]Comment, error) {

	// SQL query
	rows, err := db.c.Query(
		`SELECT c.id, c.message_id, c.author_id, u.username, c.content, c.created_at
		 FROM comments c
		 INNER JOIN users u ON u.id = c.author_id
		 WHERE c.message_id = ?
		 ORDER BY c.created_at ASC`,
		messageID,
	)
	if err != nil {
		return nil, fmt.Errorf("error listing comments: %w", err)
	}
	defer rows.Close()

	// rows decoding
	var comments []Comment
	for rows.Next() {
		var comment Comment
		if err := rows.Scan(
			&comment.ID,
			&comment.MessageID,
			&comment.AuthorID,
			&comment.AuthorUsername,
			&comment.Content,
			&comment.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning comment: %w", err)
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating comments: %w", err)
	}

	return comments, nil
}

/*
Deletes a comment only if it belongs to the given author
and is attached to the given message.
*/
func (db *appdbimpl) DeleteComment(messageID string, commentID string, userID string) error {
	var authorID string

	// Load the author of the comment and verify that the comment
	// belongs to the given message.
	err := db.c.QueryRow(
		`SELECT author_id
		 FROM comments
		 WHERE id = ? AND message_id = ?`,
		commentID,
		messageID,
	).Scan(&authorID)
	if err != nil {
		return err
	}

	// Only the original author can delete the comment.
	if authorID != userID {
		return ErrCommentNotOwned
	}

	// Remove the comment row from the database.
	result, err := db.c.Exec(
		`DELETE FROM comments WHERE id = ? AND message_id = ?`,
		commentID,
		messageID,
	)
	if err != nil {
		return fmt.Errorf("error deleting comment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking deleted comment rows: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

/*
Forwards an existing message into another conversation if the user
can access the source message and belongs to the destination conversation.
*/
func (db *appdbimpl) ForwardMessage(messageID string, senderID string, destinationConversationID string) (Message, error) {

	// The user must be allowed to see the source message.
	canAccessSource, err := db.userCanAccessMessage(messageID, senderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Message{}, ErrSourceMessageNotFound
		}
		return Message{}, err
	}
	if !canAccessSource {
		return Message{}, ErrSourceMessageNotFound
	}

	// The user must belong to the destination conversation too.
	belongsToDestination, err := db.userBelongsToConversation(destinationConversationID, senderID)
	if err != nil {
		return Message{}, err
	}
	if !belongsToDestination {
		return Message{}, ErrDestinationConversationNotFound
	}

	// Load the source message so we can copy its visible payload.
	sourceMsg, err := db.getMessageByID(messageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Message{}, ErrSourceMessageNotFound
		}
		return Message{}, err
	}

	// Create a new message in the destination conversation, linked to the source.
	forwardedMsg, err := db.createMessageWithForward(
		destinationConversationID,
		senderID,
		sourceMsg.Type,
		sourceMsg.Content,
		"",
		sourceMsg.ID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Message{}, ErrDestinationConversationNotFound
		}
		return Message{}, err
	}

	return forwardedMsg, nil
}

// Returns the latest message of a conversation, or nil if the conversation has no messages yet.
func (db *appdbimpl) getLatestMessageByConversationID(conversationID string) (*Message, error) {

	var msg Message

	// SQL query
	err := db.c.QueryRow(
		`SELECT m.id, m.conversation_id, m.sender_id, u.username, m.type, m.content,
		        m.reply_to_message_id, m.forwarded_from_message_id, m.created_at, m.deleted
		 FROM messages m
		 INNER JOIN users u ON u.id = m.sender_id
		 WHERE m.conversation_id = ?
		 ORDER BY m.created_at DESC
		 LIMIT 1`,
		conversationID,
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error loading latest message: %w", err)
	}

	// Fill derived fields so the message is consistent with the rest of the API.
	msg.DeliveredToAll = false

	msg.ReadByAll, err = db.IsMessageReadByAll(msg.ID)
	if err != nil {
		return nil, fmt.Errorf("error computing latest message read status: %w", err)
	}

	msg.Comments, err = db.ListCommentsByMessageID(msg.ID)
	if err != nil {
		return nil, fmt.Errorf("error loading latest message comments: %w", err)
	}

	return &msg, nil
}
