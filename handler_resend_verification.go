package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/salvaharp-llc/movie-reserve/internal/auth"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

func (cfg *apiConfig) handlerResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
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

	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "error retrieving user", err)
		return
	}

	if user.IsActive {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	verificationCode := auth.MakeVerificationCode()
	_, err = cfg.db.CreateEmailVerification(r.Context(), database.CreateEmailVerificationParams{
		UserID:    user.ID,
		UserEmail: params.Email,
		Code:      verificationCode,
		ExpiresAt: time.Now().Add(auth.VerificationCodeExpiresIn),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create email verification", err)
		return
	}

	err = cfg.emailSender.SendVerificationEmail(params.Email, verificationCode)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to send verification email", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
