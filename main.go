package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type TransmuteRequest struct {
	ResonanceKey string `json:"resonanceKey"`
	SourceData   string `json:"sourceData"`
	Key5D        string `json:"key5D"`
}

type TransmuteResponse struct {
	Output       string `json:"output"`
	Phrase       string `json:"phrase"`
	Frequency    string `json:"frequency"`
	SourceFormat string `json:"sourceFormat,omitempty"`
	TargetFormat string `json:"targetFormat,omitempty"`
	SourceSample string `json:"sourceSample,omitempty"`
	TargetSample string `json:"targetSample,omitempty"`
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
	ResonanceKey string `json:"resonanceKey"`
	Frequency    string `json:"frequency"`
	SourceFormat string `json:"sourceFormat,omitempty"`
	TargetFormat string `json:"targetFormat,omitempty"`
	SourceSample string `json:"sourceSample,omitempty"`
	TargetSample string `json:"targetSample,omitempty"`
}

var firestoreClient *firestore.Client

func initFirestore(ctx context.Context) error {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}

	secretClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secret manager client: %w", err)
	}
	defer secretClient.Close()

	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/firestore-credentials/versions/latest", projectID),
	}
	result, err := secretClient.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		return fmt.Errorf("failed to access secret version: %w", err)
	}

	credsOption := option.WithCredentialsJSON(result.Payload.Data)
	client, err := firestore.NewClient(ctx, projectID, credsOption)
	if err != nil {
		return fmt.Errorf("failed to create firestore client: %w", err)
	}

	firestoreClient = client
	return nil
}

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
	ctx := context.Background()
	var req TransmuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	docRef := firestoreClient.Collection("phrases").Where("resonanceKey", "==", req.ResonanceKey)
	docs, err := docRef.Documents(ctx).GetAll()
	if err != nil || len(docs) == 0 {
		http.Error(w, "Resonance key not found", http.StatusNotFound)
		return
	}

	var entry PhraseEntry
	docs[0].DataTo(&entry)

	prompt := fmt.Sprintf("Phrase: %s\nSourceData: %s\nKey5D: %s", entry.Phrase, req.SourceData, req.Key5D)
	output, err := callGPT4(prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := TransmuteResponse{
		Output:       output,
		Phrase:       entry.Phrase,
		Frequency:    entry.Frequency,
		SourceFormat: entry.SourceFormat,
		TargetFormat: entry.TargetFormat,
		SourceSample: entry.SourceSample,
		TargetSample: entry.TargetSample,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func AddPhraseHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var entry PhraseEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	entry.ResonanceKey = generateResonanceKey()
	entry.Frequency = "432Hz"

	_, _, err := firestoreClient.Collection("phrases").Add(ctx, entry)
	if err != nil {
		http.Error(w, "Failed to store phrase", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func ListPhrasesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	docs, err := firestoreClient.Collection("phrases").Documents(ctx).GetAll()
	if err != nil {
		http.Error(w, "Failed to fetch phrases", http.StatusInternalServerError)
		return
	}

	phrases := make([]PhraseEntry, 0, len(docs))
	for _, doc := range docs {
		var entry PhraseEntry
		doc.DataTo(&entry)
		phrases = append(phrases, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(phrases)
}

func generateResonanceKey() string {
	salts := []string{"sha", "val", "run", "ek", "ka", "zor", "ten", "vek", "mal"}
	return fmt.Sprintf("RK-%d-%s", time.Now().UnixNano(), salts[rand.Intn(len(salts))])
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))

	ctx := context.Background()

	log.Println("🔥 Starting up...")

	if err := initFirestore(ctx); err != nil {
		log.Printf("❌ Firestore init failed: %v", err)
		os.Exit(1)
	}

	log.Printf("✅ Firestore initialized. Listening on port %s", port)

	http.HandleFunc("/transmute", TransmuteHandler)
	http.HandleFunc("/add-phrase", AddPhraseHandler)
	http.HandleFunc("/list-phrases", ListPhrasesHandler)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("❌ Server error: %v", err)
	}
}
