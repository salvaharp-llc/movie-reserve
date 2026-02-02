package main

import (
	"database/sql"
	"net/http"

	"github.com/google/uuid"
)

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
