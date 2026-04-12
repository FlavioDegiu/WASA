package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
	"github.com/julienschmidt/httprouter"
)

type sendMessageRequest struct {
	Type             string `json:"type"`
	Content          string `json:"content"`
	ReplyToMessageID string `json:"replyToMessageId"`
}

type commentMessageRequest struct {
	Content string `json:"content"`
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

// Sends a new message inside a conversation.
func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// Authenticate the user from the Bearer token.
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Read the conversation ID from the URL.
	conversationID := ps.ByName("conversationId")
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "missing conversation identifier",
		})
		return
	}

	// Decode the JSON request body.
	var req sendMessageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode send message request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	req.Type = strings.TrimSpace(req.Type)
	req.Content = strings.TrimSpace(req.Content)
	req.ReplyToMessageID = strings.TrimSpace(req.ReplyToMessageID)

	// Validate the basic fields according to the API contract.
	if req.Type != "text" && req.Type != "gif" && req.Type != "image" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid message type",
		})
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "message content must be a non-empty string",
		})
		return
	}

	// Create the message in the database.
	msg, err := rt.db.CreateMessage(conversationID, userID, req.Type, req.Content, req.ReplyToMessageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "conversation not found",
			})
			return
		}

		ctx.Logger.WithError(err).Error("cannot create message")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusCreated, messageToResponse(msg))
}

// Marks a message as read for the authenticated user.
func (rt *_router) markMessageAsRead(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// Authenticate the user from the Bearer token.
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Read the message ID from the URL path.
	messageID := ps.ByName("messageId")
	if messageID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "missing message identifier",
		})
		return
	}

	// Mark the message as read.
	err := rt.db.MarkMessageAsRead(messageID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "message not found",
			})
			return
		}

		ctx.Logger.WithError(err).Error("cannot mark message as read")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	// This endpoint returns 204 No Content on success.
	w.WriteHeader(http.StatusNoContent)
}

// Deletes one of the authenticated user's sent messages.
// The message is not physically removed from the database; it is only marked as deleted.
func (rt *_router) deleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// Authenticate the user from the Bearer token.
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Read the message ID from the URL path.
	messageID := ps.ByName("messageId")
	if messageID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "missing message identifier",
		})
		return
	}

	// Try to delete the message.
	err := rt.db.DeleteMessage(messageID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "message not found",
			})
			return
		}

		// If the message exists but belongs to another sender,
		// return 403 because only the original sender can delete it.
		if errors.Is(err, database.ErrMessageNotOwned) {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Message: "only the sender can delete the message",
			})
			return
		}

		ctx.Logger.WithError(err).Error("cannot delete message")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	// On success, this endpoint returns 204 No Content.
	w.WriteHeader(http.StatusNoContent)
}

// Adds a reaction/comment to a message.
func (rt *_router) commentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// Authenticate the user from the Bearer token.
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Read the message ID from the URL path.
	messageID := ps.ByName("messageId")
	if messageID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "missing message identifier",
		})
		return
	}

	// Decode the JSON request body.
	var req commentMessageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode comment request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	req.Content = strings.TrimSpace(req.Content)

	// Comment is exactly one emoji
	if !isSingleEmoji(req.Content) {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "comment must contain exactly one emoji",
		})
		return
	}

	comment, err := rt.db.CreateComment(messageID, userID, req.Content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "message not found",
			})
			return
		}

		ctx.Logger.WithError(err).Error("cannot create comment")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusCreated, commentToResponse(comment))
}

// Removes one of the authenticated user's comments from a message.
func (rt *_router) uncommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// Authenticate the user from the Bearer token.
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Read both identifiers from the URL path.
	messageID := ps.ByName("messageId")
	if messageID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "missing message identifier",
		})
		return
	}

	commentID := ps.ByName("commentId")
	if commentID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "missing comment identifier",
		})
		return
	}

	// Try to delete the comment.
	err := rt.db.DeleteComment(messageID, commentID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "comment not found",
			})
			return
		}

		// Only the original author can remove their comment.
		if errors.Is(err, database.ErrCommentNotOwned) {
			writeJSON(w, http.StatusForbidden, errorResponse{
				Message: "only the comment author can remove it",
			})
			return
		}

		ctx.Logger.WithError(err).Error("cannot delete comment")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	// On success, this endpoint returns 204 No Content.
	w.WriteHeader(http.StatusNoContent)
}
