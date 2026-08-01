package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type RagRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k,omitempty"`
}

type RagResponse struct {
	Answer  string           `json:"answer"`
	Sources []map[string]any `json:"sources"`
}

func (cfg *apiConfig) handlerRAG(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		RagRequest
	}
	type response struct {
		RagResponse
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}
	if strings.TrimSpace(params.Query) == "" {
		respondWithError(w, http.StatusBadRequest, "Query is required", nil)
		return
	}

	body, err := json.Marshal(RagRequest{
		Query: params.Query,
		TopK:  params.TopK,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error marshaling request", err)
		return
	}

	request, err := http.NewRequest("POST", cfg.ragServerURL+"/query", bytes.NewBuffer(body))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating request to RAG service", err)
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+cfg.ragAPIKey)

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error making request to RAG service", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respondWithError(w, http.StatusInternalServerError, "RAG service returned non-OK status: "+resp.Status, nil)
		return
	}

	decoder = json.NewDecoder(resp.Body)
	ragResp := RagResponse{}
	err = decoder.Decode(&ragResp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding RAG response", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{ragResp})
}
