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
