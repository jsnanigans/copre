package main

import (
	"fmt"
	"strings"
)

// processString counts the words and characters in a given string.
// It returns the word count and the character count (length).
func processString(input string) (int, int) {
	words := strings.Fields(input)
	wordCount := len(words)
	charCount := len(input)
	return wordCount, charCount
}

func main() {
	inputText := "This is a sample string."
	wordCount, charCount := processString(inputText)

	fmt.Printf("Input string: \"%s\"\n", inputText)
	fmt.Printf("Word count: %d\n", wordCount)
	fmt.Printf("Character count: %d\n", charCount)
}
