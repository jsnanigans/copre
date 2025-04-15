package copre

import (
	"reflect"
	"sort"
	"testing"
	// "github.com/sergi/go-diff/diffmatchpatch"
	// "fmt"
)

// Helper function to sort predictions for stable comparison
func sortPredictions(predictions []PredictedChange) {
	sort.Slice(predictions, func(i, j int) bool {
		// Primary sort by MappedPosition, secondary by Position for tie-breaking
		if predictions[i].MappedPosition != predictions[j].MappedPosition {
			return predictions[i].MappedPosition < predictions[j].MappedPosition
		}
		return predictions[i].Position < predictions[j].Position
	})
}

func TestPredictNextChanges(t *testing.T) {
	tests := []struct {
		// Tests removing text from the middle of a line.
		name      string
		oldText   string
		newText   string
		expected  []PredictedChange
		expectErr bool
	}{
		{
			name: "Simple text change - remove middle",
			oldText: "line 1\n" +
				"line 2 middle bit\n" +
				"line 3",
			newText: "line 1\n" +
				"line 2\n" +
				"line 3",
			expected:  nil,
			expectErr: false,
		},
		{
			// Tests removing a suffix from one line when similar lines exist.
			name: "Simple text change - remove suffix (uses existing suffix test logic)",
			oldText: "line one-foo\n" +
				"line two-foo\n" +
				"line 3-foo",
			newText: "line one-foo\n" +
				"line two\n" +
				"line 3-foo",
			expected: []PredictedChange{
				{Position: 8, TextToRemove: "-foo", Line: 1, Score: 9, MappedPosition: 8},
				{Position: 32, TextToRemove: "-foo", Line: 3, Score: 9, MappedPosition: 28},
			},
			expectErr: false,
		},
		{
			// Tests adding a new line between existing lines.
			name: "Add line",
			oldText: "line 1\n" +
				"line 3",
			newText: "line 1\n" +
				"line 2\n" +
				"line 3",
			expected:  nil,
			expectErr: false,
		},
		{
			// Tests removing a single line.
			name: "Remove line",
			oldText: "line 1\n" +
				"line 2 removed\n" +
				"line 3",
			newText: "line 1\n" +
				"line 3",
			expected:  nil,
			expectErr: false,
		},
		{
			// Tests removing a block of consecutive lines.
			name: "Remove multiple lines",
			oldText: "AAA\n" +
				"BBB\n" +
				"CCC\n" +
				"DDD\n" +
				"BBB\n" +
				"CCC\n" +
				"EEE",
			newText: "AAA\n" +
				"DDD\n" +
				"BBB\n" +
				"CCC\n" +
				"EEE",
			expected: []PredictedChange{
				{Position: 16, TextToRemove: "BBB\nCCC\n", Line: 5, Score: 12, MappedPosition: 8},
			},
			expectErr: false,
		},
		{
			// Tests removing a block of text where identical blocks exist elsewhere in the file.
			name: "Remove text block with similar surrounding context",
			oldText: "keep start one\n" +
				"remove this 1\n" +
				"keep end one\n" +
				"--\n" +
				"keep start two\n" +
				"remove this 2\n" +
				"keep end two\n" +
				"--\n" +
				"keep start one\n" +
				"remove this 1\n" +
				"keep end one",
			newText: "keep start one\n" +
				"keep end one\n" +
				"--\n" +
				"keep start two\n" +
				"remove this 2\n" +
				"keep end two\n" +
				"--\n" +
				"keep start one\n" +
				"remove this 1\n" +
				"keep end one",
			expected: []PredictedChange{
				{Position: 105, TextToRemove: "remove this 1\n", Line: 10, Score: 32, MappedPosition: 91},
			},
			expectErr: false,
		},
		{
			// Tests the case where the original text is empty.
			name:      "Empty old text",
			oldText:   "",
			newText:   "line 1",
			expected:  nil,
			expectErr: false,
		},
		{
			// Tests the case where the resulting text is empty (effectively deleting all content).
			name:      "Empty new text",
			oldText:   "line 1",
			newText:   "",
			expected:  nil,
			expectErr: false,
		},
		{
			// Tests the case where both old and new texts are empty.
			name:      "Both empty",
			oldText:   "",
			newText:   "",
			expected:  nil,
			expectErr: false,
		},
		{
			// Tests the case where old and new texts are identical.
			name: "No change",
			oldText: "line 1\n" +
				"line 2",
			newText: "line 1\n" +
				"line 2",
			expected:  nil,
			expectErr: false,
		},
		{
			// Tests removing text from the beginning of a line that also occurs later in the file.
			name: "Change at start of file",
			oldText: "REMOVE line 1\n" +
				"line 2\n" +
				"REMOVE line 3",
			newText: "line 1\n" +
				"line 2\n" +
				"REMOVE line 3",
			expected: []PredictedChange{
				{Position: 21, TextToRemove: "REMOVE ", Line: 3, Score: 11, MappedPosition: 14},
			},
			expectErr: false,
		},
		{
			// Tests removing text from the end of a line that also occurs earlier in the file.
			name: "Change at end of file",
			oldText: "line 1 SUFFIX\n" +
				"line 2\n" +
				"line 3 SUFFIX",
			newText: "line 1 SUFFIX\n" +
				"line 2\n" +
				"line 3",
			expected: []PredictedChange{
				{Position: 6, TextToRemove: " SUFFIX", Line: 1, Score: 11, MappedPosition: 6},
			},
			expectErr: false,
		},
		{
			name: "No repeating pattern",
			oldText: "delete ABC\n" +
				"keep DEF",
			newText:   "keep DEF",          // Deleted "delete ABC\n"
			expected:  []PredictedChange{}, // "delete ABC\n" doesn't repeat
			expectErr: false,
		},
		{
			name:      "Whitespace only change (indentation)",
			oldText:   "line1\nline2",
			newText:   "line1\n  line2", // Indented line 2
			expected:  nil,              // No deletion, so no prediction
			expectErr: false,
		},
		{
			name: "Replacement change",
			oldText: "replace OLD with new\n" +
				"line 2\n" +
				"replace OLD with new",
			newText: "replace NEW with new\n" +
				"line 2\n" +
				"replace OLD with new",
			// Diff will see "OLD" deleted at pos 8
			expected: []PredictedChange{
				// Anchor found at pos 30. Context prefix="replace ", affix=" with new"
				// Score: 5 (base) + 8 (prefix) + 9 (affix) = 22
				{Position: 30, TextToRemove: "OLD", Line: 3, Score: 22, MappedPosition: 30},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PredictNextChanges(tt.oldText, tt.newText)

			if (err != nil) != tt.expectErr {
				t.Fatalf("PredictNextChanges() error = %v, expectErr %v", err, tt.expectErr)
			}
			if tt.expectErr {
				return
			}

			sortPredictions(got) // Use the local helper
			sortPredictions(tt.expected)

			// Convert expected predictions for comparison if needed (if wantPreds was used)
			var wantComparable []PredictedChange
			wantComparable = tt.expected // Directly use the expected field now

			if !reflect.DeepEqual(got, wantComparable) {
				t.Errorf("PredictNextChanges() mismatch (-got +want):\nGot:  %+v\nWant: %+v", got, wantComparable)
			}
		})
	}
}

// TestPredictNextChanges_RemoveSuffix removed - covered by main table test

// TestMapPosition moved to position_mapping_test.go

// TestAnalyzeDiffs moved to diff_analysis_test.go

// TestFindAndScoreAnchors moved to anchoring_test.go

// TestGeneratePredictions moved to prediction_test.go

// TestVisualizePredictions moved to visualization_test.go
