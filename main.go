package main

import (
    "encoding/json"
    "encoding/xml"
    "log"
    "net/http"
)

type TransmuteRequest struct {
    Phrase string `json:"phrase"`
}

type ResonanceResponse struct {
    XMLName   xml.Name `xml:"resonance"`
    Response  string   `xml:"response"`
    Frequency string   `xml:"frequency"`
}

func transmuteHandler(w http.ResponseWriter, r *http.Request) {
    var req TransmuteRequest
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    response := ResonanceResponse{
        Response:  "Scroll received: " + req.Phrase,
        Frequency: "432Hz",
    }

    w.Header().Set("Content-Type", "application/xml")
    xml.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/transmute", transmuteHandler)
    log.Println("Listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}