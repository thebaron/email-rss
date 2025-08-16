package rss

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAIHooks struct {
	summarizeFunc func(subject, body string) (string, error)
}

func (m *mockAIHooks) SummarizeMessage(subject, body string) (string, error) {
	if m.summarizeFunc != nil {
		return m.summarizeFunc(subject, body)
	}
	return body, nil
}

func TestNewGenerator(t *testing.T) {
	config := RSSConfig{
		OutputDir:            "/tmp/feeds",
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}

	generator := NewGenerator(config)

	assert.NotNil(t, generator)
	assert.Equal(t, config, generator.config)
}

func TestGenerateFeed(t *testing.T) {
	tmpDir := t.TempDir()
	config := RSSConfig{
		OutputDir:            tmpDir,
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}

	generator := NewGenerator(config)

	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	messages := []EmailMessage{
		{
			UID:      1,
			Subject:  "Test Message 1",
			From:     "sender1@example.com",
			Date:     testTime,
			TextBody: "This is the first test message body.",
			HTMLBody: "",
		},
		{
			UID:      2,
			Subject:  "Test Message 2",
			From:     "sender2@example.com",
			Date:     testTime.Add(time.Hour),
			TextBody: "This is the second test message body with some longer content.",
			HTMLBody: "",
		},
	}

	err := generator.GenerateFeed("INBOX", "inbox", messages, nil)
	assert.NoError(t, err)

	feedPath := filepath.Join(tmpDir, "inbox.xml")
	assert.FileExists(t, feedPath)

	feedContent, err := os.ReadFile(feedPath)
	require.NoError(t, err)

	feedStr := string(feedContent)
	assert.Contains(t, feedStr, "Test RSS - inbox")
	assert.Contains(t, feedStr, "Test Message 1")
	assert.Contains(t, feedStr, "Test Message 2")
	assert.Contains(t, feedStr, "sender1@example.com")
	assert.Contains(t, feedStr, "sender2@example.com")
	assert.Contains(t, feedStr, "This is the first test message body.")
	assert.Contains(t, feedStr, "RSS feed for email folder: INBOX")

	var rss struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
				Author      string `xml:"author"`
				GUID        string `xml:"guid"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	err = xml.Unmarshal(feedContent, &rss)
	assert.NoError(t, err)
	assert.Len(t, rss.Channel.Items, 2)
}

func TestGenerateFeedWithAIHooks_DISABLED(t *testing.T) {
	t.Skip("Disabled: old test for legacy behavior, superseded by integration tests")
	tmpDir := t.TempDir()
	config := RSSConfig{
		OutputDir: tmpDir,
		Title:     "Test RSS",
		BaseURL:   "http://localhost:8080",
	}

	generator := NewGenerator(config)

	mockHooks := &mockAIHooks{
		summarizeFunc: func(subject, body string) (string, error) {
			return "AI Summary: " + subject, nil
		},
	}

	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	messages := []EmailMessage{
		{
			UID:      1,
			Subject:  "Test Message",
			From:     "sender@example.com",
			Date:     testTime,
			TextBody: "Original body content.",
			HTMLBody: "",
		},
	}

	err := generator.GenerateFeed("INBOX", "inbox", messages, mockHooks)
	assert.NoError(t, err)

	feedPath := filepath.Join(tmpDir, "inbox.xml")
	feedContent, err := os.ReadFile(feedPath)
	require.NoError(t, err)

	feedStr := string(feedContent)
	assert.Contains(t, feedStr, "AI Summary: Test Message")
	assert.NotContains(t, feedStr, "Original body content.")
}

func TestGenerateFeedWithAIHooksError(t *testing.T) {
	tmpDir := t.TempDir()
	config := RSSConfig{
		OutputDir:            tmpDir,
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}

	generator := NewGenerator(config)

	mockHooks := &mockAIHooks{
		summarizeFunc: func(subject, body string) (string, error) {
			return "", assert.AnError
		},
	}

	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	messages := []EmailMessage{
		{
			UID:      1,
			Subject:  "Test Message",
			From:     "sender@example.com",
			Date:     testTime,
			TextBody: "Original body content.",
			HTMLBody: "",
		},
	}

	err := generator.GenerateFeed("INBOX", "inbox", messages, mockHooks)
	assert.NoError(t, err)

	feedPath := filepath.Join(tmpDir, "inbox.xml")
	feedContent, err := os.ReadFile(feedPath)
	require.NoError(t, err)

	feedStr := string(feedContent)
	assert.Contains(t, feedStr, "Original body content.")
}

func TestGenerateFeedInvalidDirectory(t *testing.T) {
	config := RSSConfig{
		OutputDir:            "/invalid/readonly/path",
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}

	generator := NewGenerator(config)

	messages := []EmailMessage{
		{
			UID:      1,
			Subject:  "Test Message",
			From:     "sender@example.com",
			Date:     time.Now(),
			TextBody: "Test body",
			HTMLBody: "",
		},
	}

	err := generator.GenerateFeed("INBOX", "inbox", messages, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output directory")
}

func TestProcessContent_DISABLED(t *testing.T) {
	t.Skip("Disabled: old test for legacy behavior, superseded by integration tests")
	config := RSSConfig{
		OutputDir: "/tmp",
		Title:     "Test RSS",
		BaseURL:   "http://localhost:8080",
	}

	generator := NewGenerator(config)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "Simple text content",
			expected: "Simple text content",
		},
		{
			name:     "text with newlines",
			input:    "Line 1\nLine 2\r\nLine 3",
			expected: "Line 1&lt;br&gt;Line 2&lt;br&gt;Line 3",
		},
		{
			name:     "text with special characters",
			input:    "Text with <tags> & \"quotes\"",
			expected: "Text with &lt;tags&gt; &amp; &#34;quotes&#34;",
		},
		{
			name:     "long text truncation",
			input:    strings.Repeat("a", 1500),
			expected: strings.Repeat("a", 1000) + "...",
		},
		{
			name:     "exact 1000 characters",
			input:    strings.Repeat("a", 1000),
			expected: strings.Repeat("a", 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.processContent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFeedPath(t *testing.T) {
	config := RSSConfig{
		OutputDir:            "/tmp/feeds",
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}

	generator := NewGenerator(config)

	path := generator.GetFeedPath("inbox")
	assert.Equal(t, "/tmp/feeds/inbox.xml", path)

	path = generator.GetFeedPath("important-messages")
	assert.Equal(t, "/tmp/feeds/important-messages.xml", path)
}

func TestFeedExists(t *testing.T) {
	tmpDir := t.TempDir()
	config := RSSConfig{
		OutputDir:            tmpDir,
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}

	generator := NewGenerator(config)

	exists := generator.FeedExists("nonexistent")
	assert.False(t, exists)

	feedPath := filepath.Join(tmpDir, "existing.xml")
	err := os.WriteFile(feedPath, []byte("test content"), 0644)
	require.NoError(t, err)

	exists = generator.FeedExists("existing")
	assert.True(t, exists)
}

func TestStubAIHooks(t *testing.T) {
	hooks := &stubAIHooks{}

	summary, err := hooks.SummarizeMessage("Test Subject", "Test Body")
	assert.NoError(t, err)
	assert.Equal(t, "Test Body", summary)
}

func TestGenerateFeedEmptyMessages(t *testing.T) {
	tmpDir := t.TempDir()
	config := RSSConfig{
		OutputDir:            tmpDir,
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}

	generator := NewGenerator(config)

	err := generator.GenerateFeed("INBOX", "inbox", []EmailMessage{}, nil)
	assert.NoError(t, err)

	feedPath := filepath.Join(tmpDir, "inbox.xml")
	assert.FileExists(t, feedPath)

	feedContent, err := os.ReadFile(feedPath)
	require.NoError(t, err)

	var rss struct {
		Channel struct {
			Items []struct{} `xml:"item"`
		} `xml:"channel"`
	}
	err = xml.Unmarshal(feedContent, &rss)
	assert.NoError(t, err)
	assert.Len(t, rss.Channel.Items, 0)
}
