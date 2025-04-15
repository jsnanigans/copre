package copre

import (
	"log"
	"strings"
)

// getLocalContext extracts prefix and affix around a given position and length in text.
func getLocalContext(text string, pos int, length int) (prefix string, affix string) {
	// Find the start of the line containing `pos`
	lineStart := strings.LastIndexByte(text[:pos], '\n') + 1 // Handles pos=0 correctly

	// Find the end of the line containing `pos + length - 1`
	lineEndSearchStart := pos + length
	if lineEndSearchStart > len(text) {
		lineEndSearchStart = len(text)
	}
	lineEnd := strings.IndexByte(text[lineEndSearchStart:], '\n')
	if lineEnd == -1 {
		lineEnd = len(text) // End of the file
	} else {
		lineEnd += lineEndSearchStart // Make absolute
	}

	// Extract prefix: from line start up to pos
	if pos >= lineStart {
		prefix = text[lineStart:pos]
	}

	// Extract affix: from pos+length up to line end
	affixStart := pos + length
	if affixStart <= lineEnd {
		affix = text[affixStart:lineEnd]
	}
	return prefix, affix
}

// findAndScoreAnchors searches for occurrences of searchText
// in oldText, skipping the original change location. It scores potential anchors
// based on matching *local* context around the anchor and the original change.
func findAndScoreAnchors(oldText, searchText string, originalChangeStartPos int) []Anchor {
	var anchors []Anchor
	if len(searchText) == 0 || originalChangeStartPos == -1 {
		return anchors
	}

	// Get the context around the original change
	originalPrefix, originalAffix := getLocalContext(oldText, originalChangeStartPos, len(searchText))
	log.Printf("DEBUG: Original Context - Prefix: %q, Affix: %q", originalPrefix, originalAffix)

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

		// Get the local context for this potential anchor
		anchorPrefix, anchorAffix := getLocalContext(oldText, anchorPos, len(searchText))

		// Initialize anchor with base score
		baseScore := 5 // Base score for matching removed text
		score := baseScore

		// Score based on prefix matching (compare anchor's prefix with original's prefix)
		prefixMatchLen := 0
		for i := 1; i <= len(originalPrefix) && i <= len(anchorPrefix); i++ {
			if originalPrefix[len(originalPrefix)-i:] == anchorPrefix[len(anchorPrefix)-i:] {
				prefixMatchLen = i
			} else {
				break
			}
		}
		score += prefixMatchLen

		// Score based on affix matching (compare anchor's affix with original's affix)
		affixMatchLen := 0
		for i := 1; i <= len(originalAffix) && i <= len(anchorAffix); i++ {
			if originalAffix[:i] == anchorAffix[:i] {
				affixMatchLen = i
			} else {
				break
			}
		}
		score += affixMatchLen

		anchors = append(anchors, Anchor{Position: anchorPos, Score: score, Line: anchorLine})

		// Move search start past the current find
		searchStart = anchorPos + 1
		if searchStart >= len(oldText) {
			break
		}
	}
	log.Printf("DEBUG: Found Anchors (using local context): %+v", anchors)
	return anchors
}
