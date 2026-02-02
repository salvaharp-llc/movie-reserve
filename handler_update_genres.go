package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

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
