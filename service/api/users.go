package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
	"github.com/mattn/go-sqlite3"
)

/*
User related endpoints fot this API
*/

// JSON modeling for requests
type updateUserNameRequest struct {
	Name string `json:"name"`
}
type updatePhotoRequest struct {
	Photo string `json:"photo"`
}

/*
Handler that manages GET /users/me
Extracts user ID from it's bearer token
Retrieves the user from the DB
Returns the profile as JSON
*/
func (rt *_router) getMyProfile(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// The authenticated user id is taken from the bearer token
	// The getBearerToken() helper is located in the helpers file
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Once the token is extracted, the handler asks the database for that user
	// The database implementation is located in /service/api/database
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

/*
This handler implements GET /users
It lists the users of the application
*/
func (rt *_router) listUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// Bearer verification
	_, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Optional query parameter used for a simple username search
	usernameFilter := r.URL.Query().Get("username")

	// Delegates filtering logic to DB
	users, err := rt.db.ListUsers(usernameFilter)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot list users")
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Message: "internal server error",
		})
		return
	}

	// Converte gli utenti in risposta JSON
	items := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		items = append(items, userToResponse(user))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

/*
This handler implements: PUT /users/me/name
*/
func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// Bearer verification
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Decode request body
	var req updateUserNameRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode update username request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	// Remove accidental spaces before validating the username length, normalize before validation
	req.Name = strings.TrimSpace(req.Name)

	// Validate username length
	if len(req.Name) < 3 || len(req.Name) > 16 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "username must be between 3 and 16 characters",
		})
		return
	}

	// Update through DB layer
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

/*
Handler that implements PUT /users/me/photo
*/
func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {

	// Bearer validation
	userID, ok := getBearerToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{
			Message: "missing or invalid Authorization header",
		})
		return
	}

	// Decode JSON body
	var req updatePhotoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode update photo request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "invalid request body",
		})
		return
	}

	// Normalize and validate photo value
	req.Photo = strings.TrimSpace(req.Photo)
	if req.Photo == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Message: "photo must be a non-empty string",
		})
		return
	}

	// DB update
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
