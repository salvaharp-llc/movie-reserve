package main

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/auth"
)

type contextKey string

const (
	userIDKey   contextKey = "userID"
	userRoleKey contextKey = "userRole"
)

func (cfg *apiConfig) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
			return
		}

		userID, role, err := auth.ValidateJWT(token, cfg.jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, userRoleKey, role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin should be chained after RequireAuth
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(userRoleKey).(string)
		if !ok {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}

		if role != auth.RoleAdmin {
			respondWithError(w, http.StatusForbidden, "Forbidden", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

func GetUserRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(userRoleKey).(string)
	return role, ok
}
