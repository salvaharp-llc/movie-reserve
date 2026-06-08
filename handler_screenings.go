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

type ScreeningSeat struct {
	SeatSummary
	Available bool `json:"available"`
}

type ScreeningSummary struct {
	ID        uuid.UUID    `json:"id"`
	Movie     MovieSummary `json:"movie"`
	StartTime time.Time    `json:"start_time"`
	EndTime   time.Time    `json:"end_time"`
}

type ScreeningDetail struct {
	ID        uuid.UUID       `json:"id"`
	Movie     MovieSummary    `json:"movie"`
	Room      RoomSummary     `json:"room"`
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Seats     []ScreeningSeat `json:"seats"`
}

func (cfg *apiConfig) handlerCreateScreenings(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		MovieIDString string    `json:"movie_id"`
		RoomIDString  string    `json:"room_id"`
		StartTime     time.Time `json:"start_time"`
		EndTime       time.Time `json:"end_time"`
	}
	type response struct {
		ScreeningDetail `json:"screening"`
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

	var screeningDetail database.GetScreeningDetailByIDRow
	err = cfg.db.ExecTx(r.Context(), func(q *database.Queries) error {
		screening, err := q.CreateScreening(r.Context(), database.CreateScreeningParams{
			MovieID:   MovieID,
			RoomID:    RoomID,
			StartTime: params.StartTime,
			EndTime:   params.EndTime,
		})
		if err != nil {
			return err
		}

		screeningDetail, err = q.GetScreeningDetailByID(r.Context(), screening.ID)
		return err
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
	}

	responseScreening := aggregateScreeningDetail(screeningDetail)

	respondWithJSON(w, http.StatusOK, response{responseScreening})
}

func (cfg *apiConfig) handlerGetScreenings(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ScreeningDetail `json:"screening"`
	}

	screeningIDString := r.PathValue("screeningID")
	screeningID, err := uuid.Parse(screeningIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	screening, err := cfg.db.GetScreeningDetailByID(r.Context(), screeningID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Screening not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't get screening", err)
		return
	}

	responseScreening := aggregateScreeningDetail(screening)

	respondWithJSON(w, http.StatusOK, response{responseScreening})
}

func (cfg *apiConfig) handlerRetrieveScreenings(w http.ResponseWriter, r *http.Request) {
	cfg.retrieveScreenings(w, r, true)
}

func (cfg *apiConfig) handlerRetrieveScreeningsAdmin(w http.ResponseWriter, r *http.Request) {
	cfg.retrieveScreenings(w, r, false)
}

func (cfg *apiConfig) retrieveScreenings(w http.ResponseWriter, r *http.Request, filter_upcoming bool) {
	type response struct {
		Screenings []ScreeningSummary `json:"screenings"`
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

	movieID, err := parseNullUUID(q.Get("movie_id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid movie_id", err)
		return
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

	if filter_upcoming && timeParams["from"].Valid && timeParams["from"].Time.Before(time.Now()) {
		respondWithError(w, http.StatusBadRequest, "'from' date cannot be in the past", nil)
		return
	}

	screenings, err := cfg.db.GetScreeningsSummary(r.Context(), database.GetScreeningsSummaryParams{
		MovieID: movieID,
		From:    timeParams["from"],
		To:      timeParams["to"],
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get screenings", err)
		return
	}

	responseScreenings := make([]ScreeningSummary, len(screenings))
	for i, s := range screenings {
		responseScreenings[i] = ScreeningSummary{
			ID: s.ID,
			Movie: MovieSummary{
				ID:        s.MovieID,
				Title:     s.MovieTitle,
				Slug:      s.MovieSlug,
				PosterUrl: nullStringToPtr(s.MoviePosterUrl),
			},
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
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
		ScreeningDetail `json:"screening"`
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

	var screeningDetail database.GetScreeningDetailByIDRow
	err = cfg.db.ExecTx(r.Context(), func(q *database.Queries) error {
		screening, err := q.UpdateScreening(r.Context(), database.UpdateScreeningParams{
			ID:        screeningID,
			MovieID:   movieID,
			RoomID:    roomID,
			StartTime: params.StartTime,
			EndTime:   params.EndTime,
		})
		if err != nil {
			return err
		}

		screeningDetail, err = q.GetScreeningDetailByID(r.Context(), screening.ID)
		return err
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Screening not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't update screening", err)
		return
	}

	responseScreening := aggregateScreeningDetail(screeningDetail)

	respondWithJSON(w, http.StatusOK, response{responseScreening})
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

func aggregateScreeningDetail(sc database.GetScreeningDetailByIDRow) ScreeningDetail {
	var seats []ScreeningSeat
	_ = json.Unmarshal(sc.Seats, &seats)

	return ScreeningDetail{
		ID: sc.ID,
		Movie: MovieSummary{
			ID:        sc.MovieID,
			Title:     sc.MovieTitle,
			Slug:      sc.MovieSlug,
			PosterUrl: nullStringToPtr(sc.MoviePosterUrl),
		},
		Room: RoomSummary{
			ID:   sc.RoomID,
			Name: sc.RoomName,
		},
		StartTime: sc.StartTime,
		EndTime:   sc.EndTime,
		CreatedAt: sc.CreatedAt,
		UpdatedAt: sc.UpdatedAt,
		Seats:     seats,
	}
}
