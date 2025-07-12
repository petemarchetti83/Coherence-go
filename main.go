
package main

import (
    "encoding/json"
    "encoding/xml"
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
    Phrase string `json:"phrase"`
}

type ResonanceResponse struct {
    XMLName   xml.Name `xml:"resonance"`
    Response  string   `xml:"response"`
    Frequency string   `xml:"frequency"`
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

    response := ResonanceResponse{
        Response:  "Scroll received: " + req.Phrase,
        Frequency: "432Hz",
    }

    w.Header().Set("Content-Type", "application/xml")
    xml.NewEncoder(w).Encode(response)
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
    log.Fatal(http.ListenAndServe(":8080", nil))
}
