package main

import (
	"net/http"

	"github.com/salvaharp-llc/movie-reserve/internal/auth"
)

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not get token from header", err)
		return
	}

	_, err = cfg.db.RevokeRefreshToken(r.Context(), auth.HashRefreshToken(refreshToken))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not revoke the refresh token", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
