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

type Room struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
}

func (cfg *apiConfig) handlerCreateRooms(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
	}
	type response struct {
		Room
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

	room, err := cfg.db.CreateRoom(r.Context(), params.Name)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating room", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		Room: Room{
			ID:        room.ID,
			CreatedAt: room.CreatedAt,
			UpdatedAt: room.UpdatedAt,
			Name:      room.Name,
		},
	})
}

func (cfg *apiConfig) handlerGetRooms(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Room
	}

	roomIDString := r.PathValue("roomID")
	roomID, err := uuid.Parse(roomIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	room, err := cfg.db.GetRoomByID(r.Context(), roomID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Room not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't get room", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Room: Room{
			ID:        room.ID,
			CreatedAt: room.CreatedAt,
			UpdatedAt: room.UpdatedAt,
			Name:      room.Name,
		},
	})
}

func (cfg *apiConfig) handlerRetrieveRooms(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Rooms []Room `json:"rooms"`
	}

	rooms, err := cfg.db.GetRooms(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get rooms", err)
		return
	}

	if len(rooms) == 0 {
		respondWithError(w, http.StatusNotFound, "Rooms not found", nil)
		return
	}

	responseRooms := make([]Room, len(rooms))
	for i, room := range rooms {
		responseRooms[i] = Room{
			ID:        room.ID,
			CreatedAt: room.CreatedAt,
			UpdatedAt: room.UpdatedAt,
			Name:      room.Name,
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
		Room
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

	room, err := cfg.db.UpdateRoom(r.Context(), database.UpdateRoomParams{
		ID:   roomID,
		Name: params.Name,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, "Room not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't update room", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Room: Room{
			ID:        room.ID,
			CreatedAt: room.CreatedAt,
			UpdatedAt: room.UpdatedAt,
			Name:      room.Name,
		},
	})
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
