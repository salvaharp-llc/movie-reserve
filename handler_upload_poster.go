package main

import (
	"database/sql"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

func (cfg *apiConfig) handlerUploadPoster(w http.ResponseWriter, r *http.Request) {
	type response struct {
		MovieDetail `json:"movie"`
	}

	movieIDString := r.PathValue("movieID")
	movieID, err := uuid.Parse(movieIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse request body", err)
		return
	}

	file, header, err := r.FormFile("poster")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type header", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid media type", nil)
		return
	}

	assetPath := getAssetPath(mediaType)
	filePath := cfg.getAssetDiskPath(assetPath)

	dst, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file on server", err)
		return
	}
	defer dst.Close()
	_, err = io.Copy(dst, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save file", err)
		return
	}

	assetURL := cfg.getAssetURL(assetPath)

	err = cfg.db.UploadMoviePoster(r.Context(), database.UploadMoviePosterParams{
		ID:        movieID,
		PosterUrl: sql.NullString{String: assetURL, Valid: true},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update movie", err)
		return
	}

	movie, err := cfg.db.GetMovieDetailByID(r.Context(), movieID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch updated movie", err)
		return
	}

	responseMovie := aggregateMovieDetail(movie)

	respondWithJSON(w, http.StatusOK, response{
		MovieDetail: responseMovie,
	})
}
