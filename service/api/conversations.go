package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

/*
This file implements the conversation-related endpoints of the API.
It handles conversation creation, conversation list retrieval, and loading a full conversation.
*/

// JSON model for POST /conversations
type createConversationRequest struct {
	IsGroup bool     `json:"isGroup"`
	Name    string   `json:"name"`
	Photo   string   `json:"photo"`
	Members []string `json:"members"`
}

/*
This handler implements

POST /conversations

Main flow:
- authenticate the user
- decode request body
- validate conversation rules
- validate requested members
- create the conversation in the database
- return the created conversation
*/
func (rt *_router) createConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// Bearer auth
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Decode request body
	var req createConversationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode create conversation request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	// Trim space
	req.Name = strings.TrimSpace(req.Name)
	req.Photo = strings.TrimSpace(req.Photo)

	// Validate that members are present
	if len(req.Members) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "members must contain at least one user",
		})
		return
	}

	// Groups require a visible name
	// Direct conversation must have only one user
	if req.IsGroup {
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message: "group name is required",
			})
			return
		}
	} else {
		if len(req.Members) != 1 {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message: "direct conversations must contain exactly one other user",
			})
			return
		}
	}

	// Validate each conversation member
	for _, memberID := range req.Members {

		// The creator is automatically added later, so it must not appear in the payload.
		if memberID == userID {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message: "creator must not be included in members list",
			})
			return
		}

		// Validate that every requested member already exists.
		_, err := rt.db.GetUserByID(memberID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, errorResponse{
					Message: "one or more users not found",
				})
				return
			}

			ctx.Logger.WithError(err).Error("cannot validate conversation members")
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Message: "internal server error",
			})
			return
		}
	}

	// Create conversation in db layer
	conv, err := rt.db.CreateConversation(userID, req.Members, req.IsGroup, req.Name, req.Photo)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot create conversation")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusCreated, conversationToResponse(conv))
}

/*
This handler implements

GET /conversations

Main flow:
- authenticate the user
- mark visible incoming messages as delivered
- load all conversations for the user
- convert them into summary objects
- return them as JSON
*/
func (rt *_router) getMyConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// Extract the authenticated user ID from the Bearer token.
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Loading the conversation list means the user has received visible incoming messages.
	// Mark them as delivered
	err := rt.db.MarkConversationsAsDelivered(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot mark conversations as delivered")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	// Load all conversations for this user.
	conversations, err := rt.db.ListConversationsByUser(userID)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list user conversations")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	// Convert database conversations into API response objects.
	items := make([]map[string]interface{}, 0, len(conversations))
	for _, conv := range conversations {
		items = append(items, conversationSummaryToResponse(conv, userID))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

/*
This handler implements

GET /conversations/:conversationId

Main flow:
- authenticate the user
- read the conversation ID from the route
- ensure the user belongs to the conversation
- return the full conversation
*/
func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// Extract the authenticated user from the Bearer token.
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Read the conversation ID from the URL path.
	conversationID := ps.ByName("conversationId")
	if conversationID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "missing conversation identifier",
		})
		return
	}

	// Load the conversation, but only if this user belongs to it.
	conv, err := rt.db.GetConversationByIDForUser(conversationID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "conversation not found",
			})
			return
		}

		ctx.Logger.WithError(err).Error("cannot load conversation")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, conversationToResponse(conv))
}
