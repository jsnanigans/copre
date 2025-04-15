package copre

// PredictedChange represents a potential future edit.
type PredictedChange struct {
	Position     int    // Byte offset in oldText where the change originates
	TextToRemove string // The text to be removed
	// TextToAdd string // Future: Text to add (for insertions/replacements)
	Line           int // Line number in oldText where the change originates (1-based)
	Score          int // Confidence score for this prediction
	MappedPosition int // Corresponding byte offset in newText where the change should be applied
}

// Anchor represents a potential location for a predicted change in the old text.
type Anchor struct {
	Position int // Position in oldText
	Score    int
	Line     int // Line number in oldText
}
