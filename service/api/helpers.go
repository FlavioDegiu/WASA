package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

// writeJSON standardizes JSON responses for all handlers
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data == nil {
		return
	}

	_ = json.NewEncoder(w).Encode(data)
}

// getBearerToken extracts the user identifier stored in the Authorization header
func getBearerToken(r *http.Request) (string, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", false
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if token == "" {
		return "", false
	}

	return token, true
}

// userToResponse converts the internal database model into the public API shape
func userToResponse(user database.User) map[string]interface{} {
	resp := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
	}

	// Only include optional fields when they actually contain data
	if user.Photo != "" {
		resp["photo"] = user.Photo
	}

	return resp
}

// This regex is a practical approximation for emoji validation, it accepts most common emoji and emoji presentation characters
var emojiRegex = regexp.MustCompile(`^(?:[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{2199}\x{21A9}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{27BF}\x{2934}-\x{2935}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F000}-\x{1FAFF}](?:\x{FE0F}|\x{200D}[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{2199}\x{21A9}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{27BF}\x{2934}-\x{2935}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F000}-\x{1FAFF}])*)$`)

func isSingleEmoji(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	// Reject obviously long text.
	if utf8.RuneCountInString(s) > 8 {
		return false
	}

	return emojiRegex.MatchString(s)
}
