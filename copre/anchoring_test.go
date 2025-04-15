package copre

import (
	"reflect"
	"testing"
)

func TestFindAndScoreAnchors(t *testing.T) {
	tests := []struct {
		name                   string
		oldText                string
		searchText             string
		prefix                 string
		affix                  string
		originalChangeStartPos int
		wantAnchors            []Anchor
	}{
		{
			name:                   "No anchors",
			oldText:                "abc def ghi",
			searchText:             "xyz",
			prefix:                 "",
			affix:                  "",
			originalChangeStartPos: 0,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "One exact anchor, no context",
			oldText:                "remove this and remove this too",
			searchText:             "remove this",
			prefix:                 "",
			affix:                  " too", // Affix from original change (pos 16)
			originalChangeStartPos: 16,     // The second "remove this"
			wantAnchors: []Anchor{
				{Position: 0, Score: 5, Line: 1}, // Base score 5
			},
		},
		{
			name:                   "Multiple exact anchors",
			oldText:                "A A A A",
			searchText:             "A",
			prefix:                 "", // Original change context
			affix:                  " ",
			originalChangeStartPos: 0, // First 'A'
			wantAnchors: []Anchor{
				{Position: 2, Score: 5 + 1, Line: 1}, // Base + affix match
				{Position: 4, Score: 5 + 1, Line: 1},
				{Position: 6, Score: 5, Line: 1}, // Last A has no affix space
			},
		},
		{
			name:                   "Anchor with prefix",
			oldText:                "prefix remove and prefix remove too",
			searchText:             "remove",
			prefix:                 "prefix ",
			affix:                  " too",
			originalChangeStartPos: 20, // Second "remove"
			wantAnchors: []Anchor{
				{Position: 7, Score: 5 + len("prefix "), Line: 1}, // Base + prefix score
			},
		},
		{
			name:                   "Anchor with affix",
			oldText:                "remove affix and remove affix too",
			searchText:             "remove",
			prefix:                 "",
			affix:                  " affix too",
			originalChangeStartPos: 19, // Second "remove"
			wantAnchors: []Anchor{
				{Position: 0, Score: 5 + len(" affix"), Line: 1}, // Base + affix score
			},
		},
		{
			name:                   "Anchor with prefix and affix",
			oldText:                "prefix remove affix and prefix remove affix too",
			searchText:             "remove",
			prefix:                 "prefix ",
			affix:                  " affix too",
			originalChangeStartPos: 27, // Second "remove"
			wantAnchors: []Anchor{
				{Position: 7, Score: 5 + len("prefix ") + len(" affix"), Line: 1}, // Base + prefix + affix score
			},
		},
		{
			name:                   "Score variation",
			oldText:                "abcremovexyz and defremovexyz",
			searchText:             "remove",
			prefix:                 "def", // From second removal
			affix:                  "xyz", // From second removal
			originalChangeStartPos: 19,    // Second "remove"
			wantAnchors: []Anchor{
				{Position: 3, Score: 5 + 0 + len("xyz"), Line: 1}, // Base + no prefix match + affix match
			},
		},
		{
			name:                   "Search text empty",
			oldText:                "abc",
			searchText:             "",
			prefix:                 "a",
			affix:                  "c",
			originalChangeStartPos: 1,
			wantAnchors:            []Anchor{},
		},
		{
			name:                   "Original pos -1",
			oldText:                "abc abc",
			searchText:             "abc",
			prefix:                 "",
			affix:                  "",
			originalChangeStartPos: -1,
			wantAnchors:            []Anchor{}, // Should not search if original pos is unknown
		},
		{
			name:                   "Multi-line anchor",
			oldText:                "line1\nremove this\nline3\nremove this",
			searchText:             "remove this",
			prefix:                 "\n", // Assuming original was second one
			affix:                  "",
			originalChangeStartPos: 23, // Start of second "remove this"
			wantAnchors: []Anchor{
				{Position: 6, Score: 5 + 1, Line: 2}, // Base + prefix ('\n')
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAnchors := findAndScoreAnchors(tt.oldText, tt.searchText, tt.prefix, tt.affix, tt.originalChangeStartPos)
			if !reflect.DeepEqual(gotAnchors, tt.wantAnchors) {
				t.Errorf("findAndScoreAnchors() = %v, want %v", gotAnchors, tt.wantAnchors)
			}
		})
	}
}
