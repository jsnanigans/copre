package copre

import (
	"github.com/sergi/go-diff/diffmatchpatch"
)

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
