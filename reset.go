package main

import "net/http"

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Reset is only allowed in dev environment.", nil)
		return
	}

	if err := cfg.db.Reset(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reseting DB", err)
		return
	}

	if err := cfg.ensureAdmin(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not ensure admin user", err)
		return
	}

	w.Header().Add("Content-Type", " text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Database reset to initial state."))
}
