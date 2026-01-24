package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

func (cfg *apiConfig) handlerUpdateMovies(w http.ResponseWriter, r *http.Request) {
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

	movieIDString := r.PathValue("movieID")
	movieID, err := uuid.Parse(movieIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
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

	movie, err := cfg.db.UpdateMovie(r.Context(), database.UpdateMovieParams{
		ID:             movieID,
		Title:          params.Title,
		Slug:           params.Slug,
		Description:    convertToNullString(params.Description),
		RuntimeMinutes: convertToNullInt32(params.RunetimeMinutes),
		ReleaseDate:    convertToNullTime(params.ReleaseDate),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating movie", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
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
