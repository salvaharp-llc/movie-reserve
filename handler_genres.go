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

type Genre struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
}

func (cfg *apiConfig) handlerCreateGenres(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
	}
	type response struct {
		Genre
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	if strings.TrimSpace(params.Name) == "" {
		respondWithError(w, http.StatusBadRequest, "Genre name is required", nil)
		return
	}

	genre, err := cfg.db.CreateGenre(r.Context(), params.Name)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating genre", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		Genre: Genre{
			ID:        genre.ID,
			CreatedAt: genre.CreatedAt,
			UpdatedAt: genre.UpdatedAt,
			Name:      genre.Name,
		},
	})
}

func (cfg *apiConfig) handlerGetGenres(w http.ResponseWriter, r *http.Request) {
	genreIDString := r.PathValue("genreID")
	genreID, err := uuid.Parse(genreIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	genre, err := cfg.db.GetGenreByID(r.Context(), genreID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get genre", err)
		return
	}

	respondWithJSON(w, http.StatusOK, genre)
}

func (cfg *apiConfig) handlerRetrieveGenres(w http.ResponseWriter, r *http.Request) {
	genres, err := cfg.db.GetGenres(r.Context())
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get genres", err)
		return
	}

	respondWithJSON(w, http.StatusOK, genres)
}

func (cfg *apiConfig) handlerUpdateGenres(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
	}
	type response struct {
		Genre
	}

	genreIDString := r.PathValue("genreID")
	genreID, err := uuid.Parse(genreIDString)
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

	if strings.TrimSpace(params.Name) == "" {
		respondWithError(w, http.StatusBadRequest, "Genre name is required", nil)
		return
	}

	genre, err := cfg.db.UpdateGenre(r.Context(), database.UpdateGenreParams{
		ID:   genreID,
		Name: params.Name,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Genre not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't update genre", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Genre: Genre{
			ID:        genre.ID,
			CreatedAt: genre.CreatedAt,
			UpdatedAt: genre.UpdatedAt,
			Name:      genre.Name,
		},
	})
}

func (cfg *apiConfig) handlerDeleteGenres(w http.ResponseWriter, r *http.Request) {
	genreIDString := r.PathValue("genreID")
	genreID, err := uuid.Parse(genreIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	err = cfg.db.DeleteGenre(r.Context(), genreID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Genre not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete genre", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
