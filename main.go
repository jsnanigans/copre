package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// FileUpdateRequest defines the structure for incoming JSON requests.
type FileUpdateRequest struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// CachedFileContent stores the current and previous content of a file.
type CachedFileContent struct {
	Current  string
	Previous string // Content before the latest update
}

// PredictedChange represents a suggested code modification.
type PredictedChange struct {
	LineNumber    int    // 1-based line number in the *new* content
	OriginalLine  string // The line content before prediction
	PredictedLine string // The suggested new line content
}

var (
	// fileCache stores the last known content and previous content for each file.
	fileCache = make(map[string]CachedFileContent)
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

// predictChanges analyzes the difference between old and new content
// and suggests potential follow-up changes.
func predictChanges(filename, oldContent, newContent string) []PredictedChange {
	// Placeholder implementation - will be filled in later
	log.Printf("[Predictor] Analyzing changes for %s", filename)
	predictions := []PredictedChange{}

	if oldContent == "" || newContent == "" {
		return predictions // Cannot predict without both versions
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldContent, newContent, false)

	// TODO: Implement more sophisticated prediction logic based on diffs
	// For now, just log the diffs
	log.Printf("[Predictor] Diffs for %s: %s", filename, dmp.DiffPrettyText(diffs))

	// --- Simple Example Logic (to be refined) ---
	// This is a very basic example trying to find a simple replacement
	// oldLines := strings.Split(oldContent, "\n") // Keep for potential future use
	newLines := strings.Split(newContent, "\n")
	chars1, chars2, lineArray := dmp.DiffLinesToChars(oldContent, newContent)

	diffs = dmp.DiffMain(chars1, chars2, false)
	dmp.DiffCharsToLines(diffs, lineArray)

	// Find a simple replacement diff
	var replacedOld, replacedNew string
	replacementFound := false
	for i := 0; i < len(diffs)-1; i++ {
		if diffs[i].Type == diffmatchpatch.DiffDelete && diffs[i+1].Type == diffmatchpatch.DiffInsert {
			// Simple adjacent delete/insert might be a replacement
			// Consider only single-line replacements for now
			if !strings.Contains(diffs[i].Text, "\n") && !strings.Contains(diffs[i+1].Text, "\n") {
				replacedOld = strings.TrimSuffix(diffs[i].Text, "\n") // Trim potential trailing newline from diff chunk
				replacedNew = strings.TrimSuffix(diffs[i+1].Text, "\n")
				replacementFound = true
				log.Printf("[Predictor] Found potential replacement: %q -> %q", replacedOld, replacedNew)
				break // Handle one replacement pattern for now
			}
		}
	}

	if replacementFound && replacedOld != "" {
		for i, line := range newLines {
			// Avoid suggesting changes on the line that was just changed
			// This check is simplistic and needs improvement
			if line == replacedNew {
				continue
			}

			if strings.Contains(line, replacedOld) {
				predicted := strings.ReplaceAll(line, replacedOld, replacedNew)
				if predicted != line { // Ensure a change actually happened
					predictions = append(predictions, PredictedChange{
						LineNumber:    i + 1, // 1-based index
						OriginalLine:  line,
						PredictedLine: predicted,
					})
					log.Printf("[Predictor] Suggesting change on line %d: %q -> %q", i+1, line, predicted)
				}
			}
		}
	}

	// --- End Simple Example Logic ---

	return predictions
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

	cacheMutex.Lock() // Lock for read and potential write
	cachedData, exists := fileCache[req.Filename]
	oldContent := "" // Initialize oldContent
	if exists {
		oldContent = cachedData.Current // Get the actual previous content
	}

	// Store previous and new content
	newCacheEntry := CachedFileContent{
		Current:  req.Content,
		Previous: oldContent, // Store the content *before* this update
	}
	fileCache[req.Filename] = newCacheEntry
	cacheMutex.Unlock() // Unlock after cache update

	// Perform analysis and prediction *after* releasing the lock
	if exists && oldContent != req.Content { // Only predict if content changed
		log.Printf("Content updated for %s. Analyzing changes...", req.Filename)
		// Use the oldContent retrieved before the cache update
		added, removed := diffStrings(oldContent, req.Content)
		if added != "" || removed != "" {
			log.Printf("  Raw diff: Removed: %q, Added: %q", removed, added)
		}

		// Trigger prediction
		predictions := predictChanges(req.Filename, oldContent, req.Content)
		if len(predictions) > 0 {
			log.Printf("Predictions for %s:", req.Filename)
			for _, p := range predictions {
				log.Printf("  L%d: - %s", p.LineNumber, p.OriginalLine)
				log.Printf("      + %s", p.PredictedLine)
			}
		} else {
			log.Printf("No specific predictions generated for %s", req.Filename)
		}

	} else if !exists {
		log.Printf("Caching initial content for %s", req.Filename)
		if req.Content != "" {
			log.Printf("  Initial content: %q", req.Content)
		}
	} else {
		log.Printf("No changes detected in %s", req.Filename)
	}

	fmt.Fprintf(w, "Content for %s received and processed.\n", req.Filename)
}

func main() {
	http.HandleFunc("/update", fileUpdateHandler)

	port := "8080"
	log.Printf("Starting server on localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
