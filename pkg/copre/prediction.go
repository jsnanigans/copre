package copre

import (
	"log"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// generatePredictions creates potential PredictedChange objects based on scored anchors.
// It maps the anchor position from the old text to the new text and proposes
// applying the same *type* of change (e.g., deletion) found in the original diff.
func generatePredictions(newText string, anchors []Anchor, charsAdded, charsRemoved string, diffs []diffmatchpatch.Diff) []PredictedChange {
	var predictions []PredictedChange
	log.Printf("DEBUG: Generating predictions. Anchors: %d, CharsAdded: %q, CharsRemoved: %q", len(anchors), charsAdded, charsRemoved)

	if len(charsRemoved) == 0 {
		return predictions // No deletion predictions if nothing was removed
	}

	for _, anchor := range anchors {
		// Map the anchor position (oldText) to the corresponding position in newText
		mappedPos := mapPosition(anchor.Position, diffs)

		// Basic check: Ensure the text to remove actually exists at the mapped position in the new text.
		// This prevents errors if the mapping is complex or the surrounding context changed drastically.
		if mappedPos+len(charsRemoved) <= len(newText) && newText[mappedPos:mappedPos+len(charsRemoved)] == charsRemoved {
			predictions = append(predictions, PredictedChange{
				Position:       anchor.Position, // Keep original position for reference
				TextToRemove:   charsRemoved,
				Line:           anchor.Line, // Line number in oldText
				Score:          anchor.Score,
				MappedPosition: mappedPos, // Position in newText
			})
		} else {
			log.Printf("WARN: Skipping prediction at oldPos %d (mapped to %d) because '%s' not found in newText at that location.",
				anchor.Position, mappedPos, charsRemoved)
		}
	}
	log.Printf("DEBUG: Generated Predictions: %+v", predictions) // Log predictions including mapped positions
	return predictions
}
