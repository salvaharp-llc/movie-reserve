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

type Reservation struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	UserID      uuid.UUID `json:"user_id"`
	ScreeningID uuid.UUID `json:"screening_id"`
	SeatID      uuid.UUID `json:"seat_id"`
}

func (cfg *apiConfig) handlerCreateReservations(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		ScreeningID string `json:"screening_id"`
		SeatID      string `json:"seat_id"`
	}
	type response struct {
		Reservation
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	userID, err := GetUserID(r.Context())
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

	reservation, err := cfg.db.CreateReservation(r.Context(), database.CreateReservationParams{
		UserID:      userID,
		ScreeningID: screeningUUID,
		SeatID:      seatUUID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating reservation", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		Reservation: Reservation{
			ID:          reservation.ID,
			CreatedAt:   reservation.CreatedAt,
			UpdatedAt:   reservation.UpdatedAt,
			UserID:      reservation.UserID,
			ScreeningID: reservation.ScreeningID,
			SeatID:      reservation.SeatID,
		},
	})
}

func (cfg *apiConfig) handlerGetReservations(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Reservation
	}

	userID, err := GetUserID(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user id", err)
		return
	}
	userRole, err := GetUserRole(r.Context())
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

	reservation, err := cfg.db.GetReservationByID(r.Context(), reservationID)
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

	respondWithJSON(w, http.StatusOK, response{
		Reservation: Reservation{
			ID:          reservation.ID,
			CreatedAt:   reservation.CreatedAt,
			UpdatedAt:   reservation.UpdatedAt,
			UserID:      reservation.UserID,
			ScreeningID: reservation.ScreeningID,
			SeatID:      reservation.SeatID,
		},
	})
}

func (cfg *apiConfig) handlerRetrieveReservations(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Reservations []Reservation `json:"reservations"`
	}

	userID, err := GetUserID(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user id", err)
		return
	}

	reservations, err := cfg.db.GetReservationsByUserID(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get reservations", err)
		return
	}

	responseReservations := make([]Reservation, len(reservations))
	for i, r := range reservations {
		responseReservations[i] = Reservation{
			ID:          r.ID,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
			UserID:      r.UserID,
			ScreeningID: r.ScreeningID,
			SeatID:      r.SeatID,
		}
	}

	respondWithJSON(w, http.StatusOK, response{
		Reservations: responseReservations,
	})
}

func (cfg *apiConfig) handlerRetrieveReservationsAdmin(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Reservations []Reservation `json:"reservations"`
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

	reservations, err := cfg.db.GetReservations(r.Context(), database.GetReservationsParams{
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

	responseReservations := make([]Reservation, len(reservations))
	for i, res := range reservations {
		responseReservations[i] = Reservation{
			ID:          res.ID,
			CreatedAt:   res.CreatedAt,
			UpdatedAt:   res.UpdatedAt,
			UserID:      res.UserID,
			ScreeningID: res.ScreeningID,
			SeatID:      res.SeatID,
		}
	}

	respondWithJSON(w, http.StatusOK, response{Reservations: responseReservations})
}

func (cfg *apiConfig) handlerUpdateReservations(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		UserID      string `json:"user_id"`
		ScreeningID string `json:"screening_id"`
		SeatID      string `json:"seat_id"`
	}
	type response struct {
		Reservation
	}

	reservationIDString := r.PathValue("reservationID")
	reservationID, err := uuid.Parse(reservationIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	if strings.TrimSpace(params.UserID) == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required", nil)
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

	userUUID, err := uuid.Parse(params.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID", err)
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

	reservation, err := cfg.db.UpdateReservation(r.Context(), database.UpdateReservationParams{
		ID:          reservationID,
		UserID:      userUUID,
		ScreeningID: screeningUUID,
		SeatID:      seatUUID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update reservation", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Reservation: Reservation{
			ID:          reservation.ID,
			CreatedAt:   reservation.CreatedAt,
			UpdatedAt:   reservation.UpdatedAt,
			UserID:      reservation.UserID,
			ScreeningID: reservation.ScreeningID,
			SeatID:      reservation.SeatID,
		},
	})
}

func (cfg *apiConfig) handlerDeleteReservations(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserID(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user id", err)
		return
	}
	userRole, err := GetUserRole(r.Context())
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

	reservation, err := cfg.db.GetReservationByID(r.Context(), reservationID)
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

	err = cfg.db.DeleteReservation(r.Context(), reservationID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete reservation", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseNullUUID(s string) (uuid.NullUUID, error) {
	if s == "" {
		return uuid.NullUUID{}, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.NullUUID{}, err
	}
	return uuid.NullUUID{UUID: id, Valid: true}, nil
}
