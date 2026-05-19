package api

import (
	"net/http"
	"strings"
)

func requireAPIKey(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" { // dev mode — skip auth
				next.ServeHTTP(w, r)
				return
			}
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				RespondErr(w, http.StatusUnauthorized, ErrUnauthorized, "invalid API key")
				return
			}
			bearer := strings.TrimPrefix(authHeader, "Bearer ")
			if bearer != key {
				RespondErr(w, http.StatusUnauthorized, ErrUnauthorized, "invalid API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
