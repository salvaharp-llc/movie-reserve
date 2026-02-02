package main

import (
	"database/sql"
	"net/http"

	"github.com/google/uuid"
)

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
