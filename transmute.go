package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type TransmuteRequest struct {
	EDI string `json:"edi"`
}

type TransmuteResponse struct {
	CSV   string `json:"csv"`
	JSON  string `json:"json"`
	Log   string `json:"log"`
	Error string `json:"error,omitempty"`
}

func TransmuteHandler(w http.ResponseWriter, r *http.Request) {
	var req TransmuteRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Println("Received EDI content for transmutation...")

	csv, jsonOut, logTrace := Transmute(req.EDI)

	resp := TransmuteResponse{
		CSV:  csv,
		JSON: jsonOut,
		Log:  logTrace,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}