package copre

import (
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Helper function to create diffs for testing mapPosition
func makeDiffsForPositionMapping(ops [][2]interface{}) []diffmatchpatch.Diff {
	var diffs []diffmatchpatch.Diff
	for _, op := range ops {
		diffType := op[0].(diffmatchpatch.Operation)
		text := op[1].(string)
		diffs = append(diffs, diffmatchpatch.Diff{Type: diffType, Text: text})
	}
	return diffs
}

func TestMapPosition(t *testing.T) {
	tests := []struct {
		name       string
		oldPos     int
		diffs      []diffmatchpatch.Diff
		wantNewPos int
	}{
		{
			name:       "No diffs",
			oldPos:     5,
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffEqual, "abcdef"}}),
			wantNewPos: 5,
		},
		{
			name:       "Simple insert at start",
			oldPos:     3,
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffInsert, "XYZ"}, {diffmatchpatch.DiffEqual, "abc"}}),
			wantNewPos: 3 + 3, // 3 for "XYZ" + original 3
		},
		{
			name:       "Simple delete at start",
			oldPos:     5, // Position 'e' in "abcde"
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffDelete, "ab"}, {diffmatchpatch.DiffEqual, "cde"}}),
			wantNewPos: 3, // 5 - 2 = 3 ('e' is now at index 3 in "cde")
		},
		{
			name:       "Insert in middle",
			oldPos:     5, // Position 'e' in "abcde"
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffEqual, "abc"}, {diffmatchpatch.DiffInsert, "XYZ"}, {diffmatchpatch.DiffEqual, "de"}}),
			wantNewPos: 5 + 3, // original 5 + length of "XYZ"
		},
		{
			name:       "Delete in middle",
			oldPos:     7, // Position 'g' in "abcdefgh"
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffEqual, "abc"}, {diffmatchpatch.DiffDelete, "de"}, {diffmatchpatch.DiffEqual, "fgh"}}),
			wantNewPos: 5, // 7 - length of "de" (2)
		},
		{
			name:       "Delete includes oldPos",
			oldPos:     4, // Position 'e' in "abcdefgh"
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffEqual, "abc"}, {diffmatchpatch.DiffDelete, "de"}, {diffmatchpatch.DiffEqual, "fgh"}}),
			wantNewPos: 3, // Maps to the position *before* the deletion in new text ("abc" -> index 3)
		},
		{
			name:   "Mixed diffs",
			oldPos: 10, // Position 'k' in "abcdefghijkl"
			diffs: makeDiffsForPositionMapping([][2]interface{}{
				{diffmatchpatch.DiffEqual, "abc"},
				{diffmatchpatch.DiffDelete, "de"},  // oldPos moves from 10 to 8
				{diffmatchpatch.DiffInsert, "XYZ"}, // newPos shifts by 3
				{diffmatchpatch.DiffEqual, "fghijk"},
				{diffmatchpatch.DiffDelete, "l"}, // 'k' is before this, no effect
			}),
			wantNewPos: 8 + 3, // 10 - 2 ("de") + 3 ("XYZ") = 11
		},
		{
			name:       "Position at end of old text after delete",
			oldPos:     5, // End of "abcde"
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffEqual, "abc"}, {diffmatchpatch.DiffDelete, "de"}}),
			wantNewPos: 3, // End of "abc"
		},
		{
			name:       "Position at end of old text after insert",
			oldPos:     3, // End of "abc"
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffEqual, "abc"}, {diffmatchpatch.DiffInsert, "XYZ"}}),
			wantNewPos: 3 + 3, // End of "abcXYZ"
		},
		{
			name:       "Old position is 0",
			oldPos:     0,
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffInsert, "XYZ"}, {diffmatchpatch.DiffEqual, "abc"}}),
			wantNewPos: 3, // Position 0 maps to 3 after insertion
		},
		{
			name:       "Old position is 0 with delete",
			oldPos:     0,
			diffs:      makeDiffsForPositionMapping([][2]interface{}{{diffmatchpatch.DiffDelete, "ab"}, {diffmatchpatch.DiffEqual, "cde"}}),
			wantNewPos: 0, // Position 0 is within deletion, maps to start of new text
		},
		{
			name:   "Unicode characters",
			oldPos: 10, // Byte position of second '界' in "abc 世界 def 世界 ghi"
			diffs: makeDiffsForPositionMapping([][2]interface{}{
				{diffmatchpatch.DiffEqual, "abc "},       // 4 bytes
				{diffmatchpatch.DiffDelete, "世界 "},       // 7 bytes deleted (世界 is 6 bytes + space)
				{diffmatchpatch.DiffEqual, "def 世界 ghi"}, // Remaining text
			}),
			// Original: abc 世界 def 世界 ghi
			//               ^        ^
			// bytes:      4       11
			// oldPos 10 is the second byte of the first 世界
			// Since it's in the deleted part, it maps to the end of the preceding equal part.
			wantNewPos: 4, // Maps to end of "abc "
		},
		{
			name:   "Position beyond original text length implied by diffs",
			oldPos: 100, // Way past the end
			diffs: makeDiffsForPositionMapping([][2]interface{}{
				{diffmatchpatch.DiffEqual, "abc"},
				{diffmatchpatch.DiffInsert, "XYZ"},
				{diffmatchpatch.DiffEqual, "def"},
			}),
			// New text is "abcXYZdef", length 9
			wantNewPos: 9, // Should map to the end of the new text
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewPos := mapPosition(tt.oldPos, tt.diffs)
			if gotNewPos != tt.wantNewPos {
				t.Errorf("mapPosition(%d, ...) = %d; want %d", tt.oldPos, gotNewPos, tt.wantNewPos)
			}
		})
	}
}
