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

type Screening struct {
	ID        uuid.UUID `json:"id"`
	MovieID   uuid.UUID `json:"movie_id"`
	RoomID    uuid.UUID `json:"room_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (cfg *apiConfig) handlerCreateScreenings(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		MovieIDString string    `json:"movie_id"`
		RoomIDString  string    `json:"room_id"`
		StartTime     time.Time `json:"start_time"`
		EndTime       time.Time `json:"end_time"`
	}
	type response struct {
		Screening
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	MovieID, err := uuid.Parse(params.MovieIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Movie ID", err)
		return
	}

	RoomID, err := uuid.Parse(params.RoomIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Room ID", err)
		return
	}

	screening, err := cfg.db.CreateScreening(r.Context(), database.CreateScreeningParams{
		MovieID:   MovieID,
		RoomID:    RoomID,
		StartTime: params.StartTime,
		EndTime:   params.EndTime,
	})
	if err != nil {
		if strings.Contains(err.Error(), "valid_time_range") {
			respondWithError(w, http.StatusBadRequest, "Screening end time must be after start time", err)
			return
		}
		if strings.Contains(err.Error(), "no_overlapping_screenings") {
			respondWithError(w, http.StatusBadRequest, "Screening times overlap with existing screening in the same room", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error creating screening", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		Screening: Screening{
			ID:        screening.ID,
			MovieID:   screening.MovieID,
			RoomID:    screening.RoomID,
			StartTime: screening.StartTime,
			EndTime:   screening.EndTime,
			CreatedAt: screening.CreatedAt,
			UpdatedAt: screening.UpdatedAt,
		},
	})
}

func (cfg *apiConfig) handlerGetScreenings(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Screening
	}

	screeningIDString := r.PathValue("screeningID")
	screeningID, err := uuid.Parse(screeningIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	screening, err := cfg.db.GetScreeningByID(r.Context(), screeningID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Screening not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't get screening", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Screening: Screening{
			ID:        screening.ID,
			MovieID:   screening.MovieID,
			RoomID:    screening.RoomID,
			StartTime: screening.StartTime,
			EndTime:   screening.EndTime,
			CreatedAt: screening.CreatedAt,
			UpdatedAt: screening.UpdatedAt,
		},
	})
}

func (cfg *apiConfig) handlerRetrieveScreenings(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Screenings []Screening `json:"screenings"`
	}

	movieIDStr := r.URL.Query().Get("movie_id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var movieID uuid.UUID
	var from, to time.Time
	var err error

	hasMovieID := movieIDStr != ""
	hasDateRange := fromStr != "" && toStr != ""

	if hasMovieID {
		movieID, err = uuid.Parse(movieIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid movie_id", err)
			return
		}
	}

	if hasDateRange {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid 'from' date, expected RFC3339 format", err)
			return
		}
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid 'to' date, expected RFC3339 format", err)
			return
		}
	}

	var screenings []database.Screening

	switch {
	case hasMovieID && hasDateRange:
		screenings, err = cfg.db.GetUpcomingScreeningsByMovieIDAndDateRange(r.Context(), database.GetUpcomingScreeningsByMovieIDAndDateRangeParams{
			MovieID:     movieID,
			StartTime:   from,
			StartTime_2: to,
		})

	case hasMovieID:
		screenings, err = cfg.db.GetUpcomingScreeningsByMovieID(r.Context(), movieID)

	case hasDateRange:
		screenings, err = cfg.db.GetUpcomingScreeningsByDateRange(r.Context(), database.GetUpcomingScreeningsByDateRangeParams{
			StartTime:   from,
			StartTime_2: to,
		})

	default:
		respondWithError(w, http.StatusBadRequest, "Provide either 'movie_id' or 'from' and 'to' query params", nil)
		return
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get screenings", err)
		return
	}

	if len(screenings) == 0 {
		respondWithError(w, http.StatusNotFound, "No upcoming screenings found", nil)
		return
	}

	responseScreenings := make([]Screening, len(screenings))
	for i, s := range screenings {
		responseScreenings[i] = Screening{
			ID:        s.ID,
			MovieID:   s.MovieID,
			RoomID:    s.RoomID,
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		}
	}

	respondWithJSON(w, http.StatusOK, response{Screenings: responseScreenings})
}

func (cfg *apiConfig) handlerRetrieveScreeningsAdmin(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Screenings []Screening `json:"screenings"`
	}

	movieIDStr := r.URL.Query().Get("movie_id")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100 // sensible default
	offset := 0

	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 1 {
			respondWithError(w, http.StatusBadRequest, "Invalid limit", err)
			return
		}
		limit = parsedLimit
	}

	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil || parsedOffset < 0 {
			respondWithError(w, http.StatusBadRequest, "Invalid offset", err)
			return
		}
		offset = parsedOffset
	}

	var movieID uuid.UUID
	var from, to time.Time
	var err error

	hasMovieID := movieIDStr != ""
	hasDateRange := fromStr != "" && toStr != ""

	if hasMovieID {
		movieID, err = uuid.Parse(movieIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid movie_id", err)
			return
		}
	}

	if hasDateRange {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid 'from' date", err)
			return
		}
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid 'to' date", err)
			return
		}
	}

	var screenings []database.Screening

	switch {
	case hasMovieID && hasDateRange:
		screenings, err = cfg.db.GetScreeningsByMovieIDAndDateRange(r.Context(), database.GetScreeningsByMovieIDAndDateRangeParams{
			MovieID:     movieID,
			StartTime:   from,
			StartTime_2: to,
		})

	case hasMovieID:
		screenings, err = cfg.db.GetScreeningsByMovieID(r.Context(), movieID)

	case hasDateRange:
		screenings, err = cfg.db.GetScreeningsByDateRange(r.Context(), database.GetScreeningsByDateRangeParams{
			StartTime:   from,
			StartTime_2: to,
		})

	default:
		screenings, err = cfg.db.GetScreeningsPaginated(r.Context(), database.GetScreeningsPaginatedParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get screenings", err)
		return
	}

	responseScreenings := make([]Screening, len(screenings))
	for i, s := range screenings {
		responseScreenings[i] = Screening{
			ID:        s.ID,
			MovieID:   s.MovieID,
			RoomID:    s.RoomID,
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		}
	}

	respondWithJSON(w, http.StatusOK, response{Screenings: responseScreenings})
}

func (cfg *apiConfig) handlerUpdateScreenings(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		MovieIDString string    `json:"movie_id"`
		RoomIDString  string    `json:"room_id"`
		StartTime     time.Time `json:"start_time"`
		EndTime       time.Time `json:"end_time"`
	}
	type response struct {
		Screening
	}

	screeningIDString := r.PathValue("screeningID")
	screeningID, err := uuid.Parse(screeningIDString)
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

	movieID, err := uuid.Parse(params.MovieIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid movie ID", err)
		return
	}

	roomID, err := uuid.Parse(params.RoomIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid room ID", err)
		return
	}

	screening, err := cfg.db.UpdateScreening(r.Context(), database.UpdateScreeningParams{
		ID:        screeningID,
		MovieID:   movieID,
		RoomID:    roomID,
		StartTime: params.StartTime,
		EndTime:   params.EndTime,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Screening not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't update screening", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Screening: Screening{
			ID:        screening.ID,
			MovieID:   screening.MovieID,
			RoomID:    screening.RoomID,
			StartTime: screening.StartTime,
			EndTime:   screening.EndTime,
			CreatedAt: screening.CreatedAt,
			UpdatedAt: screening.UpdatedAt,
		},
	})
}

func (cfg *apiConfig) handlerDeleteScreenings(w http.ResponseWriter, r *http.Request) {
	screeningIDString := r.PathValue("screeningID")
	screeningID, err := uuid.Parse(screeningIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	err = cfg.db.DeleteScreening(r.Context(), screeningID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Screening not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete screening", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
