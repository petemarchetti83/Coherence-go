
package main

import (
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
    Response  string   `xml:"response"`
    Frequency string   `xml:"frequency"`
    Output    string   `xml:"output"`
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


import (
    "bytes"
    "net/http"
    "os"
)

func transmuteHandler(w http.ResponseWriter, r *http.Request) {

    var req TransmuteRequest
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    
    target := ""
    source := ""

    phrases := loadPhrases()
    for _, p := range phrases {
        if p.Phrase == req.Phrase {
            source = p.SourceSample
            target = p.TargetSample
            break
        }
    }

    if target == "" || source == "" {
        http.Error(w, "Phrase not found", http.StatusNotFound)
        return
    }

    llmPrompt := "You are a transmutation engine.\n" +
        "Given an example input-output pair, transform new source input to match the target format.\n\n" +
        "Example Input:\n" + source + "\n\n" +
        "Example Output:\n" + target + "\n\n" +
        "New Input:\n" + req.SourceData + "\n\n" +
        "Your Response (Output):"

    transformed := callOpenAI(llmPrompt)

    if target == "" {
        http.Error(w, "Phrase not found", http.StatusNotFound)
        return
    }

    response := ResonanceResponse{
        Response:  "Scroll received: " + req.Phrase,
        Frequency: "432Hz",
        Output:    target,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func addPhraseHandler(w http.ResponseWriter, r *http.Request) {
    var req AddPhraseRequest
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }

    phrase := generateScrollPhrase(req.SourceFormat, req.TargetFormat)
    entry := PhraseEntry{
        Phrase:       phrase,
        SourceFormat: req.SourceFormat,
        TargetFormat: req.TargetFormat,
        SourceSample: req.SourceSample,
        TargetSample: req.TargetSample,
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

func lookupTargetSample(phrase string) string {
    phrases := loadPhrases()
    for _, p := range phrases {
        if p.Phrase == phrase {
            return p.TargetSample
        }
    }
    return ""
}

func generateScrollPhrase(source, target string) string {
    rand.Seed(time.Now().UnixNano())

    prefixMap := map[string]string{"edi": "Sha", "json": "El", "xml": "Ka", "csv": "Ra"}
    suffixMap := map[string]string{"edi": "ta", "json": "ruk", "xml": "ven", "csv": "lum"}
    salt := []string{"tur", "sha", "vek", "rin", "dor", "kai", "zan"}

    prefix := prefixMap[strings.ToLower(source)]
    suffix := suffixMap[strings.ToLower(target)]
    middle := salt[rand.Intn(len(salt))]

    return fmt.Sprintf("%s’%s %s", prefix, suffix, middle)
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

func main() {
    http.HandleFunc("/transmute", transmuteHandler)
    http.HandleFunc("/add-phrase", addPhraseHandler)
    log.Println("Listening on :8080")
    http.HandleFunc("/list-phrases", listPhrasesHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func listPhrasesHandler(w http.ResponseWriter, r *http.Request) {
    phrases := loadPhrases()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(phrases)
}

func callOpenAI(prompt string) string {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        return "Missing API key"
    }

    payload := `{
        "model": "gpt-4",
        "messages": [
            {"role": "system", "content": "You are a transmutation engine."},
            {"role": "user", "content": "` + prompt + `"}
        ],
        "temperature": 0.1
    }`

    req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer([]byte(payload)))
    if err != nil {
        return "Error creating request"
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer " + apiKey)

    client := &http.Client{}
    res, err := client.Do(req)
    if err != nil {
        return "Request failed"
    }
    defer res.Body.Close()

    var result map[string]interface{}
    _ = json.NewDecoder(res.Body).Decode(&result)

    if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
        if msg, ok := choices[0].(map[string]interface{})["message"].(map[string]interface{}); ok {
            return msg["content"].(string)
        }
    }

    return "Failed to parse response"
}
