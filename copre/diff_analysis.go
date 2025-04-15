package copre

import (
	"log"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// analyzeDiffs processes the diffs to find the starting position of the first change
// and the text removed in the first continuous block of deletions.
func analyzeDiffs(oldText string, diffs []diffmatchpatch.Diff) (charsRemoved string, originalChangeStartPos int) {
	originalChangeStartPos = -1 // Initialize to -1
	firstChangePosFound := false
	firstDeletionBlockEnded := false
	oldPos := 0

	// First pass: find the start position of the first change
	tempOldPos := 0
	for _, diff := range diffs {
		isChange := diff.Type == diffmatchpatch.DiffInsert || diff.Type == diffmatchpatch.DiffDelete
		if !firstChangePosFound && isChange {
			originalChangeStartPos = tempOldPos
			firstChangePosFound = true
			// break // No, need to continue to calculate oldPos correctly for the second pass
		}
		if diff.Type != diffmatchpatch.DiffInsert { // Only advance if not an insert
			tempOldPos += len(diff.Text)
		}
	}

	// Second pass: find the chars removed in the first block of deletions
	firstChangePosFound = false // Reset for this pass
	for _, diff := range diffs {
		isChange := diff.Type == diffmatchpatch.DiffInsert || diff.Type == diffmatchpatch.DiffDelete
		if !firstChangePosFound && isChange {
			firstChangePosFound = true
		}

		if diff.Type == diffmatchpatch.DiffDelete {
			// Only add to charsRemoved if it's part of the first block
			if firstChangePosFound && !firstDeletionBlockEnded {
				charsRemoved += diff.Text
			}
		} else if firstChangePosFound {
			// If we found the first change and this diff is NOT a delete,
			// then the first block of deletions (if any) has ended.
			firstDeletionBlockEnded = true
		}

		// This part is only needed if we still calculate prefix/affix, which we are removing
		// if diff.Type != diffmatchpatch.DiffInsert {
		// 	oldPos += len(diff.Text)
		// }
	}

	// log.Printf("DEBUG: Characters added: %q", charsAdded) // Removed
	log.Printf("DEBUG: Characters removed (first block): %q", charsRemoved)
	// log.Printf("DEBUG: Prefix context: %q", prefix) // Removed
	// log.Printf("DEBUG: Affix context: %q", affix) // Removed
	log.Printf("DEBUG: Original change start position: %d", originalChangeStartPos)

	return charsRemoved, originalChangeStartPos
}

// contains checks if a slice contains an integer.
// Keep this here as it might be useful, though not currently used by analyzeDiffs
func contains(slice []int, item int) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}
