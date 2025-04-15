package copre

import (
	"fmt"
	"testing"
)

func TestVisualizePredictions(t *testing.T) {
	// Helper to create a simple prediction
	makePred := func(text string, mappedPos int) PredictedChange {
		return PredictedChange{TextToRemove: text, MappedPosition: mappedPos}
	}

	// ANSI codes defined in visualization.go
	red := "\033[31m"
	reset := "\033[0m"

	tests := []struct {
		name        string
		text        string
		predictions []PredictedChange
		want        string
	}{
		{
			name:        "No predictions",
			text:        "hello world",
			predictions: []PredictedChange{},
			want:        "hello world",
		},
		{
			name:        "One prediction at start",
			text:        "delete me",
			predictions: []PredictedChange{makePred("delete ", 0)},
			want:        fmt.Sprintf("%sdelete %sme", red, reset),
		},
		{
			name:        "One prediction in middle",
			text:        "hello delete world",
			predictions: []PredictedChange{makePred("delete ", 6)},
			want:        fmt.Sprintf("hello %sdelete %sworld", red, reset),
		},
		{
			name:        "One prediction at end",
			text:        "hello delete",
			predictions: []PredictedChange{makePred(" delete", 5)},
			want:        fmt.Sprintf("hello%s delete%s", red, reset),
		},
		{
			name: "Multiple non-overlapping",
			text: "del A del B del C",
			predictions: []PredictedChange{
				makePred("del ", 0),
				makePred("del ", 6),
				makePred("del ", 12),
			},
			want: fmt.Sprintf("%sdel %sA %sdel %sB %sdel %sC", red, reset, red, reset, red, reset),
		},
		{
			name: "Overlapping predictions (handled gracefully)",
			text: "delete delete overlap",
			// Predictions should be sorted by MappedPosition by VisualizePredictions
			predictions: []PredictedChange{
				makePred("delete ", 0),
				makePred("delete ", 3), // Overlaps with first
			},
			// Expect only the first prediction to be applied correctly due to sorting and lastPos update
			want: fmt.Sprintf("%sdelete %sdelete overlap", red, reset),
		},
		{
			name: "Out of order predictions (handled gracefully)",
			text: "del A del B",
			predictions: []PredictedChange{
				makePred("del ", 6), // Second one first
				makePred("del ", 0), // First one second
			},
			// VisualizePredictions sorts them first
			want: fmt.Sprintf("%sdel %sA %sdel %sB", red, reset, red, reset),
		},
		{
			name: "Prediction end out of bounds",
			text: "hello",
			predictions: []PredictedChange{
				makePred("hello world", 0), // TextToRemove is longer than text
			},
			want: "hello", // Prediction should be skipped
		},
		{
			name: "Prediction start out of bounds",
			text: "hello",
			predictions: []PredictedChange{
				makePred("o", 10), // MappedPosition > len(text)
			},
			want: "hello", // Prediction should be skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: VisualizePredictions modifies the order of the slice it receives due to sorting.
			// Make a copy if the original order needs preservation outside this test.
			predsCopy := make([]PredictedChange, len(tt.predictions))
			copy(predsCopy, tt.predictions)

			if got := VisualizePredictions(tt.text, predsCopy); got != tt.want {
				// Use %q for clearer output showing control characters
				t.Errorf("VisualizePredictions() = %q, want %q", got, tt.want)
			}
		})
	}
}
