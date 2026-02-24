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

	q := r.URL.Query()

	if q.Get("movie_id") == "" && (q.Get("from") == "" || q.Get("to") == "") {
		respondWithError(w, http.StatusBadRequest, "Provide either 'movie_id' or both 'from' and 'to' query params", nil)
		return
	}

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

	uuidParams := map[string]uuid.NullUUID{}
	for _, key := range []string{"movie_id"} {
		parsed, err := parseNullUUID(q.Get(key))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid "+key, err)
			return
		}
		uuidParams[key] = parsed
	}
	timeParams := map[string]sql.NullTime{}
	for _, key := range []string{"from", "to"} {
		parsed, err := parseNullTime(q.Get(key))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid "+key+" date", err)
			return
		}
		timeParams[key] = parsed
	}

	if !timeParams["from"].Valid || timeParams["from"].Time.Before(time.Now()) {
		timeParams["from"] = sql.NullTime{Time: time.Now(), Valid: true}
	}

	screenings, err := cfg.db.GetScreenings(r.Context(), database.GetScreeningsParams{
		MovieID:  uuidParams["movie_id"],
		From:     timeParams["from"],
		To:       timeParams["to"],
		Upcoming: true,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})

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

	uuidParams := map[string]uuid.NullUUID{}
	for _, key := range []string{"movie_id", "room_id"} {
		parsed, err := parseNullUUID(q.Get(key))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid "+key, err)
			return
		}
		uuidParams[key] = parsed
	}

	timeParams := map[string]sql.NullTime{}
	for _, key := range []string{"from", "to"} {
		parsed, err := parseNullTime(q.Get(key))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid "+key+" date", err)
			return
		}
		timeParams[key] = parsed
	}

	screenings, err := cfg.db.GetScreenings(r.Context(), database.GetScreeningsParams{
		MovieID:  uuidParams["movie_id"],
		RoomID:   uuidParams["room_id"],
		From:     timeParams["from"],
		To:       timeParams["to"],
		Upcoming: false,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})

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
