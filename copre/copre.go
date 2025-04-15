package copre

import (
	"log"
	// "sort"
	// "strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// PredictNextChanges analyzes the differences between oldText and newText
// to predict the next likely changes (currently focusing on deletions).
func PredictNextChanges(oldText, newText string) ([]PredictedChange, error) {
	log.Printf("DEBUG: oldText:\n%s", oldText)
	log.Printf("DEBUG: newText:\n%s", newText)

	// 1. Calculate Diffs
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, true) // Use character-level diff
	log.Printf("DEBUG: Diffs: %s", dmp.DiffPrettyText(diffs))

	// 2. Analyze Diffs
	_, charsRemoved, prefix, affix, originalChangeStartPos := analyzeDiffs(oldText, diffs)
	// _, _, _, _, _ = charsAdded, charsRemoved, prefix, affix, originalChangeStartPos // Use charsAdded if needed later

	// 3. Find and Score Anchors based on removed text and context
	// TODO: Adapt anchor finding/scoring for insertions/replacements
	anchors := findAndScoreAnchors(oldText, charsRemoved, prefix, affix, originalChangeStartPos)

	// 4. Generate Predictions from Anchors
	predictions := generatePredictions(newText, anchors, charsRemoved, diffs)
	// TODO: Add prediction generation logic for insertions/replacements

	// 5. Sort/Filter Predictions (Optional)
	// Sort predictions by score (descending) - higher score is more likely
	// sort.Slice(predictions, func(i, j int) bool {
	// 	return predictions[i].Score > predictions[j].Score
	// })

	return predictions, nil
}
