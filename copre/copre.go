package copre

import (
	"log"
	"strings"

	// TODO: Add sorting import if needed later
	// "sort"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// PredictedChange represents a potential future edit.
// Currently, it only supports deletions.
// TODO: Support insertions and replacements.
type PredictedChange struct {
	Position     int    // Byte offset in oldText where the change originates
	TextToRemove string // The text to be removed
	// TextToAdd string // Future: Text to add (for insertions/replacements)
	Line           int // Line number in oldText where the change originates (1-based)
	Score          int // Confidence score for this prediction
	MappedPosition int // Corresponding byte offset in newText where the change should be applied
}

// mapPosition translates a byte offset from oldText to its corresponding offset in newText
// based on the provided diffs.
func mapPosition(oldPos int, diffs []diffmatchpatch.Diff) int {
	currentOldPos := 0
	currentNewPos := 0

	for _, diff := range diffs {
		diffLen := len(diff.Text)
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			// If the old position is within the deleted section,
			// map it to the position right before the deletion in the new text.
			if oldPos >= currentOldPos && oldPos < currentOldPos+diffLen {
				return currentNewPos
			}
			currentOldPos += diffLen
		case diffmatchpatch.DiffInsert:
			currentNewPos += diffLen
		case diffmatchpatch.DiffEqual:
			// If the old position is within this equal section, calculate the corresponding new position.
			if oldPos >= currentOldPos && oldPos < currentOldPos+diffLen {
				return currentNewPos + (oldPos - currentOldPos)
			}
			currentOldPos += diffLen
			currentNewPos += diffLen
		}
	}
	// If the position is after all diffs (e.g., at the very end of the old text),
	// return the end position of the new text.
	if oldPos >= currentOldPos {
		return currentNewPos
	}
	// Should ideally not be reached if oldPos is valid, but return currentNewPos as fallback.
	return currentNewPos
}

// PredictNextChanges analyzes the differences between oldText and newText
// to predict the next likely changes.
func PredictNextChanges(oldText, newText string) ([]PredictedChange, error) {
	log.Printf("DEBUG: oldText:\n%s", oldText)
	log.Printf("DEBUG: newText:\n%s", newText)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, true) // Use character-level diff

	log.Printf("DEBUG: Diffs: %s", dmp.DiffPrettyText(diffs))

	// Analyze diffs to identify changes
	var linesChanged []int
	var charsAdded, charsRemoved string
	var prefix, affix string  // Context for single-line changes
	prefixSetForLine := false // Flag to ensure prefix/affix are set only once per line change sequence

	currentLine := 1
	oldPos := 0
	startLineOfChange := -1 // Track the start line of the current modification sequence

	for _, diff := range diffs {
		textLines := strings.Split(diff.Text, "\n")
		numLines := len(textLines)

		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			charsRemoved += diff.Text
			startLine := currentLine
			if startLineOfChange == -1 {
				startLineOfChange = startLine
			}
			for i := 0; i < numLines-1; i++ {
				if !contains(linesChanged, currentLine) {
					linesChanged = append(linesChanged, currentLine)
				}
				currentLine++
			}
			if !contains(linesChanged, currentLine) {
				linesChanged = append(linesChanged, currentLine)
			}

			// Get prefix/affix for deletions within a single line, only if not already set for this line
			if startLine == currentLine && strings.Index(diff.Text, "\n") == -1 && !prefixSetForLine {
				lineStart := strings.LastIndexByte(oldText[:oldPos], '\n') + 1 // Find start of current line
				lineEnd := strings.IndexByte(oldText[oldPos:], '\n')
				if lineEnd == -1 { // Last line
					lineEnd = len(oldText)
				} else {
					lineEnd += oldPos
				}

				if oldPos >= lineStart {
					prefix = oldText[lineStart:oldPos]
				}
				changeEndPos := oldPos + len(diff.Text)
				if changeEndPos <= lineEnd {
					affix = oldText[changeEndPos:lineEnd]
				}
				prefixSetForLine = true // Mark as set for this line change
			}
			oldPos += len(diff.Text) // Advance old position

		case diffmatchpatch.DiffInsert:
			charsAdded += diff.Text
			startLine := currentLine
			if startLineOfChange == -1 {
				startLineOfChange = startLine
			}
			for i := 0; i < numLines-1; i++ {
				if !contains(linesChanged, currentLine) {
					linesChanged = append(linesChanged, currentLine)
				}
				currentLine++
			}
			if !contains(linesChanged, currentLine) {
				linesChanged = append(linesChanged, currentLine)
			}

			// Get prefix/affix for insertions within a single line, only if not already set for this line
			if startLine == currentLine && strings.Index(diff.Text, "\n") == -1 && !prefixSetForLine {
				lineStart := strings.LastIndexByte(oldText[:oldPos], '\n') + 1 // Find start of current line in old text
				lineEnd := strings.IndexByte(oldText[oldPos:], '\n')
				if lineEnd == -1 { // Last line
					lineEnd = len(oldText)
				} else {
					lineEnd += oldPos
				}

				if oldPos >= lineStart {
					prefix = oldText[lineStart:oldPos]
				}
				if oldPos <= lineEnd { // Affix starts right after the insertion point in the old text
					affix = oldText[oldPos:lineEnd]
				}
				prefixSetForLine = true // Mark as set for this line change
			}
			// Note: We don't advance oldPos for insertions

		case diffmatchpatch.DiffEqual:
			// If we encounter an equal diff after a change, the line modification sequence is over.
			if startLineOfChange != -1 {
				prefixSetForLine = false // Reset the flag for the next potential line change
				startLineOfChange = -1
			}
			currentLine += numLines - 1
			oldPos += len(diff.Text)
		}
	}

	log.Printf("DEBUG: Lines changed: %v", linesChanged)
	log.Printf("DEBUG: Characters added: %q", charsAdded)
	log.Printf("DEBUG: Characters removed: %q", charsRemoved)
	log.Printf("DEBUG: Prefix context: %q", prefix)
	log.Printf("DEBUG: Affix context: %q", affix)

	// --- Anchor Finding and Scoring ---
	type Anchor struct {
		Position int // Position in oldText
		Score    int
		Line     int // Line number in oldText
	}
	var anchors []Anchor
	originalChangeStartPos := -1 // Track the start position of the original change in oldText

	// Need to find the *actual* start position of the first change that contributed to the current context
	tempOldPos := 0
	firstChangePosFound := false
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			if !firstChangePosFound {
				originalChangeStartPos = tempOldPos
				firstChangePosFound = true
			}
			tempOldPos += len(diff.Text)
		case diffmatchpatch.DiffInsert:
			if !firstChangePosFound {
				originalChangeStartPos = tempOldPos
				firstChangePosFound = true
			}
			// No change in tempOldPos for insert
		case diffmatchpatch.DiffEqual:
			tempOldPos += len(diff.Text)
		}
	}
	log.Printf("DEBUG: Original change start position: %d", originalChangeStartPos)

	// Only search for anchors if something was actually removed
	// TODO: Refine anchor finding for insertions/replacements
	if len(charsRemoved) > 0 && originalChangeStartPos != -1 {
		searchStart := 0
		searchText := charsRemoved

		for {
			// Find next occurrence of the search text
			foundPos := strings.Index(oldText[searchStart:], searchText)
			if foundPos == -1 {
				break // No more occurrences
			}

			anchorPos := searchStart + foundPos // Absolute position in oldText

			// Skip the original change location
			if anchorPos == originalChangeStartPos {
				searchStart = anchorPos + 1 // Start searching after this occurrence
				if searchStart >= len(oldText) {
					break
				}
				continue
			}

			// Calculate line number for the anchor
			anchorLine := 1 + strings.Count(oldText[:anchorPos], "\n")

			// Initialize anchor with base score
			baseScore := 5 // Base score for matching removed text
			anchor := Anchor{Position: anchorPos, Score: baseScore, Line: anchorLine}

			// Score based on prefix matching (inside-out)
			for i := 1; i <= len(prefix); i++ {
				prefixMatchPos := anchorPos - i
				if prefixMatchPos >= 0 && oldText[prefixMatchPos:anchorPos] == prefix[len(prefix)-i:] {
					anchor.Score++
				} else {
					break
				}
			}

			// Score based on affix matching (inside-out)
			affixCheckStartPos := anchorPos + len(charsRemoved) // Position right after the potential removal
			for i := 1; i <= len(affix); i++ {
				affixMatchEndPos := affixCheckStartPos + i
				if affixMatchEndPos <= len(oldText) && oldText[affixCheckStartPos:affixMatchEndPos] == affix[:i] {
					anchor.Score++
				} else {
					break
				}
			}

			anchors = append(anchors, anchor)

			// Move search start past the current find
			searchStart = anchorPos + 1
			if searchStart >= len(oldText) {
				break
			}
		}
	}

	log.Printf("DEBUG: Found Anchors: %+v", anchors)

	// --- Prediction Logic ---
	var predictions []PredictedChange

	// Convert anchors into predictions (currently only deletions)
	if len(charsRemoved) > 0 { // Only generate deletion predictions if something was removed
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
	}
	// TODO: Add logic for insertions/replacements

	log.Printf("DEBUG: Generated Predictions: %+v", predictions) // Log predictions including mapped positions

	// Sort predictions by score (descending) - higher score is more likely
	// TODO: Implement sorting if needed, or perhaps filter by score.
	// sort.Slice(predictions, func(i, j int) bool {
	// 	return predictions[i].Score > predictions[j].Score
	// })

	return predictions, nil
}

// contains checks if a slice contains an integer.
func contains(slice []int, item int) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

// // getContext extracts context around a position in the text.
// // Negative size means context before the position, positive means after.
// func getContext(text string, pos int, size int) string {
// 	runes := []rune(text)
// 	start := pos
// 	end := pos
//
// 	if size < 0 { // Prefix
// 		start = pos + size
// 		if start < 0 {
// 			start = 0
// 		}
// 	} else { // Affix
// 		end = pos + size
// 		if end > len(runes) {
// 			end = len(runes)
// 		}
// 	}
//
// 	if start >= len(runes) || end < 0 || start > end {
// 		return ""
// 	}
//
// 	return string(runes[start:end])
// }
