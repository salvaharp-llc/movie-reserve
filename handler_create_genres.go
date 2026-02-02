package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
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
