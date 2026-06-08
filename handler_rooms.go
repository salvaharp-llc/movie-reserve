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

type RoomSummary struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type RoomDetail struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Seats     []SeatSummary `json:"seats"`
}

func (cfg *apiConfig) handlerCreateRooms(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
	}
	type response struct {
		RoomDetail `json:"room"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	if strings.TrimSpace(params.Name) == "" {
		respondWithError(w, http.StatusBadRequest, "Room name is required", nil)
		return
	}

	var roomDetail database.GetRoomDetailByIDRow
	err = cfg.db.ExecTx(r.Context(), func(q *database.Queries) error {
		room, err := q.CreateRoom(r.Context(), params.Name)
		if err != nil {
			return err
		}

		roomDetail, err = q.GetRoomDetailByID(r.Context(), room.ID)
		return err
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create room", err)
		return
	}

	responseRoom := aggregateRoomDetail(roomDetail)

	respondWithJSON(w, http.StatusOK, response{responseRoom})
}

func (cfg *apiConfig) handlerGetRooms(w http.ResponseWriter, r *http.Request) {
	type response struct {
		RoomDetail `json:"room"`
	}

	roomIDString := r.PathValue("roomID")
	roomID, err := uuid.Parse(roomIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	room, err := cfg.db.GetRoomDetailByID(r.Context(), roomID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Room not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't get room", err)
		return
	}

	responseRoom := aggregateRoomDetail(room)

	respondWithJSON(w, http.StatusOK, response{responseRoom})
}

func (cfg *apiConfig) handlerRetrieveRooms(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Rooms []RoomSummary `json:"rooms"`
	}

	rooms, err := cfg.db.GetRoomsSummary(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get rooms", err)
		return
	}

	if len(rooms) == 0 {
		respondWithError(w, http.StatusNotFound, "Rooms not found", nil)
		return
	}

	responseRooms := make([]RoomSummary, len(rooms))
	for i, room := range rooms {
		responseRooms[i] = RoomSummary{
			ID:   room.ID,
			Name: room.Name,
		}
	}

	respondWithJSON(w, http.StatusOK, response{
		Rooms: responseRooms,
	})
}

func (cfg *apiConfig) handlerUpdateRooms(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
	}
	type response struct {
		RoomDetail `json:"room"`
	}

	roomIDString := r.PathValue("roomID")
	roomID, err := uuid.Parse(roomIDString)
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

	if strings.TrimSpace(params.Name) == "" {
		respondWithError(w, http.StatusBadRequest, "Room name is required", nil)
		return
	}

	var roomDetail database.GetRoomDetailByIDRow
	err = cfg.db.ExecTx(r.Context(), func(q *database.Queries) error {
		_, err = q.UpdateRoom(r.Context(), database.UpdateRoomParams{
			ID:   roomID,
			Name: params.Name,
		})
		if err != nil {
			return err
		}

		roomDetail, err = q.GetRoomDetailByID(r.Context(), roomID)
		return err
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Room not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't update room", err)
		return
	}

	responseRoom := aggregateRoomDetail(roomDetail)

	respondWithJSON(w, http.StatusOK, response{responseRoom})
}

func (cfg *apiConfig) handlerDeleteRooms(w http.ResponseWriter, r *http.Request) {
	roomIDString := r.PathValue("roomID")
	roomID, err := uuid.Parse(roomIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	err = cfg.db.DeleteRoom(r.Context(), roomID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Room not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete room", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func aggregateRoomDetail(r database.GetRoomDetailByIDRow) RoomDetail {
	var seats []SeatSummary
	if r.Seats != nil {
		// sqlc returns aggregated JSON columns as interface{} which may be
		// []byte or string depending on the driver; we need this type switch
		// to unmarshal it correctly.
		switch v := r.Seats.(type) {
		case []byte:
			_ = json.Unmarshal(v, &seats)
		case string:
			_ = json.Unmarshal([]byte(v), &seats)
		default:
			// fallback if sqlc already gave us a slice (unlikely)
			if b, err := json.Marshal(v); err == nil {
				_ = json.Unmarshal(b, &seats)
			}
		}
	}

	return RoomDetail{
		ID:        r.ID,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Name:      r.Name,
		Seats:     seats,
	}
}
