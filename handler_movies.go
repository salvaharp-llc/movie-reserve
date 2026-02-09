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
	Genres          []Genre    `json:"genres,omitempty"`
	PosterUrl       *string    `json:"poster_url,omitempty"`
}

func (cfg *apiConfig) handlerCreateMovies(w http.ResponseWriter, r *http.Request) {
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

	genreUUIDs := make([]uuid.UUID, len(params.GenreIDs))
	for i, genreIDStr := range params.GenreIDs {
		genreID, err := uuid.Parse(genreIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid genre ID format", err)
			return
		}
		genreUUIDs[i] = genreID
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

	if len(genreUUIDs) > 0 {
		err = cfg.db.AssignGenresToMovie(r.Context(), database.AssignGenresToMovieParams{
			MovieID: movie.ID,
			Column2: genreUUIDs,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error assigning genres to movie", err)
			return
		}
	}

	rows, err := cfg.db.GetMovieWithGenresByID(r.Context(), movie.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch created movie", err)
		return
	}

	createdMovie := aggregateMovieWithGenres(rows)

	respondWithJSON(w, http.StatusCreated, response{
		Movie: createdMovie,
	})
}

func (cfg *apiConfig) handlerGetMovies(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Movie
	}

	movieIDString := r.PathValue("movieID")
	movieID, err := uuid.Parse(movieIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	rows, err := cfg.db.GetMovieWithGenresByID(r.Context(), movieID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch movie", err)
		return
	}

	if len(rows) == 0 {
		respondWithError(w, http.StatusNotFound, "Movie not found", nil)
		return
	}

	movie := aggregateMovieWithGenres(rows)

	respondWithJSON(w, http.StatusOK, response{
		Movie: movie,
	})
}

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
			respondWithError(w, http.StatusNotFound, "Movie not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch current movie", err)
		return
	}

	if len(genreUUIDs) > 0 {
		genres, err := cfg.db.GetGenresByIDs(r.Context(), genreUUIDs)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error fetching genres", err)
			return
		}

		if len(genres) != len(genreUUIDs) {
			respondWithError(w, http.StatusBadRequest, "One or more genre IDs not found", nil)
			return
		}
	}

	_, err = cfg.db.UpdateMovie(r.Context(), database.UpdateMovieParams{
		ID:             movieID,
		Title:          params.Title,
		Slug:           params.Slug,
		Description:    convertToNullString(params.Description),
		RuntimeMinutes: convertToNullInt32(params.RunetimeMinutes),
		ReleaseDate:    convertToNullTime(params.ReleaseDate),
		PosterUrl:      currentMovie.PosterUrl,
	})
	if err != nil {
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

	rows, err := cfg.db.GetMovieWithGenresByID(r.Context(), movieID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch updated movie", err)
		return
	}

	updatedMovie := aggregateMovieWithGenres(rows)

	respondWithJSON(w, http.StatusOK, response{
		Movie: updatedMovie,
	})
}

func (cfg *apiConfig) handlerDeleteMovies(w http.ResponseWriter, r *http.Request) {
	movieIDString := r.PathValue("movieID")
	movieID, err := uuid.Parse(movieIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	err = cfg.db.DeleteMovie(r.Context(), movieID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Movie not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete movie", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func aggregateMovieWithGenres(rows []database.GetMovieWithGenresByIDRow) Movie {
	if len(rows) == 0 {
		return Movie{}
	}

	firstRow := rows[0]

	genreMap := make(map[uuid.UUID]Genre)
	for _, row := range rows {
		if row.GenreID.Valid {
			genreMap[row.GenreID.UUID] = Genre{
				ID:        row.GenreID.UUID,
				CreatedAt: row.GenreCreatedAt.Time,
				UpdatedAt: row.GenreUpdatedAt.Time,
				Name:      row.GenreName.String,
			}
		}
	}

	genres := make([]Genre, 0, len(genreMap))
	for _, g := range genreMap {
		genres = append(genres, g)
	}

	return Movie{
		ID:              firstRow.ID,
		CreatedAt:       firstRow.CreatedAt,
		UpdatedAt:       firstRow.UpdatedAt,
		Title:           firstRow.Title,
		Slug:            firstRow.Slug,
		Description:     nullStringToPointer(firstRow.Description),
		RunetimeMinutes: nullInt32ToPointer(firstRow.RuntimeMinutes),
		ReleaseDate:     nullTimeToPointer(firstRow.ReleaseDate),
		Genres:          genres,
		PosterUrl:       nullStringToPointer(firstRow.PosterUrl),
	}
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
