package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/salvaharp-llc/movie-reserve/internal/auth"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
	"github.com/salvaharp-llc/movie-reserve/internal/email"
)

func (cfg *apiConfig) handlerRequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	if strings.TrimSpace(params.Email) == "" {
		respondWithError(w, http.StatusBadRequest, "email required", nil)
		return
	}

	if !email.IsValidEmail(params.Email) {
		respondWithError(w, http.StatusBadRequest, "invalid email address", nil)
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "error getting user", err)
		return
	}

	if !user.IsActive {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resetToken := auth.MakePwResetToken()
	hashedToken := auth.HashPwResetToken(resetToken)

	_, err = cfg.db.CreatePasswordResetToken(r.Context(), database.CreatePasswordResetTokenParams{
		HashedToken: hashedToken,
		UserID:      user.ID,
		ExpiresAt:   time.Now().Add(auth.PwResetTokenExpiresIn),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating password reset token", err)
		return
	}

	err = cfg.emailSender.SendEmail(
		params.Email,
		"Movie Reserve - Password Reset Request",
		"You requested a password reset. Use the following link to reset your password: "+
			"http://localhost:"+cfg.port+"/app/users/password-reset?token="+resetToken,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error sending password reset email", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
