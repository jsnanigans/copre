package main

import (
	"testing"
)

func TestProcessString(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedWords int
		expectedChars int
	}{
		{"Empty String", "", 0, 0},
		{"Single Word", "Hello", 1, 5},
		{"Multiple Words", "Hello World", 2, 11},
		{"Leading/Trailing Spaces", "  Spaces  ", 1, 10},
		{"Multiple Internal Spaces", "Word  with   spaces", 3, 19},
		{"Punctuation", "Go, programming! Fun.", 3, 21},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualWords, actualChars := processString(tc.input)
			if actualWords != tc.expectedWords {
				t.Errorf("processString(%q) word count = %d; want %d", tc.input, actualWords, tc.expectedWords)
			}
			if actualChars != tc.expectedChars {
				t.Errorf("processString(%q) char count = %d; want %d", tc.input, actualChars, tc.expectedChars)
			}
		})
	}
}
