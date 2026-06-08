package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/salvaharp-llc/movie-reserve/internal/database"
)

type SeatSummary struct {
	ID         uuid.UUID `json:"id"`
	RowLabel   string    `json:"row_label"`
	SeatNumber int32     `json:"seat_number"`
}

type SeatDetail struct {
	ID         uuid.UUID   `json:"id"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	Room       RoomSummary `json:"room"`
	RowLabel   string      `json:"row_label"`
	SeatNumber int32       `json:"seat_number"`
}

func (cfg *apiConfig) handlerCreateSeats(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		RoomID     string `json:"room_id"`
		RowLabel   string `json:"row_label"`
		SeatNumber int32  `json:"seat_number"`
	}
	type response struct {
		SeatDetail `json:"seat"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	if strings.TrimSpace(params.RoomID) == "" {
		respondWithError(w, http.StatusBadRequest, "Room ID is required", nil)
		return
	}
	if strings.TrimSpace(params.RowLabel) == "" {
		respondWithError(w, http.StatusBadRequest, "Row label is required", nil)
		return
	}
	if params.SeatNumber <= 0 {
		respondWithError(w, http.StatusBadRequest, "Seat number must be greater than zero", nil)
		return
	}

	roomUUID, err := uuid.Parse(params.RoomID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid room ID", err)
		return
	}

	var seatDetail database.GetSeatDetailByIDRow
	err = cfg.db.ExecTx(r.Context(), func(q *database.Queries) error {
		seat, err := q.CreateSeat(r.Context(), database.CreateSeatParams{
			RoomID:     roomUUID,
			RowLabel:   params.RowLabel,
			SeatNumber: params.SeatNumber,
		})
		if err != nil {
			return err
		}

		seatDetail, err = q.GetSeatDetailByID(r.Context(), seat.ID)
		return err
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating seat", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		SeatDetail{
			ID:        seatDetail.ID,
			CreatedAt: seatDetail.CreatedAt,
			UpdatedAt: seatDetail.UpdatedAt,
			Room: RoomSummary{
				ID:   seatDetail.RoomID,
				Name: seatDetail.RoomName,
			},
			RowLabel:   seatDetail.RowLabel,
			SeatNumber: seatDetail.SeatNumber,
		},
	})
}

func (cfg *apiConfig) handlerGetSeats(w http.ResponseWriter, r *http.Request) {
	type response struct {
		SeatDetail `json:"seat"`
	}

	seatIDString := r.PathValue("seatID")
	seatID, err := uuid.Parse(seatIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	seat, err := cfg.db.GetSeatDetailByID(r.Context(), seatID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Seat not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't get seat", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		SeatDetail{
			ID:        seat.ID,
			CreatedAt: seat.CreatedAt,
			UpdatedAt: seat.UpdatedAt,
			Room: RoomSummary{
				ID:   seat.RoomID,
				Name: seat.RoomName,
			},
			RowLabel:   seat.RowLabel,
			SeatNumber: seat.SeatNumber,
		},
	})
}

func (cfg *apiConfig) handlerUpdateSeats(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		RoomID     string `json:"room_id"`
		RowLabel   string `json:"row_label"`
		SeatNumber int32  `json:"seat_number"`
	}
	type response struct {
		SeatDetail `json:"seat"`
	}

	seatIDString := r.PathValue("seatID")
	seatID, err := uuid.Parse(seatIDString)
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

	if strings.TrimSpace(params.RoomID) == "" {
		respondWithError(w, http.StatusBadRequest, "Room ID is required", nil)
		return
	}
	if strings.TrimSpace(params.RowLabel) == "" {
		respondWithError(w, http.StatusBadRequest, "Row label is required", nil)
		return
	}
	if params.SeatNumber <= 0 {
		respondWithError(w, http.StatusBadRequest, "Seat number must be greater than zero", nil)
		return
	}

	roomUUID, err := uuid.Parse(params.RoomID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid room ID", err)
		return
	}

	var seatDetail database.GetSeatDetailByIDRow
	err = cfg.db.ExecTx(r.Context(), func(q *database.Queries) error {
		_, err := q.UpdateSeat(r.Context(), database.UpdateSeatParams{
			ID:         seatID,
			RoomID:     roomUUID,
			RowLabel:   params.RowLabel,
			SeatNumber: params.SeatNumber,
		})
		if err != nil {
			return err
		}

		seatDetail, err = q.GetSeatDetailByID(r.Context(), seatID)
		return err
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Seat not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't update seat", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		SeatDetail{
			ID:        seatDetail.ID,
			CreatedAt: seatDetail.CreatedAt,
			UpdatedAt: seatDetail.UpdatedAt,
			Room: RoomSummary{
				ID:   seatDetail.RoomID,
				Name: seatDetail.RoomName,
			},
			RowLabel:   seatDetail.RowLabel,
			SeatNumber: seatDetail.SeatNumber,
		},
	})
}

func (cfg *apiConfig) handlerDeleteSeats(w http.ResponseWriter, r *http.Request) {
	seatIDString := r.PathValue("seatID")
	seatID, err := uuid.Parse(seatIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	err = cfg.db.DeleteSeat(r.Context(), seatID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete seat", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
