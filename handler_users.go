package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/auth"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
	"github.com/salvaharp-llc/movie-reserve/internal/email"
)

type UserSummary struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
}

func (cfg *apiConfig) handlerCreateUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		User
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
	if strings.TrimSpace(params.Password) == "" {
		respondWithError(w, http.StatusBadRequest, "password required", nil)
		return
	}

	if !email.IsValidEmail(params.Email) {
		respondWithError(w, http.StatusBadRequest, "invalid email address", nil)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error hashing password", err)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user", err)
		return
	}

	verificationCode := auth.MakeVerificationCode()

	_, err = cfg.db.CreateEmailVerification(r.Context(), database.CreateEmailVerificationParams{
		UserID:    user.ID,
		Code:      verificationCode,
		ExpiresAt: time.Now().Add(auth.VerificationCodeExpiresIn),
	})
	if err != nil {
		log.Printf("Failed to create email verification for user %s: %s", user.ID.String(), err.Error())
	} else {
		err = cfg.emailSender.SendEmail(params.Email, "Movie Reserve - Verify your email", fmt.Sprintf("Your verification code is: %d", verificationCode))
		if err != nil {
			log.Printf("Failed to send verification email to %s: %s", params.Email, err.Error())
		}
	}

	respondWithJSON(w, http.StatusCreated, response{
		User: User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
			Role:      user.Role,
		},
	})
}

func (cfg *apiConfig) handlerUpdatePassword(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	userID, err := getUserID(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user id", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	if strings.TrimSpace(params.CurrentPassword) == "" {
		respondWithError(w, http.StatusBadRequest, "current password required", nil)
		return
	}
	if strings.TrimSpace(params.NewPassword) == "" {
		respondWithError(w, http.StatusBadRequest, "new password required", nil)
		return
	}

	if params.CurrentPassword == params.NewPassword {
		respondWithError(w, http.StatusBadRequest, "new password cannot be the same as current password", nil)
		return
	}

	user, err := cfg.db.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error geting user", err)
		return
	}

	match, err := auth.CheckPasswordHash(params.CurrentPassword, user.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not check password", err)
		return
	}
	if !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect password", err)
		return
	}

	newHashedPassword, err := auth.HashPassword(params.NewPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not hash new password", err)
		return
	}

	err = cfg.db.UpdateUserPassword(r.Context(), database.UpdateUserPasswordParams{
		ID:             userID,
		HashedPassword: newHashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating password", err)
		return
	}

	err = cfg.db.RevokeRefreshTokens(r.Context(), userID)
	if err != nil {
		log.Printf("Couldn't revoke refresh tokens after password update for user %s: %s", userID.String(), err.Error())
	}

	w.WriteHeader(http.StatusNoContent)
}
