package copre

import (
	"reflect"
	"sort"
	"testing"
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
		name      string
		oldText   string
		newText   string
		expected  []PredictedChange
		expectErr bool
	}{
		{
			name: "Simple text change - remove middle",
			oldText: `line 1
line 2 middle bit
line 3`,
			newText: `line 1
line 2
line 3`,
			expected:  []PredictedChange{},
			expectErr: false,
		},
		{
			name: "Simple text change - remove suffix (uses existing suffix test logic)",
			oldText: `line one-foo
line two-foo
line 3-foo`,
			newText: `line one-foo
line two
line 3-foo`,
			expected: []PredictedChange{
				{Position: 8, TextToRemove: "-foo", Line: 1, Score: 5, MappedPosition: 8},
				{Position: 30, TextToRemove: "-foo", Line: 3, Score: 5, MappedPosition: 24},
			},
			expectErr: false,
		},
		{
			name: "Add line",
			oldText: `line 1
line 3`,
			newText: `line 1
line 2
line 3`,
			expected:  []PredictedChange{},
			expectErr: false,
		},
		{
			name: "Remove line",
			oldText: `line 1
line 2 removed
line 3`,
			newText: `line 1
line 3`,
			expected:  []PredictedChange{},
			expectErr: false,
		},
		{
			name: "Remove multiple lines",
			oldText: `AAA
BBB
CCC
DDD
BBB
CCC
EEE`,
			newText: `AAA
DDD
BBB
CCC
EEE`,
			expected: []PredictedChange{
				{Position: 12, TextToRemove: "BBB\nCCC\n", Line: 5, Score: 5, MappedPosition: 8},
			},
			expectErr: false,
		},
		{
			name: "Remove text block with similar surrounding context",
			oldText: `keep start one
remove this 1
keep end one
--
keep start two
remove this 2
keep end two
--
keep start one
remove this 1
keep end one`,
			newText: `keep start one
keep end one
--
keep start two
remove this 2
keep end two
--
keep start one
remove this 1
keep end one`,
			expected: []PredictedChange{
				{Position: 88, TextToRemove: "remove this 1\n", Line: 9, Score: 7, MappedPosition: 74},
			},
			expectErr: false,
		},
		{
			name:      "Empty old text",
			oldText:   "",
			newText:   `line 1`,
			expected:  []PredictedChange{},
			expectErr: false,
		},
		{
			name:      "Empty new text",
			oldText:   `line 1`,
			newText:   "",
			expected:  []PredictedChange{},
			expectErr: false,
		},
		{
			name:      "Both empty",
			oldText:   "",
			newText:   "",
			expected:  []PredictedChange{},
			expectErr: false,
		},
		{
			name: "No change",
			oldText: `line 1
line 2`,
			newText: `line 1
line 2`,
			expected:  []PredictedChange{},
			expectErr: false,
		},
		{
			name: "Change at start of file",
			oldText: `REMOVE line 1
line 2
REMOVE line 3`,
			newText: `line 1
line 2
REMOVE line 3`,
			expected: []PredictedChange{
				{Position: 16, TextToRemove: "REMOVE ", Line: 3, Score: 5, MappedPosition: 16},
			},
			expectErr: false,
		},
		{
			name: "Change at end of file",
			oldText: `line 1 SUFFIX
line 2
line 3 SUFFIX`,
			newText: `line 1 SUFFIX
line 2
line 3`,
			expected: []PredictedChange{
				{Position: 6, TextToRemove: " SUFFIX", Line: 1, Score: 5, MappedPosition: 6},
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

			sortPredictions(got)
			sortPredictions(tt.expected)

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("PredictNextChanges() mismatch (-got +want):\nGot:  %+v\nWant: %+v", got, tt.expected)
			}
		})
	}
}

func TestPredictNextChanges_RemoveSuffix(t *testing.T) {
	oldText := `line one-smile
line two-smile
line 3-smile`
	newText := `line one-smile
line two
line 3-smile`

	expectedPredictions := []PredictedChange{
		{Position: 8, TextToRemove: "-smile", Line: 1, Score: 5, MappedPosition: 8},
		{Position: 36, TextToRemove: "-smile", Line: 3, Score: 5, MappedPosition: 30},
	}

	predictions, err := PredictNextChanges(oldText, newText)
	if err != nil {
		t.Fatalf("PredictNextChanges failed: %v", err)
	}

	if len(predictions) != len(expectedPredictions) {
		t.Errorf("Expected %d predictions, but got %d", len(expectedPredictions), len(predictions))
		t.Logf("Got predictions: %+v", predictions)
		return
	}

	foundCount := 0
	for _, expected := range expectedPredictions {
		found := false
		for _, actual := range predictions {
			if actual.Position == expected.Position &&
				actual.TextToRemove == expected.TextToRemove &&
				actual.Line == expected.Line &&
				actual.Score == expected.Score &&
				actual.MappedPosition == expected.MappedPosition {
				found = true
				break
			}
		}
		if found {
			foundCount++
		} else {
			t.Errorf("Expected prediction %+v not found", expected)
		}
	}

	if foundCount != len(expectedPredictions) {
		t.Errorf("Mismatch in predictions. Expected: %+v, Got: %+v", expectedPredictions, predictions)
	}
}
