package main

import (
	"database/sql"
	"encoding/json"
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
		UserEmail: params.Email,
		Code:      verificationCode,
		ExpiresAt: time.Now().Add(auth.VerificationCodeExpiresIn),
	})
	if err != nil {
		log.Printf("Failed to create email verification for user %s: %s", user.ID.String(), err.Error())
	} else {
		err = cfg.emailSender.SendVerificationEmail(params.Email, verificationCode)
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

func (cfg *apiConfig) handlerVerifyEmail(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Code  string `json:"code"`
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
			respondWithError(w, http.StatusBadRequest, "Wrong email or verification code", nil)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Database error", err)
		return
	}

	if user.IsActive {
		respondWithError(w, http.StatusBadRequest, "Wrong email or verification code", nil)
		return
	}

	verification, err := cfg.db.GetEmailVerificationByEmail(r.Context(), params.Email)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusBadRequest, "Wrong email or verification code", nil)

		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting email verification", err)
		return
	}

	if verification.ExpiresAt.Before(time.Now()) {
		respondWithError(w, http.StatusBadRequest, "Verification code has expired", nil)
		return
	}

	if verification.Code != params.Code {
		respondWithError(w, http.StatusBadRequest, "Wrong email or verification code", nil)
		return
	}

	err = cfg.db.VerifyUser(r.Context(), verification.UserEmail)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error verifying user", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		respondWithError(w, http.StatusInternalServerError, "Error revoking refresh tokens after password update", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerUpdateEmail(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		NewEmail string `json:"new_email"`
	}
	type response struct {
		User
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

	if strings.TrimSpace(params.Password) == "" {
		respondWithError(w, http.StatusBadRequest, "password required", nil)
		return
	}
	if strings.TrimSpace(params.NewEmail) == "" {
		respondWithError(w, http.StatusBadRequest, "new email required", nil)
		return
	}

	if !email.IsValidEmail(params.NewEmail) {
		respondWithError(w, http.StatusBadRequest, "invalid email address", nil)
		return
	}

	user, err := cfg.db.GetUserByID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error geting user", err)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not check password", err)
		return
	}
	if !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect password", err)
		return
	}

	err = cfg.db.UpdateUserEmail(r.Context(), database.UpdateUserEmailParams{
		ID:    userID,
		Email: params.NewEmail,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating email", err)
		return
	}

	err = cfg.db.UnverifyUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error unverifying user after email update", err)
		return
	}

	verificationCode := auth.MakeVerificationCode()

	_, err = cfg.db.CreateEmailVerification(r.Context(), database.CreateEmailVerificationParams{
		UserID:    userID,
		UserEmail: params.NewEmail,
		Code:      verificationCode,
		ExpiresAt: time.Now().Add(auth.VerificationCodeExpiresIn),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating email verification", err)
		return
	} else {
		err = cfg.emailSender.SendVerificationEmail(params.NewEmail, verificationCode)
		if err != nil {
			log.Printf("Failed to send verification email to %s after email update: %s", params.NewEmail, err.Error())
		}
	}

	err = cfg.db.RevokeRefreshTokens(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error revoking refresh tokens after email update", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		User: User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
			Role:      user.Role,
		},
	})
}

func (cfg *apiConfig) handlerPasswordReset(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	if strings.TrimSpace(params.Token) == "" {
		respondWithError(w, http.StatusBadRequest, "token required", nil)
		return
	}
	if strings.TrimSpace(params.NewPassword) == "" {
		respondWithError(w, http.StatusBadRequest, "new password required", nil)
		return
	}

	hashedToken := auth.HashPwResetToken(params.Token)

	resetTokenRecord, err := cfg.db.GetPasswordResetToken(r.Context(), hashedToken)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Wrong token or token has expired", nil)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "error getting password reset token", err)
		return
	}

	if resetTokenRecord.ExpiresAt.Before(time.Now()) || resetTokenRecord.RevokedAt.Valid {
		respondWithError(w, http.StatusBadRequest, "Wrong token or token has expired", nil)
		return
	}

	err = cfg.db.RevokePasswordResetToken(r.Context(), hashedToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error revoking password reset token", err)
		return
	}

	newHashedPassword, err := auth.HashPassword(params.NewPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not hash new password", err)
		return
	}

	err = cfg.db.UpdateUserPassword(r.Context(), database.UpdateUserPasswordParams{
		ID:             resetTokenRecord.UserID,
		HashedPassword: newHashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating password", err)
		return
	}

	err = cfg.db.RevokeRefreshTokens(r.Context(), resetTokenRecord.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error revoking refresh tokens after password reset", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
