package database

import (
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
