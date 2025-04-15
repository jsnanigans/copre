package copre

import (
	"reflect"
	"sort"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Copied from position_mapping_test.go
func makeDiffsForPrediction(ops [][2]interface{}) []diffmatchpatch.Diff {
	var diffs []diffmatchpatch.Diff
	for _, op := range ops {
		diffType := op[0].(diffmatchpatch.Operation)
		text := op[1].(string)
		diffs = append(diffs, diffmatchpatch.Diff{Type: diffType, Text: text})
	}
	return diffs
}

func TestGeneratePredictions(t *testing.T) {
	dmp := diffmatchpatch.New()
	oldTextSimple := "delete me here and delete me there"
	newTextSimple := " here and  there" // removed "delete me" twice
	diffsSimple := dmp.DiffMain(oldTextSimple, newTextSimple, true)

	oldTextMapFail := "delete me here and delete you there"
	newTextMapFail := " here and delete you there" // only first removed
	diffsMapFail := dmp.DiffMain(oldTextMapFail, newTextMapFail, true)

	tests := []struct {
		name         string
		newText      string
		anchors      []Anchor
		charsRemoved string
		diffs        []diffmatchpatch.Diff
		wantPreds    []PredictedChange
	}{
		{
			name:         "No anchors",
			newText:      "abc",
			anchors:      []Anchor{},
			charsRemoved: "del",
			diffs:        makeDiffsForPrediction([][2]interface{}{{diffmatchpatch.DiffDelete, "del"}, {diffmatchpatch.DiffEqual, "abc"}}),
			wantPreds:    []PredictedChange{},
		},
		{
			name:         "Chars removed empty",
			newText:      "abc",
			anchors:      []Anchor{{Position: 0, Score: 5, Line: 1}},
			charsRemoved: "",
			diffs:        makeDiffsForPrediction([][2]interface{}{{diffmatchpatch.DiffEqual, "abc"}}),
			wantPreds:    []PredictedChange{},
		},
		{
			name:    "Simple valid prediction",
			newText: newTextSimple, // " here and  there"
			anchors: []Anchor{
				{Position: 20, Score: 7, Line: 1}, // "delete me" at pos 20 in oldText
			},
			charsRemoved: "delete me",
			diffs:        diffsSimple,
			wantPreds: []PredictedChange{
				{Position: 20, TextToRemove: "delete me", Line: 1, Score: 7, MappedPosition: 11}, // "delete me" at 0 in old -> maps to "" at 0 in new; "delete me" at 20 -> maps to " there" at 11 in new
			},
		},
		{
			name:    "Multiple predictions - Skipped case", // Renamed for clarity
			newText: "keepX keepY keepZ",                   // New text doesn't contain "remove"
			anchors: []Anchor{
				{Position: 7, Score: 6, Line: 1},
				{Position: 14, Score: 6, Line: 1},
			},
			charsRemoved: "remove", // Text that was originally removed
			diffs:        dmp.DiffMain("removeXremoveYremoveZ", "keepX keepY keepZ", true),
			wantPreds:    []PredictedChange{}, // Predictions are skipped because "remove" is not found at mapped positions in newText
		},
		{
			name:    "Multiple predictions - Matching case",
			newText: "removeX removeY removeZ", // New text *does* contain "remove"
			anchors: []Anchor{
				{Position: 7, Score: 6, Line: 1},  // Anchor in old: removeX|removeY|removeZ -> 'r' of removeY
				{Position: 14, Score: 6, Line: 1}, // Anchor in old: removeXremoveY|removeZ| -> 'r' of removeZ
			},
			wantPreds: []PredictedChange{
				{Position: 7, TextToRemove: "remove", Line: 1, Score: 6, MappedPosition: 8},   // Mapped to start of "removeY" in newText
				{Position: 14, TextToRemove: "remove", Line: 1, Score: 6, MappedPosition: 15}, // Mapped to start of "removeZ" in newText
			},
		},
		{
			name:    "Prediction skipped (text mismatch at mapped pos)",
			newText: newTextMapFail, // " here and delete you there"
			anchors: []Anchor{
				{Position: 20, Score: 7, Line: 1}, // "delete me" at pos 20 in oldTextMapFail ("delete me here and delete you there")
			},
			charsRemoved: "delete me", // This is what was removed originally
			diffs:        diffsMapFail,
			wantPreds:    []PredictedChange{}, // Should be skipped
		},
		{
			name:    "Prediction skipped (mapped position out of bounds)",
			newText: "abc", // Very short new text
			anchors: []Anchor{
				// Assume this anchor existed in a longer oldText and maps beyond "abc"
				{Position: 100, Score: 5, Line: 1},
			},
			charsRemoved: "xyz",
			// Diffs that would cause position 100 to map outside len("abc")
			diffs: makeDiffsForPrediction([][2]interface{}{
				{diffmatchpatch.DiffDelete, "long text ..."},
				{diffmatchpatch.DiffEqual, "abc"},
				{diffmatchpatch.DiffDelete, "... more text"}, // Map position 100 to > 3
			}),
			wantPreds: []PredictedChange{}, // Skipped because mapped position is invalid
		},
	}

	// Corrected Unicode Test Case Setup
	oldTextUnicode := "abc 世界 def 世界 ghi"
	newTextUnicode := "abc def 世界 ghi" // Removed first " 世界"
	diffsUnicode := dmp.DiffMain(oldTextUnicode, newTextUnicode, true)
	unicodeTest := struct { // Define outside the slice for easier setup
		name         string
		newText      string
		anchors      []Anchor
		charsRemoved string
		diffs        []diffmatchpatch.Diff
		wantPreds    []PredictedChange
	}{
		name:    "Unicode characters - Prediction Match",
		newText: newTextUnicode,
		anchors: []Anchor{
			{Position: 13, Score: 8, Line: 1}, // Byte position of the ' ' before the second 世界
		},
		charsRemoved: " 世界", // What was actually removed
		diffs:        diffsUnicode,
		wantPreds: []PredictedChange{
			// Anchor pos 13 maps to new pos 7. newText[7:] starts with " 世界"
			{Position: 13, TextToRemove: " 世界", Line: 1, Score: 8, MappedPosition: 7},
		},
	}
	tests = append(tests, unicodeTest) // Add the corrected test

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPreds := generatePredictions(tt.newText, tt.anchors, tt.charsRemoved, tt.diffs)
			// Sort predictions for stable comparison
			sort.Slice(gotPreds, func(i, j int) bool {
				if gotPreds[i].MappedPosition != gotPreds[j].MappedPosition {
					return gotPreds[i].MappedPosition < gotPreds[j].MappedPosition
				}
				return gotPreds[i].Position < gotPreds[j].Position
			})
			sort.Slice(tt.wantPreds, func(i, j int) bool {
				if tt.wantPreds[i].MappedPosition != tt.wantPreds[j].MappedPosition {
					return tt.wantPreds[i].MappedPosition < tt.wantPreds[j].MappedPosition
				}
				return tt.wantPreds[i].Position < tt.wantPreds[j].Position
			})

			if !reflect.DeepEqual(gotPreds, tt.wantPreds) {
				t.Errorf("generatePredictions() = %+v, want %+v", gotPreds, tt.wantPreds)
			}
		})
	}
}
