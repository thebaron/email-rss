package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"emailrss/internal/db"
	"emailrss/internal/imap"
	"emailrss/internal/rss"
)

// ValidationSample represents the input format for validation samples
type ValidationSample struct {
	UID      uint32 `json:"uid"`
	Subject  string `json:"subject"`
	From     string `json:"from"`
	Date     string `json:"date"`
	TextBody string `json:"text_body"`
	HTMLBody string `json:"html_body"`
}

// ValidationMockIMAPClient implements IMAPClient for validation testing
type ValidationMockIMAPClient struct {
	samples []ValidationSample
}

func (m *ValidationMockIMAPClient) GetMessages(ctx context.Context, folder string, since time.Time) ([]imap.Message, error) {
	var messages []imap.Message
	for _, sample := range m.samples {
		parsedDate, err := time.Parse(time.RFC3339, sample.Date)
		if err != nil {
			parsedDate = time.Now()
		}

		messages = append(messages, imap.Message{
			ID:      1,
			UID:     sample.UID,
			Subject: sample.Subject,
			From:    sample.From,
			Date:    parsedDate,
		})
	}
	return messages, nil
}

func (m *ValidationMockIMAPClient) GetMessageContent(ctx context.Context, uid uint32) (*imap.MessageContent, error) {
	for _, sample := range m.samples {
		if sample.UID == uid {
			return &imap.MessageContent{
				TextBody: sample.TextBody,
				HTMLBody: sample.HTMLBody,
			}, nil
		}
	}
	return &imap.MessageContent{}, fmt.Errorf("message not found: %d", uid)
}

func TestValidationSamples(t *testing.T) {
	validationDir := "../../validation_samples"

	// Check if validation samples directory exists
	if _, err := os.Stat(validationDir); os.IsNotExist(err) {
		t.Skipf("Validation samples directory not found: %s", validationDir)
		return
	}

	// Find all .in files
	inputFiles, err := filepath.Glob(filepath.Join(validationDir, "*.in"))
	require.NoError(t, err, "Failed to find input files")

	if len(inputFiles) == 0 {
		t.Skip("No validation input files found")
		return
	}

	for _, inputFile := range inputFiles {
		baseName := strings.TrimSuffix(filepath.Base(inputFile), ".in")
		expectedFile := filepath.Join(validationDir, baseName+".out")

		// Check if corresponding .out file exists
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Logf("Skipping %s: no corresponding .out file", baseName)
			continue
		}

		t.Run(baseName, func(t *testing.T) {
			runValidationTest(t, inputFile, expectedFile)
		})
	}
}

func runValidationTest(t *testing.T, inputFile, expectedFile string) {
	// Read and parse input file
	inputData, err := os.ReadFile(inputFile)
	require.NoError(t, err, "Failed to read input file")

	var sample ValidationSample
	err = json.Unmarshal(inputData, &sample)
	require.NoError(t, err, "Failed to parse input JSON")

	// Read expected output
	expectedOutput, err := os.ReadFile(expectedFile)
	require.NoError(t, err, "Failed to read expected output file")
	expectedContent := strings.TrimSpace(string(expectedOutput))

	// Set up test environment
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "validation.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Create mock IMAP client with the sample
	mockIMAP := &ValidationMockIMAPClient{
		samples: []ValidationSample{sample},
	}

	// Set up RSS generator
	rssConfig := rss.RSSConfig{
		OutputDir:            tempDir,
		Title:                "Validation Test",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}
	rssGenerator := rss.NewGenerator(rssConfig)

	// Create processor
	processor := New(mockIMAP, database, rssGenerator)

	// Process the sample
	ctx := context.Background()
	folders := map[string]string{
		"INBOX": "validation",
	}

	err = processor.ProcessFolders(ctx, folders)
	require.NoError(t, err, "Failed to process folders")

	// Read generated RSS feed
	rssPath := filepath.Join(tempDir, "validation.xml")
	require.FileExists(t, rssPath, "RSS feed was not generated")

	rssContent, err := os.ReadFile(rssPath)
	require.NoError(t, err, "Failed to read generated RSS")
	rssString := string(rssContent)

	// Extract the description content from the RSS
	actualContent := extractRSSDescription(t, rssString)

	// Normalize whitespace for comparison
	expectedContent = normalizeContent(expectedContent)
	actualContent = normalizeContent(actualContent)

	// Compare the content
	assert.Equal(t, expectedContent, actualContent,
		"Generated content doesn't match expected output.\nExpected:\n%s\n\nActual:\n%s",
		expectedContent, actualContent)

	t.Logf("✅ Validation passed for %s", filepath.Base(inputFile))
}

// extractRSSDescription extracts the item description content from RSS XML
func extractRSSDescription(t *testing.T, rssContent string) string {
	// Find the first <item> tag
	itemStart := strings.Index(rssContent, "<item>")
	if itemStart == -1 {
		t.Fatal("No <item> tag found in RSS content")
	}

	// Find the description within the item
	searchStart := itemStart
	start := strings.Index(rssContent[searchStart:], "<description>")
	if start == -1 {
		t.Fatal("No <description> tag found in RSS item")
	}
	start = searchStart + start + len("<description>")

	end := strings.Index(rssContent[start:], "</description>")
	if end == -1 {
		t.Fatal("No closing </description> tag found in RSS item")
	}

	description := rssContent[start : start+end]
	return strings.TrimSpace(description)
}

// normalizeContent normalizes whitespace and XML entities for comparison
func normalizeContent(content string) string {
	// Decode common XML entities
	content = strings.ReplaceAll(content, "&#xA;", "\n")    // newline
	content = strings.ReplaceAll(content, "&amp;#39;", "'") // apostrophe
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&amp;", "&") // must be last
	content = strings.ReplaceAll(content, "&#34;", "\"")

	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Replace multiple spaces with single spaces but preserve structure
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	// Join lines back, but normalize multiple consecutive empty lines to single empty line
	result := ""
	prevEmpty := false
	for _, line := range lines {
		if line == "" {
			if !prevEmpty {
				result += "\n"
			}
			prevEmpty = true
		} else {
			if result != "" {
				result += "\n"
			}
			result += line
			prevEmpty = false
		}
	}

	return strings.TrimSpace(result)
}

// TestValidationSampleFormat tests that sample files are properly formatted
func TestValidationSampleFormat(t *testing.T) {
	validationDir := "../../validation_samples"

	if _, err := os.Stat(validationDir); os.IsNotExist(err) {
		t.Skipf("Validation samples directory not found: %s", validationDir)
		return
	}

	inputFiles, err := filepath.Glob(filepath.Join(validationDir, "*.in"))
	require.NoError(t, err)

	for _, inputFile := range inputFiles {
		baseName := strings.TrimSuffix(filepath.Base(inputFile), ".in")
		t.Run(baseName+"_format", func(t *testing.T) {
			// Test that input file is valid JSON
			inputData, err := os.ReadFile(inputFile)
			require.NoError(t, err, "Failed to read input file")

			var sample ValidationSample
			err = json.Unmarshal(inputData, &sample)
			require.NoError(t, err, "Input file is not valid JSON")

			// Validate required fields
			assert.NotZero(t, sample.UID, "UID is required")
			assert.NotEmpty(t, sample.Subject, "Subject is required")
			assert.NotEmpty(t, sample.From, "From is required")
			assert.NotEmpty(t, sample.Date, "Date is required")

			// At least one body field should be present
			assert.True(t, sample.TextBody != "" || sample.HTMLBody != "",
				"At least one of text_body or html_body must be present")

			// Test date format
			_, err = time.Parse(time.RFC3339, sample.Date)
			assert.NoError(t, err, "Date should be in RFC3339 format")

			t.Logf("✅ Format validation passed for %s", baseName)
		})
	}
}
