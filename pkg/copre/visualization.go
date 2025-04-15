package copre

import (
	"log"
	"sort"
	"strings"
)

// ANSI color codes
const (
	red   = "\033[31m"
	reset = "\033[0m"
)

// VisualizePredictions highlights predicted changes within the text.
// It sorts predictions by their mapped position in the new text and applies highlighting.
func VisualizePredictions(text string, predictions []PredictedChange) string {
	// Sort predictions by MappedPosition *ascending* so we process from start to end.
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].MappedPosition < predictions[j].MappedPosition
	})

	var builder strings.Builder
	lastPos := 0

	for _, p := range predictions {
		// Ensure prediction indices are valid for the *current* text length being processed
		if p.MappedPosition < lastPos {
			log.Printf("WARN: Skipping overlapping or out-of-order prediction: %+v", p)
			continue
		}
		// Check if the end position of the removal exceeds the text length
		endPos := p.MappedPosition + len(p.TextToRemove)
		if endPos > len(text) {
			log.Printf("WARN: Skipping prediction out of bounds (end %d > len %d): %+v", endPos, len(text), p)
			continue
		}
		// Check if MappedPosition itself is out of bounds
		if p.MappedPosition > len(text) {
			log.Printf("WARN: Skipping prediction out of bounds (start %d > len %d): %+v", p.MappedPosition, len(text), p)
			continue
		}

		// Append text segment *before* the current prediction
		if p.MappedPosition > lastPos {
			builder.WriteString(text[lastPos:p.MappedPosition])
		}

		// Append the highlighted text to remove
		builder.WriteString(red)
		builder.WriteString(text[p.MappedPosition:endPos])
		builder.WriteString(reset)

		// Update the last position handled to be *after* the removed text
		lastPos = endPos
	}

	// Append the remaining segment of the text (after the last prediction)
	if lastPos < len(text) {
		builder.WriteString(text[lastPos:])
	}

	return builder.String()
}
