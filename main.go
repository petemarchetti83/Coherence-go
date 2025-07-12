package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type TransmuteRequest struct {
	Phrase     string `json:"phrase"`
	SourceData string `json:"sourceData"`
}

type ResonanceResponse struct {
	Response  string `json:"response"`
	Frequency string `json:"frequency"`
	Output    string `json:"output"`
}

type AddPhraseRequest struct {
	SourceFormat  string `json:"sourceFormat"`
	TargetFormat  string `json:"targetFormat"`
	SourceSample  string `json:"sourceSample"`
	TargetSample  string `json:"targetSample"`
}

type PhraseEntry struct {
	Phrase        string `json:"phrase"`
	SourceFormat  string `json:"sourceFormat"`
	TargetFormat  string `json:"targetFormat"`
	SourceSample  string `json:"sourceSample"`
	TargetSample  string `json:"targetSample"`
}

var phraseFile = "phrases.json"

func transmuteHandler(w http.ResponseWriter, r *http.Request) {
	var req TransmuteRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var source, target string
	phrases := loadPhrases()
	for _, p := range phrases {
		if p.Phrase == req.Phrase {
			source = p.SourceSample
			target = p.TargetSample
			break
		}
	}

	if source == "" || target == "" {
		http.Error(w, "Phrase not found", http.StatusNotFound)
		return
	}

	llmPrompt := fmt.Sprintf(
		"You are a transmutation engine.\nGiven an example input-output pair, transform new source input to match the target format.\n\nExample Input:\n%s\n\nExample Output:\n%s\n\nNew Input:\n%s\n\nYour Response (Output):",
		source, target, req.SourceData,
	)

	transformed := callOpenAI(llmPrompt)

	response := ResonanceResponse{
		Response:  "Scroll received: " + req.Phrase,
		Frequency: "432Hz",
		Output:    transformed,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func addPhraseHandler(w http.ResponseWriter, r *http.Request) {
	var req AddPhraseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	phrase := generateScrollPhrase(req.SourceFormat, req.TargetFormat)
	entry := PhraseEntry{
		Phrase:        phrase,
		SourceFormat:  req.SourceFormat,
		TargetFormat:  req.TargetFormat,
		SourceSample:  req.SourceSample,
		TargetSample:  req.TargetSample,
	}

	phrases := loadPhrases()
	phrases = append(phrases, entry)
	savePhrases(phrases)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"phrase": phrase,
		"status": "phrase created and stored",
	})
}

func listPhrasesHandler(w http.ResponseWriter, r *http.Request) {
	phrases := loadPhrases()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(phrases)
}

func generateScrollPhrase(source, target string) string {
	rand.Seed(time.Now().UnixNano())

	prefixMap := map[string]string{"edi": "Sha", "json": "El", "xml": "Ka", "csv": "Ra"}
	suffixMap := map[string]string{"edi": "ta", "json": "ruk", "xml": "ven", "csv": "lum"}
	salts := []string{"tur", "sha", "vek", "rin", "dor", "kai", "zan"}

	prefix := prefixMap[strings.ToLower(source)]
	suffix := suffixMap[strings.ToLower(target)]
	salt := salts[rand.Intn(len(salts))]

	return fmt.Sprintf("%s’%s %s", prefix, suffix, salt)
}

func loadPhrases() []PhraseEntry {
	if _, err := os.Stat(phraseFile); os.IsNotExist(err) {
		return []PhraseEntry{}
	}

	content, err := ioutil.ReadFile(phraseFile)
	if err != nil {
		log.Printf("Error reading phrase file: %v", err)
		return []PhraseEntry{}
	}

	var phrases []PhraseEntry
	if err := json.Unmarshal(content, &phrases); err != nil {
		log.Printf("Error unmarshalling phrase file: %v", err)
		return []PhraseEntry{}
	}
	return phrases
}

func savePhrases(phrases []PhraseEntry) {
	content, err := json.MarshalIndent(phrases, "", "  ")
	if err != nil {
		log.Printf("Error marshalling phrase file: %v", err)
		return
	}
	_ = ioutil.WriteFile(phraseFile, content, 0644)
}

func callOpenAI(prompt string) string {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "Missing API key"
	}

	payload := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a transmutation engine."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "Error marshaling payload"
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "Error creating request"
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "Request failed: " + err.Error()
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "Failed to read response: " + err.Error()
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "Failed to decode JSON: " + string(body)
	}

	// Check if OpenAI returned an error
	if errData, ok := result["error"]; ok {
		errMap := errData.(map[string]interface{})
		return fmt.Sprintf("OpenAI Error: %s – %s", errMap["type"], errMap["message"])
	}

	// Extract message
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if msg, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{}); ok {
			if content, ok := msg["content"].(string); ok {
				return content
			}
		}
	}

	// Unexpected format
	return "Unexpected response format: " + string(body)
}

func main() {
	http.HandleFunc("/transmute", transmuteHandler)
	http.HandleFunc("/add-phrase", addPhraseHandler)
	http.HandleFunc("/list-phrases", listPhrasesHandler)
	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}