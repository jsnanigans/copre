package copre

import (
	"reflect"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Helper function for sorting anchors to ensure stable comparison
func sortAnchors(anchors []Anchor) {
	sort.Slice(anchors, func(i, j int) bool {
		if anchors[i].Position != anchors[j].Position {
			return anchors[i].Position < anchors[j].Position
		}
		return anchors[i].Score < anchors[j].Score // Secondary sort by score if positions are equal
	})
}

func TestFindAndScoreAnchors(t *testing.T) {
	tests := []struct {
		name                   string
		oldText                string
		searchText             string
		charsAdded             string
		originalChangeStartPos int
		wantAnchors            []Anchor
	}{
		{
			name:                   "No anchors",
			oldText:                "abc def ghi",
			searchText:             "xyz",
			charsAdded:             "",
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{},
		},
		{
			name: "One exact anchor with context mismatch",
			// Original change: remove "remove this" at pos 16. Context: prefix=" and ", affix=" too"
			// Anchor: "remove this" at pos 0. Context: prefix="", affix=" and "
			oldText:                "remove this and remove this too",
			searchText:             "remove this",
			charsAdded:             "",
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
			charsAdded:             "",
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
			charsAdded:             "",
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
			charsAdded:             "",
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
			charsAdded:             "",
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
			charsAdded:             "",
			originalChangeStartPos: 20, // Second "remove"
			wantAnchors: []Anchor{
				{Position: 3, Score: 8, Line: 1},
			},
		},
		{
			name:                   "Search text empty",
			oldText:                "abc",
			searchText:             "",
			charsAdded:             "",
			originalChangeStartPos: 1,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Original pos -1 (invalid)",
			oldText:                "abc abc",
			searchText:             "abc",
			charsAdded:             "",
			originalChangeStartPos: -1,
			wantAnchors:            []Anchor{}, // Should not search
		},
		{
			name: "Multi-line context match",
			// Original change: remove "remove this" at pos 23. Context: prefix="\n", affix=""
			oldText:                "line1\nremove this\nline3\nremove this",
			searchText:             "remove this",
			charsAdded:             "",
			originalChangeStartPos: 24, // Start of second "remove this"
			wantAnchors: []Anchor{
				{Position: 6, Score: 5, Line: 2},
			},
		},
		{
			name:                   "Old text empty",
			oldText:                "",
			searchText:             "abc",
			charsAdded:             "",
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{},
		},
		{
			name: "Original pos at end",
			// Original change: remove "end" at pos 11. Context: prefix=" ", affix=""
			oldText:                "start text end",
			searchText:             "end",
			charsAdded:             "",
			originalChangeStartPos: 11,         // Position of 'e' in 'end'
			wantAnchors:            []Anchor{}, // No other instances of "end" to anchor to
		},
		{
			name:                   "Original pos out of bounds (positive)",
			oldText:                "abc abc",
			searchText:             "abc",
			charsAdded:             "",
			originalChangeStartPos: 100,        // Out of bounds
			wantAnchors:            []Anchor{}, // Should not search
		},
		{
			name: "Unicode characters",
			// Original change: remove "โลก" at pos 12 (byte index). Context: prefix=" สวัสดี ", affix="!"
			// Anchor @ 0: Context: prefix="", affix=" สวัสดี "
			oldText:                "โลก สวัสดี โลก!", // "World Hello World!" in Thai
			searchText:             "โลก",             // "World"
			charsAdded:             "",
			originalChangeStartPos: 29, // Byte index of the second "โลก"
			wantAnchors: []Anchor{
				{Position: 0, Score: 5, Line: 1},
			},
		},
		{
			name:                   "Basic deletion match",
			oldText:                "delete me here\ndelete me there",
			searchText:             "delete me ", // This represents charsRemoved
			charsAdded:             "",           // ADDED
			originalChangeStartPos: 0,
			wantAnchors: []Anchor{
				{Position: 0, Score: 12, Line: 1},
				{Position: 12, Score: 12, Line: 2},
			},
		},
		{
			name:                   "Deletion match with context",
			oldText:                "prefix delete me here suffix\nprefix delete me there suffix",
			searchText:             "delete me ", // charsRemoved
			charsAdded:             "",           // ADDED
			originalChangeStartPos: 7,            // After "prefix "
			wantAnchors: []Anchor{
				{Position: 7, Score: 12, Line: 1},
				{Position: 19, Score: 12, Line: 2},
			},
		},
		{
			name:                   "No match",
			oldText:                "nothing to find",
			searchText:             "delete me", // charsRemoved
			charsAdded:             "",          // ADDED
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Match at original position only",
			oldText:                "only match here",
			searchText:             "match here", // charsRemoved
			charsAdded:             "",           // ADDED
			originalChangeStartPos: 5,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Empty search text",
			oldText:                "some text",
			searchText:             "", // charsRemoved
			charsAdded:             "", // ADDED
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Invalid original position",
			oldText:                "text",
			searchText:             "t", // charsRemoved
			charsAdded:             "",  // ADDED
			originalChangeStartPos: -1,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Unicode text matching",
			oldText:                "你好 世界\n你好 中国",
			searchText:             "你好 ", // charsRemoved
			charsAdded:             "",    // ADDED
			originalChangeStartPos: 0,
			wantAnchors: []Anchor{
				{Position: 0, Score: 5, Line: 1},
				{Position: 5, Score: 5, Line: 2},
			},
		},
		{
			name:                   "Context scoring partial prefix/affix",
			oldText:                "abc delete me 123\nxyz delete me 456",
			searchText:             "delete me ", // charsRemoved
			charsAdded:             "",           // ADDED
			originalChangeStartPos: 4,            // After "abc "
			wantAnchors: []Anchor{
				{Position: 4, Score: 8, Line: 1},
				{Position: 12, Score: 8, Line: 2},
			},
		},
		{
			name:                   "Context across lines - no match expected",
			oldText:                "delete me\nhere\ndelete me\nthere",
			searchText:             "delete me\nhere", // charsRemoved
			charsAdded:             "",                // ADDED
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{}, // Local context prevents matching 'delete me' on line 3
		},
		{
			name:                   "Pure insertion - no anchors expected (currently)",
			oldText:                "line1\nline3",
			searchText:             "",         // Represents charsRemoved
			charsAdded:             "\nline2",  // ADDED - This is what was inserted
			originalChangeStartPos: 5,          // Position where insertion happened
			wantAnchors:            []Anchor{}, // Current logic doesn't find anchors based on added text
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAnchors := findAndScoreAnchors(tt.oldText, tt.charsAdded, tt.searchText /*charsRemoved*/, tt.originalChangeStartPos)
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
			name:       "Basic middle",
			text:       "prefix target suffix",
			pos:        7, // start of 'target'
			length:     6, // length of 'target'
			wantPrefix: "prefix ",
			wantAffix:  " suffix",
		},
		{
			name:       "Start of text",
			text:       "target suffix",
			pos:        0, // start of 'target'
			length:     6, // length of 'target'
			wantPrefix: "",
			wantAffix:  " suffix",
		},
		{
			name:       "End of text",
			text:       "prefix target",
			pos:        7, // start of 'target'
			length:     6, // length of 'target'
			wantPrefix: "prefix ",
			wantAffix:  "",
		},
		{
			name:       "Entire text",
			text:       "target",
			pos:        0,
			length:     6,
			wantPrefix: "",
			wantAffix:  "",
		},
		{
			name:       "Middle of multi-line",
			text:       "line1\nprefix target suffix\nline3",
			pos:        13, // start of 'target'
			length:     6,
			wantPrefix: "prefix ", // Context only on the same line
			wantAffix:  " suffix",
		},
		{
			name:       "Start of line (not text)",
			text:       "line1\ntarget suffix\nline3",
			pos:        6, // start of 'target'
			length:     6,
			wantPrefix: "",
			wantAffix:  " suffix",
		},
		{
			name:       "End of line (not text)",
			text:       "line1\nprefix target\nline3",
			pos:        13, // start of 'target'
			length:     6,
			wantPrefix: "prefix ",
			wantAffix:  "",
		},
		{
			name:       "Target spans across lines (not supported by design)",
			text:       "line1 target\nline2 suffix",
			pos:        6,        // start of 'target'
			length:     12,       // 'target\nline2'
			wantPrefix: "line1 ", // Prefix only until end of line 1
			wantAffix:  "",       // Affix starts after line break, so it's empty for line 1
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
			name:       "Zero length target",
			text:       "prefixsuffix",
			pos:        6, // between prefix and suffix
			length:     0,
			wantPrefix: "prefix",
			wantAffix:  "suffix",
		},
		{
			name:       "Unicode basic",
			text:       "你好 世界 再见", // Hello World Goodbye
			pos:        7,          // start of '世界' (World)
			length:     6,          // byte length of '世界'
			wantPrefix: "你好 ",
			wantAffix:  " 再见",
		},
		{
			name:       "Unicode zero length",
			text:       "你好世界", // HelloWorld
			pos:        6,      // between 你好 and 世界
			length:     0,
			wantPrefix: "你好",
			wantAffix:  "世界",
		},
		{
			name:       "Unicode start of line",
			text:       "line1\n世界 再见", // line1\nWorld Goodbye
			pos:        6,              // start of 世界
			length:     6,              // byte length of 世界
			wantPrefix: "",
			wantAffix:  " 再见",
		},
		{
			name:       "Unicode end of line",
			text:       "line1\n你好 世界", // line1\nHello World
			pos:        12,             // start of 世界
			length:     6,              // byte length of 世界
			wantPrefix: "你好 ",
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

func TestFindAndScoreAnchors_LocalContext(t *testing.T) {
	tests := []struct {
		name                   string
		oldText                string
		searchText             string // Represents charsRemoved for anchor finding
		charsAdded             string // ADDED: Represents charsAdded, new field
		originalChangeStartPos int
		wantAnchors            []Anchor // NOTE: Scores are approximate based on logic
	}{
		{
			name:                   "Basic deletion match",
			oldText:                "delete me here\ndelete me there",
			searchText:             "delete me ",
			charsAdded:             "", // ADDED field value
			originalChangeStartPos: 0,  // Assume first one was deleted
			wantAnchors: []Anchor{
				// Original context: prefix="", affix="here"
				// Anchor context: prefix="", affix="there"
				{Position: 15, Score: 5 /*base*/ + 0 /*prefix*/ + 0 /*affix*/, Line: 2},
			},
		},
		{
			name:                   "Deletion match with context",
			oldText:                "prefix delete me here suffix\nprefix delete me there suffix",
			searchText:             "delete me ",
			charsAdded:             "", // ADDED field value
			originalChangeStartPos: 7,  // After "prefix " on line 1
			wantAnchors: []Anchor{
				// Original context: prefix="prefix ", affix="here suffix"
				// Anchor context: prefix="prefix ", affix="there suffix"
				{Position: 34, Score: 5 /*base*/ + 7 /*prefix*/ + 5 /*affix match ' suff'*/, Line: 2}, // Corrected score
			},
		},
		{
			name:                   "No match",
			oldText:                "nothing to find",
			searchText:             "delete me",
			charsAdded:             "", // ADDED field value
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Match at original position only",
			oldText:                "only match here",
			searchText:             "match here",
			charsAdded:             "", // ADDED field value
			originalChangeStartPos: 5,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Empty search text (charsRemoved)",
			oldText:                "some text",
			searchText:             "", // Represents charsRemoved
			charsAdded:             "", // ADDED field value
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{}, // Function returns early if searchText is empty
		},
		{
			name:                   "Invalid original position",
			oldText:                "text",
			searchText:             "t",
			charsAdded:             "", // ADDED field value
			originalChangeStartPos: -1,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Unicode text matching",
			oldText:                "你好 世界\n你好 中国", // "Hello World\nHello China"
			searchText:             "你好 ",          // "Hello "
			charsAdded:             "",             // ADDED field value
			originalChangeStartPos: 0,
			wantAnchors: []Anchor{
				// Original context: prefix="", affix="世界"
				// Anchor context: prefix="", affix="中国"
				{Position: 10, Score: 5 /*base*/ + 0 /*prefix*/ + 0 /*affix*/, Line: 2},
			},
		},
		{
			name:                   "Context scoring partial prefix/affix",
			oldText:                "abc delete me 123\nxyz delete me 456",
			searchText:             "delete me ",
			charsAdded:             "", // ADDED field value
			originalChangeStartPos: 4,  // After "abc "
			wantAnchors: []Anchor{
				// Original context: prefix="abc ", affix="123"
				// Anchor context: prefix="xyz ", affix="456"
				{Position: 22, Score: 5 /*base*/ + 1 /*prefix (' ')*/ + 0 /*affix*/, Line: 2},
			},
		},
		{
			name:                   "Context across lines - no match expected",
			oldText:                "delete me\nhere\ndelete me\nthere",
			searchText:             "delete me\nhere", // Search text includes newline
			charsAdded:             "",                // ADDED field value
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{}, // Local context prevents matching
		},
		{
			// Test case simulating pure insertion (charsRemoved is empty)
			name:                   "Pure insertion - no anchors expected (currently)",
			oldText:                "line1\nline3",
			searchText:             "",         // Represents charsRemoved = empty
			charsAdded:             "\nline2",  // ADDED: This is what was inserted
			originalChangeStartPos: 5,          // Position where insertion happened
			wantAnchors:            []Anchor{}, // Current logic returns empty if searchText is empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass tt.charsAdded to the function call
			gotAnchors := findAndScoreAnchors(tt.oldText, tt.charsAdded, tt.searchText /*charsRemoved*/, tt.originalChangeStartPos)

			// Custom comparison logic for slices of Anchors
			sortAnchors(gotAnchors)     // Sort actual anchors
			sortAnchors(tt.wantAnchors) // Sort expected anchors for consistent comparison

			if diff := cmp.Diff(tt.wantAnchors, gotAnchors); diff != "" {
				t.Errorf("findAndScoreAnchors() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
