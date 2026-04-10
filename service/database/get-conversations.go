package database

import "fmt"

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
