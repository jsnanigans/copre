package copre

import (
	"log"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// PredictNextChanges analyzes the differences between oldText and newText
// to predict the next likely change.
func PredictNextChanges(oldText, newText string) (string, error) {
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
		Position int
		Score    int
		Line     int // Add line number for better context
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
	if len(charsRemoved) > 0 && originalChangeStartPos != -1 {
		searchStart := 0
		for {
			// Find next occurrence of the removed text
			foundPos := strings.Index(oldText[searchStart:], charsRemoved)
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

			// Initialize anchor with base score for matching removed text
			anchor := Anchor{Position: anchorPos, Score: 5, Line: anchorLine}

			// Score based on prefix matching (inside-out)
			for i := 1; i <= len(prefix); i++ {
				prefixMatchPos := anchorPos - i
				if prefixMatchPos >= 0 && oldText[prefixMatchPos:anchorPos] == prefix[len(prefix)-i:] {
					anchor.Score++
				} else {
					// Stop prefix scoring if a longer match fails
					// (We could make this more lenient, but strict for now)
					break
				}
			}

			// Score based on affix matching (inside-out)
			affixStartPos := anchorPos + len(charsRemoved)
			for i := 1; i <= len(affix); i++ {
				affixMatchEndPos := affixStartPos + i
				if affixMatchEndPos <= len(oldText) && oldText[affixStartPos:affixMatchEndPos] == affix[:i] {
					anchor.Score++
				} else {
					// Stop affix scoring if a longer match fails
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

	// --- Prediction Logic (Placeholder) ---
	// Based on the analysis (linesChanged, charsAdded, charsRemoved, prefix, affix),
	// predict the next change. This part requires a more sophisticated model
	// or heuristic.

	return "___not_implemented___", nil // Keep placeholder for now
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
