package copre

import (
	"reflect"
	"sort"
	"testing"
)

// Helper function for sorting anchors to ensure stable comparison
func sortAnchors(anchors []Anchor) {
	sort.Slice(anchors, func(i, j int) bool { return anchors[i].Position < anchors[j].Position })
}

func TestFindAndScoreAnchors(t *testing.T) {
	tests := []struct {
		name                   string
		oldText                string
		searchText             string
		originalChangeStartPos int
		wantAnchors            []Anchor
	}{
		{
			name:                   "No anchors",
			oldText:                "abc def ghi",
			searchText:             "xyz",
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{},
		},
		{
			name: "One exact anchor with context mismatch",
			// Original change: remove "remove this" at pos 16. Context: prefix=" and ", affix=" too"
			// Anchor: "remove this" at pos 0. Context: prefix="", affix=" and "
			oldText:                "remove this and remove this too",
			searchText:             "remove this",
			originalChangeStartPos: 16, // The second "remove this"
			wantAnchors: []Anchor{
				{Position: 0, Score: 6, Line: 1},
			},
		},
		{
			name: "Multiple exact anchors with varying context match",
			// Original change: remove "A" at pos 0. Context: prefix="", affix=" "
			oldText:                "A A A A",
			searchText:             "A",
			originalChangeStartPos: 0, // First 'A'
			wantAnchors: []Anchor{
				{Position: 2, Score: 9, Line: 1},
				{Position: 4, Score: 7, Line: 1},
				{Position: 6, Score: 5, Line: 1},
			},
		},
		{
			name: "Matching prefix only",
			// Original change: remove "remove" at pos 20. Context: prefix="prefix ", affix=" too"
			// Anchor @ 7: Context prefix="prefix ", affix=" and "
			oldText:                "prefix remove and prefix remove too",
			searchText:             "remove",
			originalChangeStartPos: 25, // Second "remove"
			wantAnchors: []Anchor{
				{Position: 7, Score: 13, Line: 1},
			},
		},
		{
			name: "Matching affix only",
			// Original change: remove "remove" at pos 19. Context: prefix=" and ", affix=" affix too"
			// Anchor @ 0: Context prefix="", affix=" affix and "
			oldText:                "remove affix and remove affix too",
			searchText:             "remove",
			originalChangeStartPos: 17, // Second "remove"
			wantAnchors: []Anchor{
				{Position: 0, Score: 12, Line: 1},
			},
		},
		{
			name: "Matching prefix and affix",
			// Original change: remove "remove" at pos 27. Context: prefix="prefix ", affix=" affix too"
			// Anchor @ 7: Context prefix="prefix ", affix=" affix and "
			oldText:                "prefix remove affix and prefix remove affix too",
			searchText:             "remove",
			originalChangeStartPos: 31, // Second "remove"
			wantAnchors: []Anchor{
				{Position: 7, Score: 19, Line: 1},
			},
		},
		{
			name: "Score variation based on context length",
			// Original change: remove "remove" at pos 19. Context: prefix=" def", affix="xyz"
			oldText:                "abcremovexyz and defremovexyz",
			searchText:             "remove",
			originalChangeStartPos: 20, // Second "remove"
			wantAnchors: []Anchor{
				{Position: 3, Score: 8, Line: 1},
			},
		},
		{
			name:                   "Search text empty",
			oldText:                "abc",
			searchText:             "",
			originalChangeStartPos: 1,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Original pos -1 (invalid)",
			oldText:                "abc abc",
			searchText:             "abc",
			originalChangeStartPos: -1,
			wantAnchors:            []Anchor{}, // Should not search
		},
		{
			name: "Multi-line context match",
			// Original change: remove "remove this" at pos 23. Context: prefix="\n", affix=""
			oldText:                "line1\nremove this\nline3\nremove this",
			searchText:             "remove this",
			originalChangeStartPos: 24, // Start of second "remove this"
			wantAnchors: []Anchor{
				{Position: 6, Score: 5, Line: 2},
			},
		},
		{
			name:                   "Old text empty",
			oldText:                "",
			searchText:             "abc",
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{},
		},
		{
			name: "Original pos at end",
			// Original change: remove "end" at pos 11. Context: prefix=" ", affix=""
			oldText:                "start text end",
			searchText:             "end",
			originalChangeStartPos: 11,         // Position of 'e' in 'end'
			wantAnchors:            []Anchor{}, // No other instances of "end" to anchor to
		},
		{
			name:                   "Original pos out of bounds (positive)",
			oldText:                "abc abc",
			searchText:             "abc",
			originalChangeStartPos: 100,        // Out of bounds
			wantAnchors:            []Anchor{}, // Should not search
		},
		{
			name: "Unicode characters",
			// Original change: remove "โลก" at pos 12 (byte index). Context: prefix=" สวัสดี ", affix="!"
			// Anchor @ 0: Context: prefix="", affix=" สวัสดี "
			oldText:                "โลก สวัสดี โลก!", // "World Hello World!" in Thai
			searchText:             "โลก",             // "World"
			originalChangeStartPos: 29,                // Byte index of the second "โลก"
			wantAnchors: []Anchor{
				{Position: 0, Score: 5, Line: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAnchors := findAndScoreAnchors(tt.oldText, tt.searchText, tt.originalChangeStartPos)
			sortAnchors(gotAnchors)
			sortAnchors(tt.wantAnchors) // Sort expected anchors too for consistent comparison
			var bothEmpty = len(gotAnchors) == 0 && len(tt.wantAnchors) == 0
			if !bothEmpty && !reflect.DeepEqual(gotAnchors, tt.wantAnchors) {
				// Use %+v for detailed struct output
				t.Errorf("findAndScoreAnchors() mismatch:\n  Got: %+v\n Want: %+v", gotAnchors, tt.wantAnchors)
			}
		})
	}
}

func TestGetLocalContext(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		pos        int
		length     int
		wantPrefix string
		wantAffix  string
	}{
		{
			name:       "Basic mid-line",
			text:       "abc def ghi",
			pos:        4, // 'd'
			length:     3, // "def"
			wantPrefix: "abc ",
			wantAffix:  " ghi",
		},
		{
			name:       "Start of line",
			text:       "abc def ghi",
			pos:        0, // 'a'
			length:     3, // "abc"
			wantPrefix: "",
			wantAffix:  " def ghi",
		},
		{
			name:       "End of line",
			text:       "abc def ghi",
			pos:        8, // 'g'
			length:     3, // "ghi"
			wantPrefix: "abc def ",
			wantAffix:  "",
		},
		{
			name:       "Start of file",
			text:       "line1\nline2",
			pos:        0, // 'l' of line1
			length:     5, // "line1"
			wantPrefix: "",
			wantAffix:  "", // Context is only within the same line
		},
		{
			name:       "End of file",
			text:       "line1\nline2",
			pos:        6, // 'l' of line2
			length:     5, // "line2"
			wantPrefix: "",
			wantAffix:  "", // Context is only within the same line
		},
		{
			name:       "Middle of second line",
			text:       "line1\nabc def ghi\nline3",
			pos:        10, // 'd' in "def" on line 2
			length:     3,  // "def"
			wantPrefix: "abc ",
			wantAffix:  " ghi",
		},
		{
			name:       "Start of second line",
			text:       "line1\nabc def ghi\nline3",
			pos:        6,  // 'a' in "abc" on line 2
			length:     3,  // "abc"
			wantPrefix: "", // Start of line 2
			wantAffix:  " def ghi",
		},
		{
			name:       "End of second line",
			text:       "line1\nabc def ghi\nline3",
			pos:        14, // 'g' in "ghi" on line 2
			length:     3,  // "ghi"
			wantPrefix: "abc def ",
			wantAffix:  "", // End of line 2
		},
		{
			name:       "Empty text",
			text:       "",
			pos:        0,
			length:     0,
			wantPrefix: "",
			wantAffix:  "",
		},
		{
			name:       "Text with only newline",
			text:       "\n",
			pos:        0,
			length:     0,
			wantPrefix: "",
			wantAffix:  "",
		},
		{
			name:       "Position at end of text (after content)",
			text:       "abc",
			pos:        3,
			length:     0,
			wantPrefix: "abc",
			wantAffix:  "",
		},
		{
			name:       "Length extends beyond text",
			text:       "abc def",
			pos:        4,
			length:     10, // "def" plus more
			wantPrefix: "abc ",
			wantAffix:  "",
		},
		{
			name:       "Position + Length exactly at end of line",
			text:       "line1\nline2",
			pos:        6, // 'l' of line2
			length:     5, // "line2"
			wantPrefix: "",
			wantAffix:  "",
		},
		{
			name:       "Position + Length exactly at end of file",
			text:       "abc",
			pos:        0, // 'a'
			length:     3, // "abc"
			wantPrefix: "",
			wantAffix:  "",
		},
		{
			name:       "Unicode characters",
			text:       "สวัสดี โลก!", // Hello World! in Thai
			pos:        19,            // Byte index of 'โ' in "โลก"
			length:     9,             // Byte length of "โลก"
			wantPrefix: "สวัสดี ",     // Note the space
			wantAffix:  "!",
		},
		{
			name:       "Unicode at start of line",
			text:       "โลก สวัสดี",
			pos:        0,
			length:     9, // "โลก"
			wantPrefix: "",
			wantAffix:  " สวัสดี",
		},
		{
			name:       "Unicode at end of line",
			text:       "สวัสดี โลก",
			pos:        19, // 'โ'
			length:     9,  // "โลก"
			wantPrefix: "สวัสดี ",
			wantAffix:  "",
		},
		{
			name:       "Multiple newlines",
			text:       "line1\n\nline3",
			pos:        7, // 'l' of line3
			length:     5, // "line3"
			wantPrefix: "",
			wantAffix:  "",
		},
		{
			name:       "Position on an empty line between lines",
			text:       "line1\n\nline3",
			pos:        6, // The second newline character itself
			length:     0,
			wantPrefix: "",
			wantAffix:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPrefix, gotAffix := getLocalContext(tt.text, tt.pos, tt.length)
			if gotPrefix != tt.wantPrefix {
				t.Errorf("getLocalContext() gotPrefix = %q, want %q", gotPrefix, tt.wantPrefix)
			}
			if gotAffix != tt.wantAffix {
				t.Errorf("getLocalContext() gotAffix = %q, want %q", gotAffix, tt.wantAffix)
			}
		})
	}
}
