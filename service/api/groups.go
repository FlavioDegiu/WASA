package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
	"github.com/julienschmidt/httprouter"
)

type addToGroupRequest struct {
	UserID string `json:"userId"`
}

// Adds a user to a group conversation.
func (rt *_router) addToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// Authenticate the requester from the Bearer token.
	requesterID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Read the group ID from the URL path.
	groupID := ps.ByName("groupId")
	if groupID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "missing group identifier",
		})
		return
	}

	// Decode the request body.
	var req addToGroupRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode add-to-group request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "userId must be a non-empty string",
		})
		return
	}

	// Prevent the requester from trying to add themselves again.
	if req.UserID == requesterID {
		writeJSON(w, http.StatusConflict, errorResponse{
			Message: "user already belongs to the group",
		})
		return
	}

	// Try to add the new member to the group.
	conv, err := rt.db.AddUserToGroup(groupID, requesterID, req.UserID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrGroupNotFound):
			writeJSON(w, http.StatusForbidden, errorResponse{
				Message: "access denied",
			})
			return

		case errors.Is(err, database.ErrConversationNotGroup):
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message: "target conversation is not a group",
			})
			return

		case errors.Is(err, database.ErrUserNotFound):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "group or user not found",
			})
			return

		case errors.Is(err, database.ErrUserAlreadyInGroup):
			writeJSON(w, http.StatusConflict, errorResponse{
				Message: "user already belongs to the group",
			})
			return

		default:
			ctx.Logger.WithError(err).Error("cannot add user to group")
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Message: "internal server error",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, conversationToResponse(conv))
}
