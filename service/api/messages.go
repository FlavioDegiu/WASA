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

// Converts one database message into the API response object.
func messageToResponse(msg database.Message) map[string]interface{} {
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
			"deliveredToAll": false,
			"readByAll":      false,
		},
		"comments": []interface{}{},
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
