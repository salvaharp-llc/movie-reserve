package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/auth"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

type ReservationSummary struct {
	ID        uuid.UUID        `json:"id"`
	Screening ScreeningSummary `json:"screening"`
	Seat      SeatSummary      `json:"seat"`
}

type ReservationDetail struct {
	ID        uuid.UUID        `json:"id"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	User      UserSummary      `json:"user"`
	Screening ScreeningSummary `json:"screening"`
	Room      RoomSummary      `json:"room"`
	Seat      SeatSummary      `json:"seat"`
}

func (cfg *apiConfig) handlerCreateReservations(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		ScreeningID string `json:"screening_id"`
		SeatID      string `json:"seat_id"`
	}
	type response struct {
		ReservationDetail `json:"reservation"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	userID, err := getUserID(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user id", err)
		return
	}
	if strings.TrimSpace(params.ScreeningID) == "" {
		respondWithError(w, http.StatusBadRequest, "Screening ID is required", nil)
		return
	}
	if strings.TrimSpace(params.SeatID) == "" {
		respondWithError(w, http.StatusBadRequest, "Seat ID is required", nil)
		return
	}

	screeningUUID, err := uuid.Parse(params.ScreeningID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid screening ID", err)
		return
	}

	seatUUID, err := uuid.Parse(params.SeatID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid seat ID", err)
		return
	}

	var reservationDetails database.GetReservationDetailByIDRow
	err = cfg.db.ExecTx(r.Context(), func(q *database.Queries) error {
		reservation, err := q.CreateReservation(r.Context(), database.CreateReservationParams{
			UserID:      userID,
			ScreeningID: screeningUUID,
			SeatID:      seatUUID,
		})
		if err != nil {
			return err
		}

		reservationDetails, err = q.GetReservationDetailByID(r.Context(), reservation.ID)
		return err
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create reservation", err)
		return
	}

	responseReservation := aggregateReservationDetails(reservationDetails)

	respondWithJSON(w, http.StatusCreated, response{responseReservation})
}

func (cfg *apiConfig) handlerGetReservations(w http.ResponseWriter, r *http.Request) {
	type response struct {
		ReservationDetail `json:"reservation"`
	}

	userID, err := getUserID(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user id", err)
		return
	}
	userRole, err := getUserRole(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user role", err)
		return
	}

	reservationIDString := r.PathValue("reservationID")
	reservationID, err := uuid.Parse(reservationIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	reservation, err := cfg.db.GetReservationDetailByID(r.Context(), reservationID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Reservation not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't get reservation", err)
		return
	}

	if reservation.UserID != userID && userRole != auth.RoleAdmin {
		respondWithError(w, http.StatusForbidden, "You don't have permission to access this reservation", nil)
		return
	}

	responseReservation := aggregateReservationDetails(reservation)

	respondWithJSON(w, http.StatusOK, response{responseReservation})
}

func (cfg *apiConfig) handlerRetrieveReservations(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Reservations []ReservationSummary `json:"reservations"`
	}

	userID, err := getUserID(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user id", err)
		return
	}

	reservations, err := cfg.db.GetReservationsSummary(r.Context(), database.GetReservationsSummaryParams{
		UserID: uuid.NullUUID{UUID: userID, Valid: true},
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get reservations", err)
		return
	}

	responseReservations := aggregateReservationSummaries(reservations)

	respondWithJSON(w, http.StatusOK, response{
		Reservations: responseReservations,
	})
}

func (cfg *apiConfig) handlerRetrieveReservationsAdmin(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Reservations []ReservationSummary `json:"reservations"`
	}

	q := r.URL.Query()

	if q.Get("screening_id") != "" && (q.Get("room_id") != "" || q.Get("movie_id") != "") {
		respondWithError(w, http.StatusBadRequest, "Cannot specify both screening_id and room_id or movie_id", nil)
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
	for _, key := range []string{"user_id", "screening_id", "movie_id", "room_id"} {
		parsed, err := parseNullUUID(q.Get(key))
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid "+key, err)
			return
		}
		uuidParams[key] = parsed
	}

	reservations, err := cfg.db.GetReservationsSummary(r.Context(), database.GetReservationsSummaryParams{
		UserID:      uuidParams["user_id"],
		ScreeningID: uuidParams["screening_id"],
		MovieID:     uuidParams["movie_id"],
		RoomID:      uuidParams["room_id"],
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get reservations", err)
		return
	}

	responseReservations := aggregateReservationSummaries(reservations)

	respondWithJSON(w, http.StatusOK, response{Reservations: responseReservations})
}

func (cfg *apiConfig) handlerDeleteReservations(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user id", err)
		return
	}
	userRole, err := getUserRole(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user role", err)
		return
	}

	reservationIDString := r.PathValue("reservationID")
	reservationID, err := uuid.Parse(reservationIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	reservation, err := cfg.db.GetReservationMetaByID(r.Context(), reservationID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Reservation not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't get reservation", err)
		return
	}

	if reservation.UserID != userID && userRole != auth.RoleAdmin {
		respondWithError(w, http.StatusForbidden, "You don't have permission to delete this reservation", nil)
		return
	}

	if reservation.ScreeningStartTime.Before(time.Now()) {
		respondWithError(w, http.StatusBadRequest, "You can't delete this reservation after screening started", nil)
		return
	}

	err = cfg.db.DeleteReservation(r.Context(), reservationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete reservation", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func aggregateReservationDetails(reservation database.GetReservationDetailByIDRow) ReservationDetail {
	return ReservationDetail{
		ID:        reservation.ID,
		CreatedAt: reservation.CreatedAt,
		UpdatedAt: reservation.UpdatedAt,
		User: UserSummary{
			ID:    reservation.UserID,
			Email: reservation.UserEmail,
		},
		Screening: ScreeningSummary{
			ID:        reservation.ScreeningID,
			StartTime: reservation.ScreeningStartTime,
			EndTime:   reservation.ScreeningEndTime,
			Movie: MovieSummary{
				ID:        reservation.MovieID,
				Title:     reservation.MovieTitle,
				Slug:      reservation.MovieSlug,
				PosterUrl: nullStringToPtr(reservation.MoviePosterUrl),
			},
		},
		Room: RoomSummary{
			ID:   reservation.RoomID,
			Name: reservation.RoomName,
		},
		Seat: SeatSummary{
			ID:         reservation.SeatID,
			RowLabel:   reservation.SeatRowLabel,
			SeatNumber: reservation.SeatNumber,
		},
	}
}

func aggregateReservationSummaries(reservations []database.GetReservationsSummaryRow) []ReservationSummary {
	summaries := make([]ReservationSummary, len(reservations))
	for i, reservation := range reservations {
		summaries[i] = ReservationSummary{
			ID: reservation.ID,
			Screening: ScreeningSummary{
				ID:        reservation.ScreeningID,
				StartTime: reservation.ScreeningStartTime,
				EndTime:   reservation.ScreeningEndTime,
				Movie: MovieSummary{
					ID:        reservation.MovieID,
					Title:     reservation.MovieTitle,
					Slug:      reservation.MovieSlug,
					PosterUrl: nullStringToPtr(reservation.MoviePosterUrl),
				},
			},
			Seat: SeatSummary{
				ID:         reservation.SeatID,
				RowLabel:   reservation.SeatRowLabel,
				SeatNumber: reservation.SeatNumber,
			},
		}
	}
	return summaries
}
