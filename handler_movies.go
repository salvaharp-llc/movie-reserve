package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

type MovieSummary struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	PosterUrl *string   `json:"poster_url"`
}

type MovieDetail struct {
	ID             uuid.UUID  `json:"id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Description    *string    `json:"description"`
	RuntimeMinutes *int32     `json:"runtime_minutes"`
	ReleaseDate    *time.Time `json:"release_date"`
	PosterUrl      *string    `json:"poster_url"`
	Genres         []Genre    `json:"genres"`
}

func (cfg *apiConfig) handlerCreateMovies(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Title          string     `json:"title"`
		Slug           string     `json:"slug"`
		Description    *string    `json:"description"`     // optional
		RuntimeMinutes *int32     `json:"runtime_minutes"` // optional
		ReleaseDate    *time.Time `json:"release_date"`    // optional
		GenreIDs       []string   `json:"genre_ids"`
	}
	type response struct {
		MovieDetail `json:"movie"`
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

	if len(params.GenreIDs) == 0 {
		respondWithError(w, http.StatusBadRequest, "At least one genre ID is required", nil)
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

	existingGenres, err := cfg.db.GetGenresByIDs(r.Context(), genreUUIDs)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching genres", err)
		return
	}

	if len(existingGenres) != len(genreUUIDs) {
		respondWithError(w, http.StatusBadRequest, "One or more genre IDs not found", nil)
		return
	}

	desc := sql.NullString{}
	if params.Description != nil {
		if strings.TrimSpace(*params.Description) == "" {
			respondWithError(w, http.StatusBadRequest, "Description cannot be empty if provided", nil)
			return
		}
		desc.String = *params.Description
		desc.Valid = true
	}

	runtime := sql.NullInt32{}
	if params.RuntimeMinutes != nil {
		if *params.RuntimeMinutes < 0 {
			respondWithError(w, http.StatusBadRequest, "Runtime minutes must be non-negative", nil)
			return
		}
		runtime.Int32 = *params.RuntimeMinutes
		runtime.Valid = true // explicit zero allowed
	}

	relDate := sql.NullTime{}
	if params.ReleaseDate != nil {
		if params.ReleaseDate.IsZero() {
			respondWithError(w, http.StatusBadRequest, "Release date cannot be empty if provided", nil)
			return
		}
		relDate.Time = *params.ReleaseDate
		relDate.Valid = true
	}

	movie, err := cfg.db.CreateMovie(r.Context(), database.CreateMovieParams{
		Title:          params.Title,
		Slug:           params.Slug,
		Description:    desc,
		RuntimeMinutes: runtime,
		ReleaseDate:    relDate,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating movie", err)
		return
	}

	err = cfg.db.AssignGenresToMovie(r.Context(), database.AssignGenresToMovieParams{
		MovieID:  movie.ID,
		GenreIds: genreUUIDs,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error assigning genres to movie", err)
		return
	}

	movieDetail, err := cfg.db.GetMovieDetailByID(r.Context(), movie.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch created movie", err)
		return
	}

	movieResponse := aggregateMovieDetail(movieDetail)

	respondWithJSON(w, http.StatusCreated, response{
		MovieDetail: movieResponse,
	})
}

func (cfg *apiConfig) handlerGetMovies(w http.ResponseWriter, r *http.Request) {
	type response struct {
		MovieDetail `json:"movie"`
	}

	movieIDString := r.PathValue("movieID")
	movieID, err := uuid.Parse(movieIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	movie, err := cfg.db.GetMovieDetailByID(r.Context(), movieID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Movie not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch movie", err)
		return
	}

	movieResponse := aggregateMovieDetail(movie)

	respondWithJSON(w, http.StatusOK, response{
		MovieDetail: movieResponse,
	})
}

func (cfg *apiConfig) handlerRetrieveMovies(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Movies []MovieSummary `json:"movies"`
	}

	q := r.URL.Query()

	limit, offset := 100, 0

	if limitStr := q.Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 {
			respondWithError(w, http.StatusBadRequest, "Invalid limit", err)
			return
		}
		limit = parsed
	}

	if offsetStr := q.Get("offset"); offsetStr != "" {
		parsed, err := strconv.Atoi(offsetStr)
		if err != nil || parsed < 0 {
			respondWithError(w, http.StatusBadRequest, "Invalid offset", err)
			return
		}
		offset = parsed
	}

	genreID, err := parseNullUUID(q.Get("genre_id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid genre_id", err)
		return
	}
	timeParams := map[string]sql.NullTime{}
	for _, key := range []string{"release_date_from", "release_date_to"} {
		parsed, err := parseNullTime(q.Get(key))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid "+key+" date", err)
			return
		}
		timeParams[key] = parsed
	}
	intParams := map[string]sql.NullInt32{}
	for _, key := range []string{"runtime_min", "runtime_max"} {
		parsed, err := parseNullInt32(q.Get(key))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid "+key, err)
			return
		}
		intParams[key] = parsed
	}

	if intParams["runtime_min"].Valid && intParams["runtime_max"].Valid &&
		intParams["runtime_min"].Int32 > intParams["runtime_max"].Int32 {
		respondWithError(w, http.StatusBadRequest, "runtime_min cannot be greater than runtime_max", nil)
		return
	}
	if timeParams["release_date_from"].Valid && timeParams["release_date_to"].Valid &&
		timeParams["release_date_from"].Time.After(timeParams["release_date_to"].Time) {
		respondWithError(w, http.StatusBadRequest, "release_date_from cannot be after release_date_to", nil)
		return
	}

	movies, err := cfg.db.GetMoviesSummary(r.Context(), database.GetMoviesSummaryParams{
		GenreID:         genreID,
		Title:           sql.NullString{String: q.Get("title"), Valid: strings.TrimSpace(q.Get("title")) != ""},
		ReleaseDateFrom: timeParams["release_date_from"],
		ReleaseDateTo:   timeParams["release_date_to"],
		RuntimeMin:      intParams["runtime_min"],
		RuntimeMax:      intParams["runtime_max"],
		Limit:           int32(limit),
		Offset:          int32(offset),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get movies", err)
		return
	}

	responseMovies := make([]MovieSummary, len(movies))
	for i, r := range movies {
		responseMovies[i] = MovieSummary{
			ID:        r.ID,
			Title:     r.Title,
			Slug:      r.Slug,
			PosterUrl: nullStringToPtr(r.PosterUrl),
		}
	}

	respondWithJSON(w, http.StatusOK, response{
		Movies: responseMovies,
	})
}

func (cfg *apiConfig) handlerRetrieveCurrentMovies(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Movies []MovieSummary `json:"movies"`
	}

	movies, err := cfg.db.GetCurrentMoviesSummary(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get current movies", err)
		return
	}

	responseMovies := make([]MovieSummary, len(movies))
	for i, r := range movies {
		responseMovies[i] = MovieSummary{
			ID:        r.ID,
			Title:     r.Title,
			Slug:      r.Slug,
			PosterUrl: nullStringToPtr(r.PosterUrl),
		}
	}

	respondWithJSON(w, http.StatusOK, response{
		Movies: responseMovies,
	})
}

func (cfg *apiConfig) handlerUpdateMovies(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Title          string     `json:"title"`
		Slug           string     `json:"slug"`
		Description    *string    `json:"description"`     // optional
		RuntimeMinutes *int32     `json:"runtime_minutes"` // optional
		ReleaseDate    *time.Time `json:"release_date"`    // optional
		GenreIDs       []string   `json:"genre_ids"`
	}
	type response struct {
		MovieDetail `json:"movie"`
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

	if len(params.GenreIDs) == 0 {
		respondWithError(w, http.StatusBadRequest, "At least one genre ID is required", nil)
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

	existingGenres, err := cfg.db.GetGenresByIDs(r.Context(), genreUUIDs)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching genres", err)
		return
	}

	if len(existingGenres) != len(genreUUIDs) {
		respondWithError(w, http.StatusBadRequest, "One or more genre IDs not found", nil)
		return
	}

	desc := sql.NullString{}
	if params.Description != nil {
		if strings.TrimSpace(*params.Description) == "" {
			respondWithError(w, http.StatusBadRequest, "Description cannot be empty if provided", nil)
			return
		}
		desc.String = *params.Description
		desc.Valid = true
	}

	runtime := sql.NullInt32{}
	if params.RuntimeMinutes != nil {
		if *params.RuntimeMinutes < 0 {
			respondWithError(w, http.StatusBadRequest, "Runtime minutes must be non-negative", nil)
			return
		}
		runtime.Int32 = *params.RuntimeMinutes
		runtime.Valid = true
	}

	relDate := sql.NullTime{}
	if params.ReleaseDate != nil {
		if params.ReleaseDate.IsZero() {
			respondWithError(w, http.StatusBadRequest, "Release date cannot be empty if provided", nil)
			return
		}
		relDate.Time = *params.ReleaseDate
		relDate.Valid = true
	}

	_, err = cfg.db.UpdateMovie(r.Context(), database.UpdateMovieParams{
		ID:             movieID,
		Title:          params.Title,
		Slug:           params.Slug,
		Description:    desc,
		RuntimeMinutes: runtime,
		ReleaseDate:    relDate,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Movie not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error updating movie", err)
		return
	}

	err = cfg.db.DeleteMovieGenres(r.Context(), movieID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting old genres", err)
		return
	}

	err = cfg.db.AssignGenresToMovie(r.Context(), database.AssignGenresToMovieParams{
		MovieID:  movieID,
		GenreIds: genreUUIDs,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error assigning genres to movie", err)
		return
	}

	movieDetail, err := cfg.db.GetMovieDetailByID(r.Context(), movieID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch updated movie", err)
		return
	}

	movieResponse := aggregateMovieDetail(movieDetail)

	respondWithJSON(w, http.StatusOK, response{
		MovieDetail: movieResponse,
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
			respondWithError(w, http.StatusNotFound, "Movie not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete movie", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func aggregateMovieDetail(r database.GetMovieDetailByIDRow) MovieDetail {
	var genres []Genre
	if r.Genres != nil {
		// sqlc returns aggregated JSON columns as interface{} which may be
		// []byte or string depending on the driver; we need this type switch
		// to unmarshal it correctly.
		switch v := r.Genres.(type) {
		case []byte:
			_ = json.Unmarshal(v, &genres)
		case string:
			_ = json.Unmarshal([]byte(v), &genres)
		default:
			// fallback if sqlc already gave us a slice (unlikely)
			if b, err := json.Marshal(v); err == nil {
				_ = json.Unmarshal(b, &genres)
			}
		}
	}

	return MovieDetail{
		ID:             r.ID,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
		Title:          r.Title,
		Slug:           r.Slug,
		Description:    nullStringToPtr(r.Description),
		RuntimeMinutes: nullInt32ToPtr(r.RuntimeMinutes),
		ReleaseDate:    nullTimeToPtr(r.ReleaseDate),
		PosterUrl:      nullStringToPtr(r.PosterUrl),
		Genres:         genres,
	}
}
