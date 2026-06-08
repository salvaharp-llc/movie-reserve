package main

import (
	"context"
	"database/sql"
	"errors"
	"os"

	"github.com/salvaharp-llc/movie-reserve/internal/auth"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

func (cfg *apiConfig) ensureAdmin() error {
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		return errors.New("ADMIN_EMAIL must be set")
	}
	pass := os.Getenv("ADMIN_PASSWORD")
	if pass == "" {
		return errors.New("ADMIN_PASSWORD must be set")
	}

	user, err := cfg.db.GetUserByEmail(context.Background(), email)
	if err == nil {
		if user.Role != auth.RoleAdmin {
			return errors.New("user exists but is not an admin")
		}
		match, err := auth.CheckPasswordHash(pass, user.HashedPassword)
		if err != nil {
			return err
		}
		if !match {
			return errors.New("admin user exists but password does not match")
		}
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}

	hashed, err := auth.HashPassword(pass)
	if err != nil {
		return err
	}

	_, err = cfg.db.CreateAdmin(context.Background(), database.CreateAdminParams{
		Email:          email,
		HashedPassword: hashed,
	})
	return err
}
