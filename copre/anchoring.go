package copre

import (
	"log"
	"strings"
)

// findAndScoreAnchors searches for occurrences of searchText (typically removed text)
// in oldText, skipping the original change location. It scores potential anchors
// based on matching prefix and affix context.
func findAndScoreAnchors(oldText, searchText, prefix, affix string, originalChangeStartPos int) []Anchor {
	var anchors []Anchor
	if len(searchText) == 0 || originalChangeStartPos == -1 {
		// Don't search if nothing was removed or we don't know where the original change was.
		return anchors
	}

	searchStart := 0
	for {
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
		affixCheckStartPos := anchorPos + len(searchText) // Position right after the potential removal
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
	log.Printf("DEBUG: Found Anchors: %+v", anchors)
	return anchors
}
