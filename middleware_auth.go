package main

import (
	"context"
	"errors"
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
			respondWithError(w, http.StatusUnauthorized, "Could not get token from header", err)
			return
		}

		userID, role, err := auth.ValidateJWT(token, cfg.jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
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
		role, err := GetUserRole(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not find user role", err)
			return
		}

		if role != auth.RoleAdmin {
			respondWithError(w, http.StatusForbidden, "Admin users only", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func GetUserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, errors.New("user id missing from context; ensure RequireAuth middleware is applied")
	}
	return userID, nil
}

func GetUserRole(ctx context.Context) (string, error) {
	role, ok := ctx.Value(userRoleKey).(string)
	if !ok {
		return "", errors.New("user role missing from context; ensure RequireAuth middleware is applied")
	}
	return role, nil
}
