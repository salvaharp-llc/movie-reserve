package main

import (
	"database/sql"
	"log"
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

	refreshTokenInfo, err := cfg.db.GetRefreshToken(r.Context(), auth.HashRefreshToken(refreshToken))
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Invalid refresh token", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Could not get refresh token", err)
		return
	}

	if refreshTokenInfo.RevokedAt.Valid {
		err = cfg.db.RevokeRefreshTokens(r.Context(), refreshTokenInfo.UserID)
		if err != nil {
			log.Printf("Error revoking refresh tokens for user %d: %v", refreshTokenInfo.UserID, err)
		}
		respondWithError(w, http.StatusUnauthorized, "Refresh token is revoked", nil)
		return
	}

	if refreshTokenInfo.ExpiresAt.Before(time.Now()) {
		respondWithError(w, http.StatusUnauthorized, "Refresh token is expired", nil)
		return
	}

	newRefreshToken := auth.MakeRefreshToken()

	accessToken, err := auth.MakeJWT(
		refreshTokenInfo.UserID,
		refreshTokenInfo.Role,
		cfg.jwtSecret,
		auth.JwtExpiresIn)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create access JWT", err)
		return
	}

	err = cfg.db.ExecTx(r.Context(), func(q *database.Queries) error {
		err = q.RevokeRefreshToken(r.Context(), refreshTokenInfo.Token)
		if err != nil {
			return err
		}

		_, err = q.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			Token:     auth.HashRefreshToken(newRefreshToken),
			UserID:    refreshTokenInfo.UserID,
			ExpiresAt: time.Now().Add(auth.RefreshTokenExpiresIn),
		})
		return err
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not rotate refresh token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
	})
}
