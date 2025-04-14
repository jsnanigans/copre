package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	fileCache = make(map[string]string)
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
				if content, ok := fileCache["test.txt"]; !ok || content != "hello" {
					t.Errorf("Cache verification failed: expected 'hello' for test.txt, got %q (exists: %v)", content, ok)
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
				fileCache["test.txt"] = "hello"
				cacheMutex.Unlock()
			},
			verifyCache: func(t *testing.T) {
				cacheMutex.RLock()
				defer cacheMutex.RUnlock()
				if content, ok := fileCache["test.txt"]; !ok || content != "hello world" {
					t.Errorf("Cache verification failed: expected 'hello world' for test.txt, got %q (exists: %v)", content, ok)
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
				fileCache["test.txt"] = "same content"
				cacheMutex.Unlock()
			},
			verifyCache: func(t *testing.T) {
				cacheMutex.RLock()
				defer cacheMutex.RUnlock()
				if content, ok := fileCache["test.txt"]; !ok || content != "same content" {
					t.Errorf("Cache verification failed: expected 'same content' for test.txt, got %q (exists: %v)", content, ok)
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
