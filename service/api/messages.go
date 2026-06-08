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

/*
This file implements the message-related endpoints of the API.
It handles
	sending messages,
	marking them as read,
	soft-deleting them,
	adding and removing reactions,
	forwarding messages
*/

// JSON Request models
type sendMessageRequest struct {
	Type             string `json:"type"`
	Content          string `json:"content"`
	ReplyToMessageID string `json:"replyToMessageId"`
}

type commentMessageRequest struct {
	Content string `json:"content"`
}

type forwardMessageRequest struct {
	ConversationID string `json:"conversationId"`
}

/*
This handler implements

POST /conversations/:conversationId/messages

Main flow:
- authenticate the sender
- extract conversation ID
- decode request body
- validate message type/content
- create the message in DB
- return the created message
*/
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

	// Field normalization
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

	// Validate content not empty
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

/*
Mark messages as read for the authenticated user

# This handler implements

PUT /messages/:messageId/read]

Main flow:
- authenticate the user
- identify the message
- record a read receipt for that user
*/
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

	// Mark the message as read via the DB layer.
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

/*
This handler implements

DELETE /messages/:messageId

Its purpose is to soft-delete one of the authenticated user’s own sent messages.
*/
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

/*
This handler implements

POST /messages/:messageId/comments

Main flow:
- authenticate user
- identify target message
- decode comment body
- validate it as exactly one emoji
- create the reaction
- return the created comment
*/
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

	// Add comment in the DB layer
	comment, err := rt.db.CreateComment(messageID, userID, req.Content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "message not found",
			})
			return
		}

		//User can only react once to the comment
		if errors.Is(err, database.ErrUserAlreadyReacted) {
			writeJSON(w, http.StatusConflict, errorResponse{
				Message: "you already reacted to this message",
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

/*
This handler implements

DELETE /messages/:messageId/comments/:commentId

Its purpose is to remove one of the current user’s own reactions/comments.
*/
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

/*
This handler implements

POST /messages/:messageId/forward

Main flow:
- authenticate the user
- identify the source message
- read destination conversation from request body
- create a forwarded copy in the destination conversation
- return the new message
*/
func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// Authenticate the user from the Bearer token.
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Source
	// Read the source message ID from the URL path.
	messageID := ps.ByName("messageId")
	if messageID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "missing message identifier",
		})
		return
	}

	// Destination
	// Decode the destination from request body.
	var req forwardMessageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode forward-message request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	// Destination check
	req.ConversationID = strings.TrimSpace(req.ConversationID)
	if req.ConversationID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "conversationId must be a non-empty string",
		})
		return
	}

	// Ask the database layer to create the forwarded message.
	msg, err := rt.db.ForwardMessage(messageID, userID, req.ConversationID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrSourceMessageNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "message or destination conversation not found",
			})
			return

		case errors.Is(err, database.ErrDestinationConversationNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "message or destination conversation not found",
			})
			return

		default:
			ctx.Logger.WithError(err).Error("cannot forward message")
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Message: "internal server error",
			})
			return
		}
	}

	writeJSON(w, http.StatusCreated, messageToResponse(msg))
}
