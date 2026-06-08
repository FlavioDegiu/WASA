package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

/*
This file contains utility functions shared across API handlers.
It is useful to centralize repeated logic.
*/

// WriteJSON standardizes JSON responses for all handlers
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {

	// Set content time - the response is JSON
	w.Header().Set("Content-Type", "application/json")

	// Write HTTP status code
	w.WriteHeader(statusCode)

	// Check for body
	if data == nil {
		return
	}

	// Encode as JSON
	_ = json.NewEncoder(w).Encode(data)
}

// getBearerToken extracts the user identifier stored in the Authorization header
func getBearerToken(r *http.Request) (string, bool) {
	// Read header and trim spaces
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))

	// Reject if empty
	if authHeader == "" {
		return "", false
	}

	// Require "Bearer " prefix
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}

	// Remove prefix
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if token == "" {
		return "", false
	}

	// Return string, bool as it does not give any further information about the eventual error
	return token, true
}

// Converts a database user into the full API response object.
func usersToResponse(users []database.User) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, userToResponse(user))
	}
	return items
}

// userToResponse converts the internal database model into the public API shape
func userToResponse(user database.User) map[string]interface{} {
	resp := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	}

	// Only include optional fields when they actually contain data
	if user.Photo != "" {
		resp["photo"] = user.Photo
	}

	return resp
}

// Converts a database conversation into the full API response object.
func conversationToResponse(conv database.Conversation) map[string]interface{} {

	// Converting messages
	messageItems := make([]interface{}, 0, len(conv.Messages))
	for _, msg := range conv.Messages {
		messageItems = append(messageItems, messageToResponse(msg))
	}

	// Also converting Members
	resp := map[string]interface{}{
		"id":       conv.ID,
		"isGroup":  conv.IsGroup,
		"members":  usersToResponse(conv.Members),
		"messages": messageItems,
	}

	// Add optional fields only when present.
	if conv.Name != "" {
		resp["name"] = conv.Name
	}
	if conv.Photo != "" {
		resp["photo"] = conv.Photo
	}

	return resp
}

// Converts a database conversation into the smaller summary object used by GET /conversations.
func conversationSummaryToResponse(conv database.Conversation, currentUserID string) map[string]interface{} {
	title := conv.Name
	photo := conv.Photo

	// For direct conversations, show the other user's username and photo.
	if !conv.IsGroup {
		for _, member := range conv.Members {
			if member.ID != currentUserID {
				title = member.Username
				photo = member.Photo
				break
			}
		}
	}

	// Build a default empty preview for conversations with no messages yet.
	lastMessage := map[string]interface{}{
		"type":      "text",
		"content":   "",
		"senderId":  "",
		"createdAt": "",
	}

	// If there is a latest message, build a real preview from it.
	if conv.LastMessage != nil {
		content := conv.LastMessage.Content

		// Deleted messages get a placeholder preview.
		if conv.LastMessage.Deleted {
			content = "[deleted message]"
		}

		// For image and gif messages, keep the preview simple.
		if conv.LastMessage.Type == "image" || conv.LastMessage.Type == "gif" {
			content = ""
		}

		lastMessage = map[string]interface{}{
			"type":      conv.LastMessage.Type,
			"content":   content,
			"senderId":  conv.LastMessage.SenderID,
			"createdAt": conv.LastMessage.CreatedAt,
		}
	}

	resp := map[string]interface{}{
		"id":          conv.ID,
		"isGroup":     conv.IsGroup,
		"title":       title,
		"updatedAt":   conv.LastActivityAt,
		"lastMessage": lastMessage,
	}

	// Add photo only if present.
	if photo != "" {
		resp["photo"] = photo
	}

	return resp
}

// Converts one database message into the API response object.
func messageToResponse(msg database.Message) map[string]interface{} {

	commentItems := make([]interface{}, 0, len(msg.Comments))
	for _, comment := range msg.Comments {
		commentItems = append(commentItems, commentToResponse(comment))
	}

	resp := map[string]interface{}{
		"id":             msg.ID,
		"conversationId": msg.ConversationID,
		"senderId":       msg.SenderID,
		"senderUsername": msg.SenderUsername,
		"type":           msg.Type,
		"content":        msg.Content,
		"createdAt":      msg.CreatedAt,
		"deleted":        msg.Deleted,
		"status": map[string]interface{}{
			"deliveredToAll": msg.DeliveredToAll,
			"readByAll":      msg.ReadByAll,
		},
		"comments": commentItems,
	}

	// Add optional fields only when present.
	if msg.ReplyToMessageID != "" {
		resp["replyToMessageId"] = msg.ReplyToMessageID
	}
	if msg.ForwardedFromMessageID != "" {
		resp["forwardedFromMessageId"] = msg.ForwardedFromMessageID
	}

	return resp
}

// Converts one database comment into the API response object.
func commentToResponse(comment database.Comment) map[string]interface{} {
	return map[string]interface{}{
		"id":             comment.ID,
		"messageId":      comment.MessageID,
		"authorId":       comment.AuthorID,
		"authorUsername": comment.AuthorUsername,
		"content":        comment.Content,
		"createdAt":      comment.CreatedAt,
	}
}

// This regex is a practical approximation for emoji validation, it accepts most common emoji and emoji presentation characters
var emojiRegex = regexp.MustCompile(`^(?:[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{2199}\x{21A9}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{27BF}\x{2934}-\x{2935}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F000}-\x{1FAFF}](?:\x{FE0F}|\x{200D}[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{2199}\x{21A9}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{27BF}\x{2934}-\x{2935}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F000}-\x{1FAFF}])*)$`)

func isSingleEmoji(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	// Reject obviously long text.
	if utf8.RuneCountInString(s) > 8 {
		return false
	}

	return emojiRegex.MatchString(s)
}
