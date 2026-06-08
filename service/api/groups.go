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
This file implements the group-management endpoints of the API.

It handles
	adding members to a group,
	leaving a group,
	and updating group metadata
*/

// JSON Request models
type addToGroupRequest struct {
	UserID string `json:"userId"`
}

type setGroupNameRequest struct {
	Name string `json:"name"`
}

type setGroupPhotoRequest struct {
	Photo string `json:"photo"`
}

/*
This handler implements

POST /groups/:groupId/members

Main flow:
- authenticate the requester
- identify the target group
- decode the user to add
- validate the request
- delegate the real membership update to the DB layer
- return the updated conversation
*/
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

	// Validate user
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

	// Try to add the new member to the group via the DB layer.
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

/*
This handler implements

DELETE /groups/:groupId/members/me

Its job is to remove the authenticated user from a group.
*/
func (rt *_router) leaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// Authenticate the user from the Bearer token.
	userID, ok := getBearerToken(r)
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

	// Ask the database layer to remove the authenticated user from the group.
	err := rt.db.LeaveGroup(groupID, userID)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNotGroupMember):
			writeJSON(w, http.StatusForbidden, errorResponse{
				Message: "access denied",
			})
			return

		case errors.Is(err, database.ErrConversationNotGroup):
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message: "target conversation is not a group",
			})
			return

		default:
			ctx.Logger.WithError(err).Error("cannot leave group")
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Message: "internal server error",
			})
			return
		}
	}

	// On success, this endpoint returns 204 No Content.
	w.WriteHeader(http.StatusNoContent)
}

/*
This handler implements

PUT /groups/:groupId/name

Its purpose is to change the visible name of a group.
*/
func (rt *_router) setGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

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
	var req setGroupNameRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode set-group-name request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	// Validate group name
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "group name must be a non-empty string",
		})
		return
	}

	// Ask the database layer to update the group name.
	conv, err := rt.db.SetGroupName(groupID, requesterID, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNotGroupMember):
			writeJSON(w, http.StatusForbidden, errorResponse{
				Message: "access denied",
			})
			return

		case errors.Is(err, database.ErrConversationNotGroup):
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message: "target conversation is not a group",
			})
			return

		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "group not found",
			})
			return

		default:
			ctx.Logger.WithError(err).Error("cannot update group name")
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Message: "internal server error",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, conversationToResponse(conv))
}

/*
This handler implements

PUT /groups/:groupId/photo

Very similar top setGroupName
*/
func (rt *_router) setGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

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
	var req setGroupPhotoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode set-group-photo request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	// Validate photo input (Base64)
	req.Photo = strings.TrimSpace(req.Photo)
	if req.Photo == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "group photo must be a non-empty string",
		})
		return
	}

	// Ask the database layer to update the group photo.
	conv, err := rt.db.SetGroupPhoto(groupID, requesterID, req.Photo)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNotGroupMember):
			writeJSON(w, http.StatusForbidden, errorResponse{
				Message: "access denied",
			})
			return

		case errors.Is(err, database.ErrConversationNotGroup):
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Message: "target conversation is not a group",
			})
			return

		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, http.StatusNotFound, errorResponse{
				Message: "group not found",
			})
			return

		default:
			ctx.Logger.WithError(err).Error("cannot update group photo")
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Message: "internal server error",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, conversationToResponse(conv))
}
