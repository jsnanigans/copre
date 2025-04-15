package copre

import (
	"log"
	"strings"
	"unicode/utf8"
)

// getLocalContext extracts prefix and affix around a given position and length in text,
// ensuring the context is limited to the same line as the start position 'pos'
// and correctly handles Unicode characters.
func getLocalContext(text string, pos int, length int) (prefix string, affix string) {
	if pos < 0 || pos > len(text) {
		return "", "" // Invalid position
	}

	// Find the start of the line containing `pos` (byte index)
	lineStart := 0
	if pos > 0 {
		lastNewline := strings.LastIndexByte(text[:pos], '\n')
		if lastNewline != -1 {
			lineStart = lastNewline + 1
		}
	}

	// Find the end of the line containing `pos` (byte index)
	lineEnd := strings.IndexByte(text[lineStart:], '\n')
	if lineEnd == -1 {
		lineEnd = len(text) // End of the text
	} else {
		lineEnd += lineStart // Make absolute index
	}

	// Work with the relevant line text
	lineText := text[lineStart:lineEnd]
	lineRunes := []rune(lineText)

	// Calculate byte position relative to the start of the line
	relativePosBytes := pos - lineStart
	if relativePosBytes < 0 {
		relativePosBytes = 0
	} // Clamp to line start

	// Convert relative byte position and length to rune indices within the line
	prefixEndRuneIndex := 0
	affixStartRuneIndex := 0
	currentBytePos := 0
	foundPrefixEnd := false

	for i, r := range lineRunes {
		runeLen := utf8.RuneLen(r)
		// Check if the start of the *next* rune is past our target byte position
		if !foundPrefixEnd && currentBytePos+runeLen > relativePosBytes {
			// If the exact byte position matches the start of this rune, use current index
			if currentBytePos == relativePosBytes {
				prefixEndRuneIndex = i
			} else {
				// Otherwise, the byte position falls within the *previous* rune's span.
				// But our prefix should end *before* the rune containing `pos`.
				// So the index `i` (start of the current rune) is correct.
				prefixEndRuneIndex = i
			}
			foundPrefixEnd = true
		}

		// Check if the start of the *next* rune is past our target end byte position
		if currentBytePos+runeLen > relativePosBytes+length {
			// Similar logic: if exact match, use current index, else use current index
			if currentBytePos == relativePosBytes+length {
				affixStartRuneIndex = i
			} else {
				affixStartRuneIndex = i
			}
			// Once we've found the affix start rune index, we can stop iterating
			// unless we haven't found the prefix end yet (can happen if length is 0)
			if foundPrefixEnd {
				break
			}
		}
		currentBytePos += runeLen
	}

	// Handle cases where pos or pos+length is at/after the end of the line text
	if !foundPrefixEnd {
		prefixEndRuneIndex = len(lineRunes)
	}
	// If affix start wasn't found within the loop, it means pos+length is at or past the end
	if affixStartRuneIndex == 0 && currentBytePos <= relativePosBytes+length {
		affixStartRuneIndex = len(lineRunes)
	}

	// Extract prefix and affix using rune indices
	if prefixEndRuneIndex > 0 && prefixEndRuneIndex <= len(lineRunes) {
		prefix = string(lineRunes[:prefixEndRuneIndex])
	} else {
		prefix = ""
	}

	if affixStartRuneIndex < len(lineRunes) && affixStartRuneIndex >= 0 {
		affix = string(lineRunes[affixStartRuneIndex:])
	} else {
		affix = ""
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
