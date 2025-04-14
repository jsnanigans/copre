package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// FileUpdateRequest defines the structure for incoming JSON requests.
type FileUpdateRequest struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

var (
	// fileCache stores the last known content for each file.
	fileCache = make(map[string]string)
	// cacheMutex protects concurrent access to fileCache.
	cacheMutex sync.RWMutex
)

// diffStrings finds the characters added and removed between two strings
// by identifying the longest common prefix and suffix.
func diffStrings(oldStr, newStr string) (added string, removed string) {
	oldLen := len(oldStr)
	newLen := len(newStr)

	// Find length of common prefix
	prefixLen := 0
	for prefixLen < oldLen && prefixLen < newLen && oldStr[prefixLen] == newStr[prefixLen] {
		prefixLen++
	}

	// Find length of common suffix
	suffixLen := 0
	for suffixLen < oldLen-prefixLen && suffixLen < newLen-prefixLen && oldStr[oldLen-1-suffixLen] == newStr[newLen-1-suffixLen] {
		suffixLen++
	}

	// Extract the differing parts
	if prefixLen+suffixLen <= oldLen {
		removed = oldStr[prefixLen : oldLen-suffixLen]
	}
	if prefixLen+suffixLen <= newLen {
		added = newStr[prefixLen : newLen-suffixLen]
	}

	return added, removed
}

// fileUpdateHandler handles incoming requests to update file content.
func fileUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FileUpdateRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Filename == "" {
		http.Error(w, "Filename cannot be empty", http.StatusBadRequest)
		return
	}

	cacheMutex.Lock() // Lock for potential write
	oldContent, exists := fileCache[req.Filename]

	if exists {
		added, removed := diffStrings(oldContent, req.Content)
		if added != "" || removed != "" {
			log.Printf("Changes detected in %s:", req.Filename)
			if removed != "" {
				log.Printf("  Removed: %q", removed)
			}
			if added != "" {
				log.Printf("  Added:   %q", added)
			}
		} else {
			log.Printf("No changes detected in %s", req.Filename)
		}
	} else {
		log.Printf("Caching initial content for %s", req.Filename)
		// Log the entire content as "added" for the first time
		if req.Content != "" {
			log.Printf("  Added:   %q", req.Content)
		}
	}

	// Update cache
	fileCache[req.Filename] = req.Content
	cacheMutex.Unlock()

	fmt.Fprintf(w, "Content for %s received and processed.\n", req.Filename)
}

func main() {
	http.HandleFunc("/update", fileUpdateHandler)

	port := "8080"
	log.Printf("Starting server on localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
