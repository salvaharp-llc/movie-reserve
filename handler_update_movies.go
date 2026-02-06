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

func (cfg *apiConfig) handlerUpdateMovies(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Title           string     `json:"title"`
		Slug            string     `json:"slug"`
		Description     *string    `json:"description"`
		RunetimeMinutes *int32     `json:"runetime_minutes"`
		ReleaseDate     *time.Time `json:"release_date"`
		GenreIDs        []string   `json:"genre_ids"`
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

	genreUUIDs := make([]uuid.UUID, len(params.GenreIDs))
	for i, genreIDStr := range params.GenreIDs {
		genreID, err := uuid.Parse(genreIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid genre ID format", err)
			return
		}
		genreUUIDs[i] = genreID
	}

	currentMovie, err := cfg.db.GetMovieByID(r.Context(), movieID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Movie not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch current movie", err)
		return
	}

	var genres []database.Genre
	if len(genreUUIDs) > 0 {
		genres, err = cfg.db.GetGenresByIDs(r.Context(), genreUUIDs)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error fetching genres", err)
			return
		}

		if len(genres) != len(genreUUIDs) {
			respondWithError(w, http.StatusBadRequest, "One or more genre IDs not found", nil)
			return
		}
	}

	movie, err := cfg.db.UpdateMovie(r.Context(), database.UpdateMovieParams{
		ID:             movieID,
		Title:          params.Title,
		Slug:           params.Slug,
		Description:    convertToNullString(params.Description),
		RuntimeMinutes: convertToNullInt32(params.RunetimeMinutes),
		ReleaseDate:    convertToNullTime(params.ReleaseDate),
		PosterUrl:      currentMovie.PosterUrl,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Movie not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't update movie", err)
		return
	}

	if len(genreUUIDs) > 0 {
		err = cfg.db.DeleteMovieGenres(r.Context(), movieID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error deleting old genres", err)
			return
		}

		err = cfg.db.AssignGenresToMovie(r.Context(), database.AssignGenresToMovieParams{
			MovieID: movieID,
			Column2: genreUUIDs,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error assigning genres to movie", err)
			return
		}
	}

	responseGenres := make([]Genre, len(genres))
	for i, dbGenre := range genres {
		responseGenres[i] = Genre{
			ID:        dbGenre.ID,
			CreatedAt: dbGenre.CreatedAt,
			UpdatedAt: dbGenre.UpdatedAt,
			Name:      dbGenre.Name,
		}
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
			Genres:          responseGenres,
			PosterUrl:       nullStringToPointer(movie.PosterUrl),
		},
	})
}
