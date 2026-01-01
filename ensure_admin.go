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

	_, err := cfg.db.GetUserByEmail(context.Background(), email)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}

	hashed, err := auth.HashPassword(pass)
	if err != nil {
		return err
	}

	user, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{
		Email:          email,
		HashedPassword: hashed,
	})
	if err != nil {
		return err
	}

	_, err = cfg.db.MakeAdmin(context.Background(), user.ID)
	return err
}
