package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type TransmuteRequest struct {
	Phrase     string `json:"phrase"`
	SourceData string `json:"sourceData"`
	Key5D      string `json:"key5D"`
}

type TransmuteResponse struct {
	Output    string `json:"output"`
	Frequency string `json:"frequency"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

type OpenAIChoice struct {
	Message OpenAIMessage `json:"message"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
}

type PhraseEntry struct {
	Phrase       string `json:"phrase"`
	Frequency    string `json:"frequency"`
	SourceFormat string `json:"sourceFormat,omitempty"`
	TargetFormat string `json:"targetFormat,omitempty"`
	SourceSample string `json:"sourceSample,omitempty"`
	TargetSample string `json:"targetSample,omitempty"`
}

var (
	phraseStore = make([]PhraseEntry, 0)
	storeMutex  sync.Mutex
)

func callGPT4(prompt string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("missing OPENAI_API_KEY")
	}

	reqBody := OpenAIRequest{
		Model: "gpt-4",
		Messages: []OpenAIMessage{
			{Role: "system", Content: "You are a frequency-based transmutation assistant that uses 5D resonance keys to decode and transform text."},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error: %s", body)
	}

	var result OpenAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from OpenAI")
}

func TransmuteHandler(w http.ResponseWriter, r *http.Request) {
	var req TransmuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	prompt := fmt.Sprintf("Phrase: %s\nSourceData: %s\nKey5D: %s", req.Phrase, req.SourceData, req.Key5D)
	output, err := callGPT4(prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := TransmuteResponse{
		Output:    output,
		Frequency: "432Hz",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func AddPhraseHandler(w http.ResponseWriter, r *http.Request) {
	var entry PhraseEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	storeMutex.Lock()
	phraseStore = append(phraseStore, entry)
	storeMutex.Unlock()
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func ListPhrasesHandler(w http.ResponseWriter, r *http.Request) {
	storeMutex.Lock()
	defer storeMutex.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(phraseStore)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))

	http.HandleFunc("/transmute", TransmuteHandler)
	http.HandleFunc("/add-phrase", AddPhraseHandler)
	http.HandleFunc("/list-phrases", ListPhrasesHandler)

	log.Printf("Server started on :%s\n", port)
	http.ListenAndServe(":"+port, nil)
}
