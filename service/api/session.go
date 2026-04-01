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

type loginRequest struct {
	Name string `json:"name"`
}

type loginResponse struct {
	Identifier string `json:"identifier"`
}

type errorResponse struct {
	Message string `json:"message"`
}

func (rt *_router) doLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	var req loginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ctx.Logger.WithError(err).Error("cannot decode login request body")
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)

	if len(req.Name) < 3 || len(req.Name) > 16 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Message: "username must be between 3 and 16 characters"})
		return
	}

	user, err := rt.db.GetUserByUsername(req.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			user, err = rt.db.CreateUser(req.Name)
			if err != nil {
				ctx.Logger.WithError(err).Error("cannot create user")
				writeJSON(w, http.StatusInternalServerError, errorResponse{Message: "internal server error"})
				return
			}
		} else {
			ctx.Logger.WithError(err).Error("cannot search user by username")
			writeJSON(w, http.StatusInternalServerError, errorResponse{Message: "internal server error"})
			return
		}
	}

	writeJSON(w, http.StatusCreated, loginResponse{
		Identifier: user.ID,
	})
}
