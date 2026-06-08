package main

import "net/http"

func respondWithText(w http.ResponseWriter, code int, text string) {
	w.Header().Add("Content-Type", " text/plain; charset=utf-8")
	w.WriteHeader(code)
	w.Write([]byte(text))
}
