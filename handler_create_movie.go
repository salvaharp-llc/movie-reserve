package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

type Movie struct {
	ID              uuid.UUID  `json:"id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Description     *string    `json:"description,omitempty"`
	RunetimeMinutes *int32     `json:"runetime_minutes,omitempty"`
	ReleaseDate     *time.Time `json:"release_date,omitempty"`
}

func (cfg *apiConfig) handlerCreateMovies(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Title           string     `json:"title"`
		Slug            string     `json:"slug"`
		Description     *string    `json:"description"`
		RunetimeMinutes *int32     `json:"runetime_minutes"`
		ReleaseDate     *time.Time `json:"release_date"`
	}
	type response struct {
		Movie
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	if strings.TrimSpace(params.Title) == "" {
		respondWithError(w, http.StatusBadRequest, "Movie title is required", nil)
		return
	}
	if strings.TrimSpace(params.Slug) == "" {
		respondWithError(w, http.StatusBadRequest, "Movie slug is required", nil)
		return
	}

	movie, err := cfg.db.CreateMovie(r.Context(), database.CreateMovieParams{
		Title:          params.Title,
		Slug:           params.Slug,
		Description:    convertToNullString(params.Description),
		RuntimeMinutes: convertToNullInt32(params.RunetimeMinutes),
		ReleaseDate:    convertToNullTime(params.ReleaseDate),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating movie", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		Movie: Movie{
			ID:              movie.ID,
			CreatedAt:       movie.CreatedAt,
			UpdatedAt:       movie.UpdatedAt,
			Title:           movie.Title,
			Slug:            movie.Slug,
			Description:     nullStringToPointer(movie.Description),
			RunetimeMinutes: nullInt32ToPointer(movie.RuntimeMinutes),
			ReleaseDate:     nullTimeToPointer(movie.ReleaseDate),
		},
	})
}

// Helper functions to convert pointers to sql.Null types
func convertToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

func convertToNullInt32(i *int32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: *i, Valid: true}
}

func convertToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// Helper functions to convert sql.Null types to pointers
func nullStringToPointer(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func nullInt32ToPointer(ni sql.NullInt32) *int32 {
	if ni.Valid {
		return &ni.Int32
	}
	return nil
}

func nullTimeToPointer(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
