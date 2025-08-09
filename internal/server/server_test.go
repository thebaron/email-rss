package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: "/tmp/feeds",
	}

	server := New(config)

	assert.NotNil(t, server)
	assert.Equal(t, config, server.config)
}

func TestHandleRoot(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFeed(t, tmpDir, "inbox.xml")
	createTestFeed(t, tmpDir, "sent.xml")
	createTestFeed(t, tmpDir, "important.xml")

	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: tmpDir,
	}

	server := New(config)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.handleRoot(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "<title>EmailRSS Feeds</title>")
	assert.Contains(t, body, "<h1>Available RSS Feeds</h1>")
	assert.Contains(t, body, "inbox.xml")
	assert.Contains(t, body, "sent.xml")
	assert.Contains(t, body, "important.xml")
	assert.Contains(t, body, `href="/feeds/inbox.xml"`)
	assert.Contains(t, body, `href="/feeds/sent.xml"`)
	assert.Contains(t, body, `href="/feeds/important.xml"`)
}

func TestHandleRootNonRootPath(t *testing.T) {
	tmpDir := t.TempDir()
	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: tmpDir,
	}

	server := New(config)

	req := httptest.NewRequest("GET", "/nonroot", nil)
	w := httptest.NewRecorder()

	server.handleRoot(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHandleRootNoFeeds(t *testing.T) {
	tmpDir := t.TempDir()
	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: tmpDir,
	}

	server := New(config)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.handleRoot(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	assert.Contains(t, body, "<h1>Available RSS Feeds</h1>")
	assert.Contains(t, body, "<ul></ul>")
}

func TestHandleRootDirectoryError(t *testing.T) {
	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: "/nonexistent/directory",
	}

	server := New(config)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.handleRoot(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	assert.Contains(t, body, "<ul></ul>")
}

func TestHandleFeed(t *testing.T) {
	tmpDir := t.TempDir()
	feedContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <description>Test RSS Feed</description>
  </channel>
</rss>`

	feedPath := filepath.Join(tmpDir, "inbox.xml")
	err := os.WriteFile(feedPath, []byte(feedContent), 0644)
	require.NoError(t, err)

	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: tmpDir,
	}

	server := New(config)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
		expectedBody   string
	}{
		{
			name:           "existing feed with .xml extension",
			path:           "/feeds/inbox.xml",
			expectedStatus: http.StatusOK,
			expectedType:   "application/rss+xml",
			expectedBody:   feedContent,
		},
		{
			name:           "existing feed without .xml extension",
			path:           "/feeds/inbox",
			expectedStatus: http.StatusOK,
			expectedType:   "application/rss+xml",
			expectedBody:   feedContent,
		},
		{
			name:           "nonexistent feed",
			path:           "/feeds/nonexistent.xml",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty feed name",
			path:           "/feeds/",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			server.handleFeed(w, req)

			resp := w.Result()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedType != "" {
				assert.Equal(t, tt.expectedType, resp.Header.Get("Content-Type"))
			}

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "max-age=3600", resp.Header.Get("Cache-Control"))
				body := w.Body.String()
				assert.Equal(t, tt.expectedBody, body)
			}
		})
	}
}

func TestHandleFeedPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	outsideDir := filepath.Dir(tmpDir)
	maliciousFile := filepath.Join(outsideDir, "malicious.xml")
	err := os.WriteFile(maliciousFile, []byte("malicious content"), 0644)
	require.NoError(t, err)
	defer os.Remove(maliciousFile)

	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: tmpDir,
	}

	server := New(config)

	pathTraversalAttempts := []string{
		"/feeds/../malicious.xml",
		"/feeds/..%2Fmalicious.xml",
		"/feeds/%2E%2E%2Fmalicious.xml",
		"/feeds/subdir/../../malicious.xml",
	}

	for _, attempt := range pathTraversalAttempts {
		t.Run("path_traversal_"+attempt, func(t *testing.T) {
			req := httptest.NewRequest("GET", attempt, nil)
			w := httptest.NewRecorder()

			server.handleFeed(w, req)

			resp := w.Result()
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

func TestHandleHealth(t *testing.T) {
	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: "/tmp",
	}

	server := New(config)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body := w.Body.String()
	assert.JSONEq(t, `{"status": "ok", "service": "emailrss"}`, body)
}

func TestListFeeds(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFeed(t, tmpDir, "inbox.xml")
	createTestFeed(t, tmpDir, "sent.xml")
	createTestFile(t, tmpDir, "notxml.txt")

	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)
	createTestFeed(t, subDir, "subfeed.xml")

	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: tmpDir,
	}

	server := New(config)

	feeds, err := server.listFeeds()
	assert.NoError(t, err)
	assert.Len(t, feeds, 2)
	assert.Contains(t, feeds, "inbox.xml")
	assert.Contains(t, feeds, "sent.xml")
	assert.NotContains(t, feeds, "notxml.txt")
	assert.NotContains(t, feeds, "subfeed.xml")
}

func TestListFeedsNonexistentDirectory(t *testing.T) {
	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: "/nonexistent/directory",
	}

	server := New(config)

	feeds, err := server.listFeeds()
	assert.NoError(t, err)
	assert.Len(t, feeds, 0)
}

func TestIsValidFeedPath(t *testing.T) {
	tmpDir := t.TempDir()
	config := ServerConfig{
		Host:     "localhost",
		Port:     8080,
		FeedsDir: tmpDir,
	}

	server := New(config)

	tests := []struct {
		name     string
		feedPath string
		expected bool
	}{
		{
			name:     "valid feed in feeds directory",
			feedPath: filepath.Join(tmpDir, "inbox.xml"),
			expected: true,
		},
		{
			name:     "valid feed with subdirectory",
			feedPath: filepath.Join(tmpDir, "sub", "feed.xml"),
			expected: true,
		},
		{
			name:     "path outside feeds directory",
			feedPath: filepath.Join(filepath.Dir(tmpDir), "outside.xml"),
			expected: false,
		},
		{
			name:     "path with double dots",
			feedPath: filepath.Join(tmpDir, "..", "outside.xml"),
			expected: false,
		},
		{
			name:     "non-xml file",
			feedPath: filepath.Join(tmpDir, "not-xml.txt"),
			expected: false,
		},
		{
			name:     "path without extension",
			feedPath: filepath.Join(tmpDir, "noext"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.isValidFeedPath(tt.feedPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func createTestFeed(t *testing.T, dir, filename string) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>` + filename + `</title>
  </channel>
</rss>`

	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

func createTestFile(t *testing.T, dir, filename string) {
	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte("test content"), 0644)
	require.NoError(t, err)
}
