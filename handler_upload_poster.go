package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

func (cfg *apiConfig) handlerUploadPoster(w http.ResponseWriter, r *http.Request) {
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

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't rewind file", err)
		return
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("image_file", "image"+mediaTypeToExt(mediaType))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create form writer", err)
		return
	}

	if _, err := io.Copy(part, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create form writer", err)
		return
	}

	if err := writer.Close(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't close form writer", err)
		return
	}

	assetURL := cfg.getAssetURL(assetPath)

	err = cfg.db.ExecTx(r.Context(), func(q *database.Queries) error {
		err = q.UploadMoviePoster(r.Context(), database.UploadMoviePosterParams{
			ID:        movieID,
			PosterUrl: sql.NullString{String: assetURL, Valid: true},
		})
		if err != nil {
			return err
		}

		request, err := http.NewRequestWithContext(
			r.Context(),
			"PUT",
			cfg.ragServerURL+"/images/"+movieIDString,
			&buf,
		)
		if err != nil {
			return err
		}

		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.Header.Set("Authorization", "Bearer "+cfg.ragAPIKey)

		client := &http.Client{}
		resp, err := client.Do(request)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("Could not add image to RAG service: %s", resp.Status)
		}

		return nil
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update movie poster or notify RAG service", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
