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

type createConversationRequest struct {
	IsGroup bool     `json:"isGroup"`
	Name    string   `json:"name"`
	Photo   string   `json:"photo"`
	Members []string `json:"members"`
}

// conversationToResponse adapts the database model to the JSON returned by the API.
func conversationToResponse(conv database.Conversation) map[string]interface{} {
	resp := map[string]interface{}{
		"id":       conv.ID,
		"isGroup":  conv.IsGroup,
		"members":  usersToResponse(conv.Members),
		"messages": []interface{}{},
	}

	if conv.Name != "" {
		resp["name"] = conv.Name
	}
	if conv.Photo != "" {
		resp["photo"] = conv.Photo
	}

	return resp
}

// usersToResponse applies the same conversion to every conversation member
func usersToResponse(users []database.User) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, userToResponse(user))
	}
	return items
}

func (rt *_router) createConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	var req createConversationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode create conversation request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Photo = strings.TrimSpace(req.Photo)

	if len(req.Members) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "members must contain at least one user",
		})
		return
	}

	// Groups require a visible name
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

// Converts a database conversation into the smaller summary object used by GET /conversations.
func conversationSummaryToResponse(conv database.Conversation, currentUserID string) map[string]interface{} {
	title := conv.Name

	// For direct conversations, the title should be the other user's username.
	if !conv.IsGroup {
		for _, member := range conv.Members {
			if member.ID != currentUserID {
				title = member.Username
				break
			}
		}
	}

	resp := map[string]interface{}{
		"id":        conv.ID,
		"isGroup":   conv.IsGroup,
		"title":     title,
		"updatedAt": "",
	}

	// Add photo only if present.
	if conv.Photo != "" {
		resp["photo"] = conv.Photo
	}

	// For now we return a minimal preview placeholder.
	// Later, when messages exist, we will replace this with the latest message preview.
	resp["lastMessage"] = map[string]interface{}{
		"type":      "text",
		"snippet":   "",
		"senderId":  "",
		"createdAt": "",
	}

	return resp
}

// Returns all conversations of the authenticated user.
func (rt *_router) getMyConversations(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// Extract the authenticated user ID from the Bearer token.
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
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
