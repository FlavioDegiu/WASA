package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
)

// boolToInt adapts Go booleans to the integer representation used in SQLite
func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (db *appdbimpl) CreateConversation(currentUserID string, memberIDs []string, isGroup bool, name string, photo string) (Conversation, error) {
	conversationID := "conv_" + uuid.Must(uuid.NewV4()).String()

	tx, err := db.c.Begin()
	if err != nil {
		return Conversation{}, fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.Exec(
		`INSERT INTO conversations (id, is_group, name, photo) VALUES (?, ?, ?, ?)`,
		conversationID,
		boolToInt(isGroup),
		name,
		photo,
	)
	if err != nil {
		return Conversation{}, fmt.Errorf("error creating conversation: %w", err)
	}

	// The creator is always part of the conversation
	allMemberIDs := []string{currentUserID}
	allMemberIDs = append(allMemberIDs, memberIDs...)

	for _, userID := range allMemberIDs {
		_, err = tx.Exec(
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`,
			conversationID,
			userID,
		)
		if err != nil {
			return Conversation{}, fmt.Errorf("error adding conversation member: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return Conversation{}, fmt.Errorf("error committing transaction: %w", err)
	}

	members := make([]User, 0, len(allMemberIDs))
	for _, userID := range allMemberIDs {
		user, err := db.GetUserByID(userID)
		if err != nil {
			return Conversation{}, fmt.Errorf("error loading conversation member: %w", err)
		}
		members = append(members, user)
	}

	return Conversation{
		ID:      conversationID,
		IsGroup: isGroup,
		Name:    name,
		Photo:   photo,
		Members: members,
	}, nil
}

// Returns all users that belong to a conversation.
func (db *appdbimpl) getConversationMembers(conversationID string) ([]User, error) {
	rows, err := db.c.Query(
		`SELECT u.id, u.username, u.photo
		 FROM users u
		 INNER JOIN conversation_members cm ON cm.user_id = u.id
		 WHERE cm.conversation_id = ?
		 ORDER BY u.username ASC`,
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("error querying conversation members: %w", err)
	}
	defer rows.Close()

	var members []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Photo); err != nil {
			return nil, fmt.Errorf("error scanning conversation member: %w", err)
		}
		members = append(members, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversation members: %w", err)
	}

	return members, nil
}

// Returns all conversations where the given user is a member.
func (db *appdbimpl) ListConversationsByUser(userID string) ([]Conversation, error) {
	rows, err := db.c.Query(
		`SELECT c.id, c.is_group, c.name, c.photo
		 FROM conversations c
		 INNER JOIN conversation_members cm ON cm.conversation_id = c.id
		 WHERE cm.user_id = ?
		 ORDER BY c.id ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("error listing conversations: %w", err)
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		var isGroupInt int

		// Read the basic conversation fields from the database row.
		if err := rows.Scan(&conv.ID, &isGroupInt, &conv.Name, &conv.Photo); err != nil {
			return nil, fmt.Errorf("error scanning conversation: %w", err)
		}

		// Convert SQLite integer flag into a Go boolean.
		conv.IsGroup = isGroupInt == 1

		// Load all members for this conversation.
		members, err := db.getConversationMembers(conv.ID)
		if err != nil {
			return nil, fmt.Errorf("error loading conversation members: %w", err)
		}
		conv.Members = members

		conversations = append(conversations, conv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversations: %w", err)
	}

	return conversations, nil
}

// Returns one conversation only if the given user is a member of it.
func (db *appdbimpl) GetConversationByIDForUser(conversationID string, userID string) (Conversation, error) {
	var conv Conversation
	var isGroupInt int

	// We join with conversation_members so that a user can only read
	// conversations they actually belong to.
	err := db.c.QueryRow(
		`SELECT c.id, c.is_group, c.name, c.photo
		 FROM conversations c
		 INNER JOIN conversation_members cm ON cm.conversation_id = c.id
		 WHERE c.id = ? AND cm.user_id = ?`,
		conversationID,
		userID,
	).Scan(&conv.ID, &isGroupInt, &conv.Name, &conv.Photo)
	if err != nil {
		return Conversation{}, err
	}

	// Convert the SQLite integer flag into a Go boolean.
	conv.IsGroup = isGroupInt == 1

	// Load the full list of members for this conversation.
	members, err := db.getConversationMembers(conv.ID)
	if err != nil {
		return Conversation{}, fmt.Errorf("error loading conversation members: %w", err)
	}
	conv.Members = members

	// Load all messages visible to this user in this conversation.
	messages, err := db.ListMessagesByConversationForUser(conv.ID, userID)
	if err != nil {
		return Conversation{}, fmt.Errorf("error loading conversation messages: %w", err)
	}
	conv.Messages = messages

	return conv, nil
}

// Adds a user to a group conversation if the requester already belongs to it.
func (db *appdbimpl) AddUserToGroup(groupID string, requesterID string, userIDToAdd string) (Conversation, error) {
	// Load the conversation only if the requester already belongs to it.
	conv, err := db.GetConversationByIDForUser(groupID, requesterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Conversation{}, ErrGroupNotFound
		}
		return Conversation{}, err
	}

	// The target conversation must be a group.
	if !conv.IsGroup {
		return Conversation{}, ErrConversationNotGroup
	}

	// The user to add must exist.
	_, err = db.GetUserByID(userIDToAdd)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Conversation{}, ErrUserNotFound
		}
		return Conversation{}, err
	}

	// The user must not already be in the group.
	userAlreadyInside, err := db.userBelongsToConversation(groupID, userIDToAdd)
	if err != nil {
		return Conversation{}, err
	}
	if userAlreadyInside {
		return Conversation{}, ErrUserAlreadyInGroup
	}

	// Add the new member to the group.
	_, err = db.c.Exec(
		`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`,
		groupID,
		userIDToAdd,
	)
	if err != nil {
		return Conversation{}, fmt.Errorf("error adding user to group: %w", err)
	}

	// Return the updated conversation as visible to the requester.
	updatedConv, err := db.GetConversationByIDForUser(groupID, requesterID)
	if err != nil {
		return Conversation{}, fmt.Errorf("error loading updated group: %w", err)
	}

	return updatedConv, nil
}
