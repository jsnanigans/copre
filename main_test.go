package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestDiffStrings(t *testing.T) {
	tests := []struct {
		name        string
		oldStr      string
		newStr      string
		wantAdded   string
		wantRemoved string
	}{
		{"no change", "abc", "abc", "", ""},
		{"simple addition", "abc", "abcd", "d", ""},
		{"simple removal", "abcd", "abc", "", "d"},
		{"replacement", "abc", "axc", "x", "b"},
		{"add beginning", "abc", "xabc", "x", ""},
		{"remove beginning", "xabc", "abc", "", "x"},
		{"add end", "abc", "abcx", "x", ""},
		{"remove end", "abcx", "abc", "", "x"},
		{"complex change", "abcdef", "axyeef", "xye", "bcd"},
		{"empty old", "", "abc", "abc", ""},
		{"empty new", "abc", "", "", "abc"},
		{"both empty", "", "", "", ""},
		{"multiline add", "line1\nline2", "line1\nnew line\nline2", "new line\n", ""},
		{"multiline remove", "line1\nold line\nline2", "line1\nline2", "", "old line\n"},
		{"multiline replace", "line1\nold\nline3", "line1\nnew\nline3", "new", "old"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdded, gotRemoved := diffStrings(tt.oldStr, tt.newStr)
			if gotAdded != tt.wantAdded {
				t.Errorf("diffStrings() gotAdded = %q, want %q", gotAdded, tt.wantAdded)
			}
			if gotRemoved != tt.wantRemoved {
				t.Errorf("diffStrings() gotRemoved = %q, want %q", gotRemoved, tt.wantRemoved)
			}
		})
	}
}

// Helper to reset cache between handler tests
func resetCache() {
	cacheMutex.Lock()
	fileCache = make(map[string]CachedFileContent)
	cacheMutex.Unlock()
}

func TestFileUpdateHandler(t *testing.T) {
	// Note: Testing log output directly is complex in standard unit tests.
	// These tests focus on HTTP response, status codes, and cache state.

	server := httptest.NewServer(http.HandlerFunc(fileUpdateHandler))
	defer server.Close()

	tests := []struct {
		name           string
		method         string
		body           interface{}
		wantStatusCode int
		wantBody       string             // Expected substring in response body
		setupCache     func()             // Function to set up cache before test
		verifyCache    func(t *testing.T) // Function to verify cache after test
	}{
		{
			name:           "initial post",
			method:         http.MethodPost,
			body:           FileUpdateRequest{Filename: "test.txt", Content: "hello"},
			wantStatusCode: http.StatusOK,
			wantBody:       "Content for test.txt received",
			setupCache:     resetCache,
			verifyCache: func(t *testing.T) {
				cacheMutex.RLock()
				defer cacheMutex.RUnlock()
				if cached, ok := fileCache["test.txt"]; !ok || cached.Current != "hello" {
					t.Errorf("Cache verification failed for initial post: expected Current='hello', got %+v (exists: %v)", cached, ok)
				}
			},
		},
		{
			name:           "update post",
			method:         http.MethodPost,
			body:           FileUpdateRequest{Filename: "test.txt", Content: "hello world"},
			wantStatusCode: http.StatusOK,
			wantBody:       "Content for test.txt received",
			setupCache: func() {
				resetCache()
				cacheMutex.Lock()
				fileCache["test.txt"] = CachedFileContent{Current: "hello", Previous: ""}
				cacheMutex.Unlock()
			},
			verifyCache: func(t *testing.T) {
				cacheMutex.RLock()
				defer cacheMutex.RUnlock()
				if cached, ok := fileCache["test.txt"]; !ok || cached.Current != "hello world" || cached.Previous != "hello" {
					t.Errorf("Cache verification failed for update post: expected Current='hello world', Previous='hello', got %+v (exists: %v)", cached, ok)
				}
			},
		},
		{
			name:           "no change post",
			method:         http.MethodPost,
			body:           FileUpdateRequest{Filename: "test.txt", Content: "same content"},
			wantStatusCode: http.StatusOK,
			wantBody:       "Content for test.txt received",
			setupCache: func() {
				resetCache()
				cacheMutex.Lock()
				fileCache["test.txt"] = CachedFileContent{Current: "same content", Previous: "something else"}
				cacheMutex.Unlock()
			},
			verifyCache: func(t *testing.T) {
				cacheMutex.RLock()
				defer cacheMutex.RUnlock()
				if cached, ok := fileCache["test.txt"]; !ok || cached.Current != "same content" || cached.Previous != "same content" {
					t.Errorf("Cache verification failed for no change post: expected Current='same content', Previous='same content', got %+v (exists: %v)", cached, ok)
				}
			},
		},
		{
			name:           "invalid method",
			method:         http.MethodGet,
			body:           nil,
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "Only POST method is allowed",
			setupCache:     resetCache,
			verifyCache:    nil, // Cache shouldn't change
		},
		{
			name:           "invalid json",
			method:         http.MethodPost,
			body:           "not json", // Send raw string instead of marshalled struct
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "Invalid request body",
			setupCache:     resetCache,
			verifyCache:    nil, // Cache shouldn't change
		},
		{
			name:           "empty filename",
			method:         http.MethodPost,
			body:           FileUpdateRequest{Filename: "", Content: "some content"},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "Filename cannot be empty",
			setupCache:     resetCache,
			verifyCache:    nil, // Cache shouldn't change
		},
	}

	// Override http.DefaultServeMux for tests to use our handler directly
	// Store original mux
	originalMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	http.HandleFunc("/update", fileUpdateHandler)
	// Restore original mux after tests
	defer func() {
		http.DefaultServeMux = originalMux
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupCache != nil {
				tt.setupCache()
			}

			var reqBody *bytes.Buffer
			if tt.body != nil {
				if rawString, ok := tt.body.(string); ok {
					reqBody = bytes.NewBufferString(rawString)
				} else {
					jsonData, err := json.Marshal(tt.body)
					if err != nil {
						t.Fatalf("Failed to marshal request body: %v", err)
					}
					reqBody = bytes.NewBuffer(jsonData)
				}
			} else {
				reqBody = bytes.NewBuffer([]byte{})
			}

			req := httptest.NewRequest(tt.method, "/update", reqBody)
			if tt.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			fileUpdateHandler(w, req) // Directly call the handler

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", res.StatusCode, tt.wantStatusCode)
			}

			bodyBytes := new(bytes.Buffer)
			bodyBytes.ReadFrom(res.Body)
			bodyString := bodyBytes.String()

			if !bytes.Contains(bodyBytes.Bytes(), []byte(tt.wantBody)) {
				t.Errorf("handler returned unexpected body: got %q want substring %q", bodyString, tt.wantBody)
			}

			if tt.verifyCache != nil {
				tt.verifyCache(t)
			}
		})
	}
}

func TestPredictChanges(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		oldContent      string
		newContent      string
		wantPredictions []PredictedChange
	}{
		{
			name:            "no change",
			filename:        "test.txt",
			oldContent:      "line1\nline2",
			newContent:      "line1\nline2",
			wantPredictions: []PredictedChange{}, // Expect no predictions
		},
		{
			name:            "initial content",
			filename:        "test.txt",
			oldContent:      "",
			newContent:      "line1\nline2",
			wantPredictions: []PredictedChange{}, // Expect no predictions
		},
		{
			name:     "simple replacement prediction",
			filename: "test.lua",
			oldContent: `local qweqwe = require('nvim.suggest')
local oni = require('nvim.suggest')
local two = require('nvim.suggest')
local ui = require('nvim-complete.ui')`,
			newContent: `local qweqwe = require('nvim.suggest')
local oni = require('nvim.suggest')
local two = require('nvim-adji.suggest')
local ui = require('nvim-complete.ui')`,
			wantPredictions: []PredictedChange{
				{
					LineNumber:    1, // 1-based line number in newContent
					OriginalLine:  "local qweqwe = require('nvim.suggest')",
					PredictedLine: "local qweqwe = require('nvim-adji.suggest')",
				},
				{
					LineNumber:    2,
					OriginalLine:  "local oni = require('nvim.suggest')",
					PredictedLine: "local oni = require('nvim-adji.suggest')",
				},
				// Line 3 was the actual change, so it shouldn't be predicted
			},
		},
		{
			name:            "replacement with no other matches",
			filename:        "test.py",
			oldContent:      `print("hello")\nprint("world")`,
			newContent:      `print("hello")\nprint("goodbye")`, // Change "world" -> "goodbye"
			wantPredictions: []PredictedChange{},                // No other lines contain "world"
		},
		{
			name:            "insertion only",
			filename:        "test.txt",
			oldContent:      `line1\nline3`,
			newContent:      `line1\nline2\nline3`,
			wantPredictions: []PredictedChange{}, // Current logic doesn't predict insertions
		},
		{
			name:            "deletion only",
			filename:        "test.txt",
			oldContent:      `line1\nline2\nline3`,
			newContent:      `line1\nline3`,
			wantPredictions: []PredictedChange{}, // Current logic doesn't predict deletions
		},
		{
			name:            "multi-line change",
			filename:        "test.java",
			oldContent:      `System.out.println("one");\nSystem.out.println("two");`,
			newContent:      `// System.out.println("one");\n// System.out.println("two");`,
			wantPredictions: []PredictedChange{}, // Current logic only handles simple single-line replacements
		},
		// --- NEW TEST CASES --- //
		{
			name:       "variable renaming",
			filename:   "script.py",
			oldContent: `old_variable = 10\nprint(old_variable)\nresult = old_variable * 5`,
			newContent: `new_variable = 10\nprint(old_variable)\nresult = old_variable * 5`, // Change first line only
			wantPredictions: []PredictedChange{
				{
					LineNumber:    2, // 1-based index in newContent
					OriginalLine:  "print(old_variable)",
					PredictedLine: "print(new_variable)",
				},
				{
					LineNumber:    3,
					OriginalLine:  "result = old_variable * 5",
					PredictedLine: "result = new_variable * 5",
				},
			},
		},
		{
			name:       "partial string removal",
			filename:   "config.json",
			oldContent: `{\n  "path1": "/path/to/old/file1",\n  "path2": "/path/to/old/file2"\n}`,
			newContent: `{\n  "path1": "/path/to/file1",\n  "path2": "/path/to/old/file2"\n}`, // Removed "/old" from path1 only
			wantPredictions: []PredictedChange{
				{
					LineNumber:    3,
					OriginalLine:  "  \"path2\": \"/path/to/old/file2\"",
					PredictedLine: "  \"path2\": \"/path/to/file2\"",
				},
			},
		},
		{
			name:       "partial string replacement",
			filename:   "api_client.go",
			oldContent: `endpoint_users := "/api/v1/users"\nendpoint_posts := "/api/v1/posts"\nfetch(endpoint_users)`,
			newContent: `endpoint_users := "/api/v2/users"\nendpoint_posts := "/api/v1/posts"\nfetch(endpoint_users)`, // Changed v1 to v2 in first line
			wantPredictions: []PredictedChange{
				{
					LineNumber:    2,
					OriginalLine:  "endpoint_posts := \"/api/v1/posts\"",
					PredictedLine: "endpoint_posts := \"/api/v2/posts\"",
				},
				// Note: Might not predict change inside 'fetch' depending on logic sophistication
			},
		},
		{
			name:       "adding prefix",
			filename:   "main.js",
			oldContent: `const data1 = getData();\nconst data2 = getData();\nprocess(data1);`,
			newContent: `const data1 = new_getData();\nconst data2 = getData();\nprocess(data1);`, // Added prefix in first line
			wantPredictions: []PredictedChange{
				{
					LineNumber:    2,
					OriginalLine:  "const data2 = getData();",
					PredictedLine: "const data2 = new_getData();",
				},
			},
		},
		{
			name:       "adding suffix",
			filename:   "styles.css",
			oldContent: `.MyClass { color: red; }\n.AnotherClass { background: blue; }\n.MyClass .child { color: green; }`,
			newContent: `.MyClass_v2 { color: red; }\n.AnotherClass { background: blue; }\n.MyClass .child { color: green; }`, // Added suffix in first line
			wantPredictions: []PredictedChange{
				{
					LineNumber:    3,
					OriginalLine:  ".MyClass .child { color: green; }",
					PredictedLine: ".MyClass_v2 .child { color: green; }",
				},
			},
		},
		// --- END NEW TEST CASES --- //
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: predictChanges currently logs internally. We are only checking the return value.
			gotPredictions := predictChanges(tt.filename, tt.oldContent, tt.newContent)

			// Use reflect.DeepEqual for slice comparison. Order matters.
			// Consider sorting or using a map-based comparison if order is not guaranteed.
			if !reflect.DeepEqual(gotPredictions, tt.wantPredictions) {
				// Provide detailed error output
				t.Errorf("predictChanges() got = %v, want %v", gotPredictions, tt.wantPredictions)
				// Log differences for easier debugging
				if len(gotPredictions) != len(tt.wantPredictions) {
					t.Logf("Length mismatch: got %d, want %d", len(gotPredictions), len(tt.wantPredictions))
				} else {
					for i := range gotPredictions {
						if !reflect.DeepEqual(gotPredictions[i], tt.wantPredictions[i]) {
							t.Logf("Difference at index %d:\n  Got:  %+v\n  Want: %+v", i, gotPredictions[i], tt.wantPredictions[i])
						}
					}
				}
			}
		})
	}
}
