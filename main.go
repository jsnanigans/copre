package main

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/jsnanigans/copre/copre"
)

// ANSI color codes
const (
	reset = "\033[0m"
	red   = "\033[31m"
	// green = "\033[32m" // For future use (insertions)
)

// visualizePredictions takes the new text and a slice of predictions,
// and returns a string with ANSI color codes highlighting the predicted changes.
// Currently highlights deletions in red.
func visualizePredictions(text string, predictions []copre.PredictedChange) string {
	// Sort predictions by MappedPosition descending so we process later changes first.
	// This avoids messing up indices of earlier changes when we modify the string.
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].MappedPosition > predictions[j].MappedPosition
	})

	var buf bytes.Buffer
	lastPos := len(text)

	for _, p := range predictions {
		// Append text segment after the current prediction up to the last position handled
		if p.MappedPosition+len(p.TextToRemove) < lastPos {
			buf.WriteString(text[p.MappedPosition+len(p.TextToRemove) : lastPos])
		}

		// Append the highlighted text to remove
		buf.WriteString(red)
		buf.WriteString(text[p.MappedPosition : p.MappedPosition+len(p.TextToRemove)])
		buf.WriteString(reset)

		// Append text segment before the current prediction (handled in next iteration or initial segment)

		lastPos = p.MappedPosition // Update the last position handled
	}

	// Append the initial segment of the text (before the first prediction)
	buf.WriteString(text[:lastPos])

	// The buffer was built in reverse order, so we need to reverse it.
	// This is a bit inefficient but straightforward for now.
	segments := make([]string, 0)
	// Rough split - could be more precise, but this works for visualization
	// Split based on reset code occurrences to get segments
	parts := strings.Split(buf.String(), reset)
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			// Re-add reset if it was a delimiter (except possibly for the last part)
			if i < len(parts)-1 || strings.HasSuffix(buf.String(), reset) {
				segments = append(segments, parts[i]+reset)
			} else {
				segments = append(segments, parts[i])
			}
		}
	}

	return strings.Join(segments, "")

}

func main() {
	oldText := `line one-smile
line two-smile
line 3-smile`
	newText := `line one-smile
line two
line 3-smile`

	predictions, err := copre.PredictNextChanges(oldText, newText)
	if err != nil {
		log.Fatalf("Error predicting changes: %v", err)
	}

	// Visualize the predictions on the new text
	if len(predictions) > 0 {
		fmt.Println("--- Predicted Changes Preview ---")
		visualizedText := visualizePredictions(newText, predictions)
		fmt.Println(visualizedText)
		fmt.Println("---------------------------------")
	} else {
		fmt.Println("No specific next changes predicted based on anchors.")
	}

	// Keep the detailed log for debugging if needed
	// fmt.Printf("Predicted next changes (raw): %+v\n", predictions)
}
