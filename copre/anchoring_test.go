package copre

import (
	"reflect"
	"sort"
	"testing"
)

func TestFindAndScoreAnchors(t *testing.T) {
	// Scoring: Base (5) + prefix match len + affix match len
	tests := []struct {
		name       string
		oldText    string
		searchText string
		// prefix                 string // Removed
		// affix                  string // Removed
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
			name: "One exact anchor, context from original",
			// Original change: remove "remove this" at pos 16. Context: prefix=" and ", affix=" too"
			// Anchor: "remove this" at pos 0. Context: prefix="", affix=" and "
			oldText:                "remove this and remove this too",
			searchText:             "remove this",
			originalChangeStartPos: 16, // The second "remove this"
			wantAnchors: []Anchor{
				// Score: 5 (base) + 0 (prefix mismatch "" vs " and ") + 0 (affix mismatch " and " vs " too") = 5
				{Position: 0, Score: 5, Line: 1},
			},
		},
		{
			name: "Multiple exact anchors",
			// Original change: remove "A" at pos 0. Context: prefix="", affix=" "
			oldText:                "A A A A",
			searchText:             "A",
			originalChangeStartPos: 0, // First 'A'
			wantAnchors: []Anchor{
				// Anchor @ 2: Context prefix=" ", affix=" ". Score: 5 (base) + 0 (prefix "" vs " ") + 1 (affix " " vs " ") = 6
				{Position: 2, Score: 6, Line: 1},
				// Anchor @ 4: Context prefix=" ", affix=" ". Score: 5 + 0 + 1 = 6
				{Position: 4, Score: 6, Line: 1},
				// Anchor @ 6: Context prefix=" ", affix="". Score: 5 + 0 + 0 (affix "" vs " ") = 5
				{Position: 6, Score: 5, Line: 1},
			},
		},
		{
			name: "Anchor with matching prefix",
			// Original change: remove "remove" at pos 20. Context: prefix="prefix ", affix=" too"
			// Anchor @ 7: Context prefix="prefix ", affix=" and "
			oldText:                "prefix remove and prefix remove too",
			searchText:             "remove",
			originalChangeStartPos: 20, // Second "remove"
			wantAnchors: []Anchor{
				// Score: 5 (base) + 7 (prefix match) + 0 (affix mismatch) = 12
				{Position: 7, Score: 12, Line: 1},
			},
		},
		{
			name: "Anchor with matching affix",
			// Original change: remove "remove" at pos 19. Context: prefix=" and ", affix=" affix too"
			// Anchor @ 0: Context prefix="", affix=" affix and "
			oldText:                "remove affix and remove affix too",
			searchText:             "remove",
			originalChangeStartPos: 19, // Second "remove"
			wantAnchors: []Anchor{
				// Score: 5 (base) + 0 (prefix mismatch) + 6 (affix match " affix") = 11
				{Position: 0, Score: 11, Line: 1},
			},
		},
		{
			name: "Anchor with matching prefix and affix",
			// Original change: remove "remove" at pos 27. Context: prefix="prefix ", affix=" affix too"
			// Anchor @ 7: Context prefix="prefix ", affix=" affix and "
			oldText:                "prefix remove affix and prefix remove affix too",
			searchText:             "remove",
			originalChangeStartPos: 27, // Second "remove"
			wantAnchors: []Anchor{
				// Score: 5 (base) + 7 (prefix) + 6 (affix " affix") = 18
				{Position: 7, Score: 18, Line: 1},
			},
		},
		{
			name: "Score variation based on local context",
			// Original change: remove "remove" at pos 19. Context: prefix=" def", affix="xyz"
			oldText:                "abcremovexyz and defremovexyz",
			searchText:             "remove",
			originalChangeStartPos: 19, // Second "remove"
			wantAnchors: []Anchor{
				// Anchor @ 3: Context prefix="abc", affix="xyz".
				// Score: 5 (base) + 0 (prefix mismatch) + 3 (affix "xyz") = 8
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
			name:                   "Original pos -1",
			oldText:                "abc abc",
			searchText:             "abc",
			originalChangeStartPos: -1,
			wantAnchors:            []Anchor{}, // Should not search if original pos is unknown
		},
		{
			name: "Multi-line anchor context match",
			// Original change: remove "remove this" at pos 23. Context: prefix="\n", affix=""
			oldText:                "line1\nremove this\nline3\nremove this",
			searchText:             "remove this",
			originalChangeStartPos: 23, // Start of second "remove this"
			wantAnchors: []Anchor{
				// Anchor @ 6: Context prefix="\n", affix="\nline3".
				// Score: 5 (base) + 1 (prefix "\n") + 0 (affix mismatch) = 6
				{Position: 6, Score: 6, Line: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Remove prefix/affix from the call
			gotAnchors := findAndScoreAnchors(tt.oldText, tt.searchText, tt.originalChangeStartPos)
			// Sort for stable comparison
			sort.Slice(gotAnchors, func(i, j int) bool { return gotAnchors[i].Position < gotAnchors[j].Position })
			sort.Slice(tt.wantAnchors, func(i, j int) bool { return tt.wantAnchors[i].Position < tt.wantAnchors[j].Position })

			if !reflect.DeepEqual(gotAnchors, tt.wantAnchors) {
				t.Errorf("findAndScoreAnchors() = %+v, want %+v", gotAnchors, tt.wantAnchors)
			}
		})
	}
}
