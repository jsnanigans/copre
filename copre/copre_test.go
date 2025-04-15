package copre

import (
	"testing"
)

func TestPredictNextChanges(t *testing.T) {
	tests := []struct {
		name      string
		oldText   string
		newText   string
		expected  string
		expectErr bool
	}{
		{
			name: "Simple text change",
			oldText: `line 1
line 2
line 3`,
			newText: `line 1
line two
line 3`,
			expected:  "___not_implemented___",
			expectErr: false,
		},
		{
			name: "Add line",
			oldText: `line 1
line 3`,
			newText: `line 1
line 2
line 3`,
			expected:  "___not_implemented___",
			expectErr: false,
		},
		{
			name: "Remove line",
			oldText: `line 1
line 2
line 3`,
			newText: `line 1
line 3`,
			expected:  "___not_implemented___",
			expectErr: false,
		},
		{
			name: "Go code change",
			oldText: `package main

func main() {
	fmt.Println("Hello")
}`,
			newText: `package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}`,
			expected:  "___not_implemented___",
			expectErr: false,
		},
		{
			name: "Python code change",
			oldText: `def greet(name):
    print(f"Hello, {name}")`,
			newText: `def greet(name):
    greeting = f"Hello, {name}!"
    print(greeting)`,
			expected:  "___not_implemented___",
			expectErr: false,
		},
		{
			name: "JavaScript code change",
			oldText: `function add(a, b) {
  return a + b;
}`,
			newText: `const add = (a, b) => {
  console.log("Adding:", a, b);
  return a + b;
};`,
			expected:  "___not_implemented___",
			expectErr: false,
		},
		{
			name:    "Empty old text",
			oldText: "",
			newText: `line 1
line 2`,
			expected:  "___not_implemented___",
			expectErr: false,
		},
		{
			name: "Empty new text",
			oldText: `line 1
line 2`,
			newText:   "",
			expected:  "___not_implemented___",
			expectErr: false,
		},
		{
			name:      "Both empty",
			oldText:   "",
			newText:   "",
			expected:  "___not_implemented___",
			expectErr: false,
		},
		{
			name: "No change",
			oldText: `line 1
line 2`,
			newText: `line 1
line 2`,
			expected:  "___not_implemented___",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PredictNextChanges(tt.oldText, tt.newText)

			if (err != nil) != tt.expectErr {
				t.Fatalf("PredictNextChanges() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !tt.expectErr && got != tt.expected {
				t.Errorf("PredictNextChanges() = %v, want %v", got, tt.expected)
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
		{Position: 8, TextToRemove: "-smile", Line: 1, Score: 5, MappedPosition: 8},   // Prediction on line 1
		{Position: 36, TextToRemove: "-smile", Line: 3, Score: 5, MappedPosition: 30}, // Prediction on line 3
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

	// Basic check: Ensure all expected predictions are found, regardless of order.
	// A more robust check would involve sorting or using a map.
	foundCount := 0
	for _, expected := range expectedPredictions {
		found := false
		for _, actual := range predictions {
			// Compare relevant fields. Position is oldText, MappedPosition is newText.
			if actual.Position == expected.Position &&
				actual.TextToRemove == expected.TextToRemove &&
				actual.Line == expected.Line &&
				actual.Score == expected.Score && // Score might be heuristic, adjust comparison if needed
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
