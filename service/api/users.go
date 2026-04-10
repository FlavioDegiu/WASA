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
	"github.com/mattn/go-sqlite3"
)

type updateUserNameRequest struct {
	Name string `json:"name"`
}
type updatePhotoRequest struct {
	Photo string `json:"photo"`
}

// Converts a database user into the full API response object.
func usersToResponse(users []database.User) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, userToResponse(user))
	}
	return items
}

func (rt *_router) getMyProfile(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	// The authenticated user id is taken from the bearer token
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	user, err := rt.db.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Message: "invalid user identifier",
			})
			return
		}

		ctx.Logger.WithError(err).Error("cannot get user by id")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (rt *_router) listUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	_, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Optional query parameter used for a simple username search
	usernameFilter := r.URL.Query().Get("username")

	users, err := rt.db.ListUsers(usernameFilter)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list users")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, userToResponse(user))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	var req updateUserNameRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode update username request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	// Remove accidental spaces before validating the username length
	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 3 || len(req.Name) > 16 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "username must be between 3 and 16 characters",
		})
		return
	}

	user, err := rt.db.UpdateUserName(userID, req.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Message: "invalid user identifier",
			})
			return
		}

		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			writeJSON(w, http.StatusConflict, errorResponse{
				Message: "username already in use",
			})
			return
		}

		ctx.Logger.WithError(err).Error("cannot update username")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	var req updatePhotoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode update photo request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	req.Photo = strings.TrimSpace(req.Photo)
	if req.Photo == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "photo must be a non-empty string",
		})
		return
	}

	user, err := rt.db.UpdateUserPhoto(userID, req.Photo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{
				Message: "invalid user identifier",
			})
			return
		}

		ctx.Logger.WithError(err).Error("cannot update user photo")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}
