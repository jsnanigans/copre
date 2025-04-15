package copre

import (
	"log"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// analyzeDiffs processes the diffs to extract relevant information about the changes.
// It identifies added/removed characters, the prefix/affix context around single-line changes,
// and the starting position of the first effective change in the old text.
func analyzeDiffs(oldText string, diffs []diffmatchpatch.Diff) (charsAdded, charsRemoved, prefix, affix string, originalChangeStartPos int) {
	var linesChanged []int    // Keep track for prefix/affix logic, not returned
	prefixSetForLine := false // Flag to ensure prefix/affix are set only once per line change sequence

	currentLine := 1
	oldPos := 0
	startLineOfChange := -1     // Track the start line of the current modification sequence
	originalChangeStartPos = -1 // Initialize to -1
	firstChangePosFound := false

	for _, diff := range diffs { // Use blank identifier for index
		textLines := strings.Split(diff.Text, "\n")
		numLines := len(textLines)
		currentDiffStartOldPos := oldPos // Track start position for this specific diff

		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			charsRemoved += diff.Text
			if !firstChangePosFound {
				originalChangeStartPos = currentDiffStartOldPos
				firstChangePosFound = true
			}

			startLine := currentLine
			if startLineOfChange == -1 {
				startLineOfChange = startLine
			}
			for j := 0; j < numLines-1; j++ {
				if !contains(linesChanged, currentLine) {
					linesChanged = append(linesChanged, currentLine)
				}
				currentLine++
			}
			if !contains(linesChanged, currentLine) {
				linesChanged = append(linesChanged, currentLine)
			}

			// Get prefix/affix for deletions within a single line
			if startLine == currentLine && strings.Index(diff.Text, "\n") == -1 && !prefixSetForLine {
				lineStart := strings.LastIndexByte(oldText[:currentDiffStartOldPos], '\n')
				if lineStart == -1 {
					lineStart = 0 // Beginning of the file
				} else {
					lineStart++ // Move past the newline
				}

				lineEnd := strings.IndexByte(oldText[currentDiffStartOldPos:], '\n')
				if lineEnd == -1 {
					lineEnd = len(oldText) // End of the file
				} else {
					lineEnd += currentDiffStartOldPos // Absolute position
				}

				if currentDiffStartOldPos > lineStart {
					prefix = oldText[lineStart:currentDiffStartOldPos]
				}
				changeEndPos := currentDiffStartOldPos + len(diff.Text)
				if changeEndPos < lineEnd {
					affix = oldText[changeEndPos:lineEnd]
				}
				prefixSetForLine = true
			}
			oldPos += len(diff.Text) // Advance old position

		case diffmatchpatch.DiffInsert:
			charsAdded += diff.Text
			if !firstChangePosFound {
				// The original position is the current `oldPos` (which is currentDiffStartOldPos here)
				originalChangeStartPos = currentDiffStartOldPos
				firstChangePosFound = true
			}

			startLine := currentLine
			if startLineOfChange == -1 {
				startLineOfChange = startLine
			}
			for j := 0; j < numLines-1; j++ {
				if !contains(linesChanged, currentLine) {
					linesChanged = append(linesChanged, currentLine)
				}
				currentLine++
			}
			if !contains(linesChanged, currentLine) {
				linesChanged = append(linesChanged, currentLine)
			}

			// Get prefix/affix for insertions within a single line
			if startLine == currentLine && strings.Index(diff.Text, "\n") == -1 && !prefixSetForLine {
				lineStart := strings.LastIndexByte(oldText[:currentDiffStartOldPos], '\n')
				if lineStart == -1 {
					lineStart = 0 // Beginning of the file
				} else {
					lineStart++ // Move past the newline
				}

				lineEnd := strings.IndexByte(oldText[currentDiffStartOldPos:], '\n')
				if lineEnd == -1 {
					lineEnd = len(oldText) // End of the file
				} else {
					lineEnd += currentDiffStartOldPos // Absolute position
				}

				if currentDiffStartOldPos > lineStart {
					prefix = oldText[lineStart:currentDiffStartOldPos]
				}
				// Affix starts right at the insertion point in the old text
				if currentDiffStartOldPos < lineEnd {
					affix = oldText[currentDiffStartOldPos:lineEnd]
				}
				prefixSetForLine = true
			}
			// Note: We don't advance oldPos for insertions

		case diffmatchpatch.DiffEqual:
			if startLineOfChange != -1 {
				prefixSetForLine = false
				startLineOfChange = -1
			}
			currentLine += numLines - 1
			oldPos += len(diff.Text)
		}
	}
	log.Printf("DEBUG: Lines changed: %v", linesChanged) // Kept for debugging visibility
	log.Printf("DEBUG: Characters added: %q", charsAdded)
	log.Printf("DEBUG: Characters removed: %q", charsRemoved)
	log.Printf("DEBUG: Prefix context: %q", prefix)
	log.Printf("DEBUG: Affix context: %q", affix)
	log.Printf("DEBUG: Original change start position: %d", originalChangeStartPos)

	return charsAdded, charsRemoved, prefix, affix, originalChangeStartPos
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
