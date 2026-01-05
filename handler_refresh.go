package main

import (
	"net/http"
	"time"

	"github.com/salvaharp-llc/movie-reserve/internal/auth"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not find token in header", err)
		return
	}

	accessInfo, err := cfg.db.RevokeRefreshToken(r.Context(), auth.HashRefreshToken(refreshToken))
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid refresh token", err)
		return
	}

	newRefreshToken := auth.MakeRefreshToken()

	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     auth.HashRefreshToken(newRefreshToken),
		UserID:    accessInfo.UserID,
		ExpiresAt: time.Now().Add(auth.RefreshTokenExpiresIn),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not save refresh token", err)
		return
	}

	accessToken, err := auth.MakeJWT(
		accessInfo.UserID,
		accessInfo.Role,
		cfg.JWTSecret,
		auth.JwtExpiresIn)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not create access JWT", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
	})
}
