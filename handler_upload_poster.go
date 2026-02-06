package main

import (
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

func (cfg *apiConfig) handlerUploadPoster(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Movie
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

	movie, err := cfg.db.GetMovieByID(r.Context(), movieID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't find movie", err)
		return
	}

	assetURL := cfg.getAssetURL(assetPath)

	movie, err = cfg.db.UpdateMovie(r.Context(), database.UpdateMovieParams{
		ID:             movieID,
		Title:          movie.Title,
		Slug:           movie.Slug,
		Description:    movie.Description,
		RuntimeMinutes: movie.RuntimeMinutes,
		ReleaseDate:    movie.ReleaseDate,
		PosterUrl:      convertToNullString(&assetURL),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update movie", err)
		return
	}

	dbGenres, err := cfg.db.GetGenresByMovieID(r.Context(), movieID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't fetch movie genres", err)
		return
	}

	responseGenres := make([]Genre, len(dbGenres))
	for i, dbGenre := range dbGenres {
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
