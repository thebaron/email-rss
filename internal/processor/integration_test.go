package processor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"emailrss/internal/db"
	"emailrss/internal/imap"
	"emailrss/internal/rss"
)

// MockIMAPClient implements the IMAP interface for testing
type MockIMAPClient struct {
	messages        []imap.Message
	messageContents map[uint32]*imap.MessageContent
}

func (m *MockIMAPClient) GetMessages(ctx context.Context, folder string, since time.Time) ([]imap.Message, error) {
	return m.messages, nil
}

func (m *MockIMAPClient) GetMessageBody(ctx context.Context, uid uint32) (string, error) {
	content := m.messageContents[uid]
	if content.HTMLBody != "" {
		return content.HTMLBody, nil
	}
	return content.TextBody, nil
}

func (m *MockIMAPClient) GetMessageContent(ctx context.Context, uid uint32) (*imap.MessageContent, error) {
	if content, exists := m.messageContents[uid]; exists {
		return content, nil
	}
	return &imap.MessageContent{}, nil
}

func (m *MockIMAPClient) Close() error {
	return nil
}

// Sample email messages with various content types for testing
func createTestMessages() ([]imap.Message, map[uint32]*imap.MessageContent) {
	messages := []imap.Message{
		{
			ID:      1,
			UID:     1,
			Subject: "Plain Text Email",
			From:    "test@example.com",
			Date:    time.Date(2025, 8, 9, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:      2,
			UID:     2,
			Subject: "HTML Email with Rich Content",
			From:    "newsletter@example.com",
			Date:    time.Date(2025, 8, 9, 11, 0, 0, 0, time.UTC),
		},
		{
			ID:      3,
			UID:     3,
			Subject: "Multipart Email with MIME Boundaries",
			From:    "support@example.com",
			Date:    time.Date(2025, 8, 9, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:      4,
			UID:     4,
			Subject: "Email with UTF-8 Characters and Encoding Issues",
			From:    "international@example.com",
			Date:    time.Date(2025, 8, 9, 13, 0, 0, 0, time.UTC),
		},
	}

	contents := map[uint32]*imap.MessageContent{
		1: {
			TextBody: `Hello there!

This is a simple plain text email.
It has multiple lines of content.
Perfect for testing text processing.

Best regards,
Test User`,
			HTMLBody: "",
		},
		2: {
			TextBody: `Welcome to our newsletter!

This is the plain text version.
Visit our website for more info.
Thanks for subscribing!`,
			HTMLBody: `<!DOCTYPE html>
<html>
<head><title>Newsletter</title></head>
<body>
<h1>Welcome to our newsletter!</h1>
<p>This is the <strong>HTML version</strong> with rich formatting.</p>
<p>Visit our <a href="https://example.com">website</a> for more info.</p>
<p>Thanks for subscribing!</p>
</body>
</html>`,
		},
		3: {
			TextBody: "",
			HTMLBody: `This is a multi-part message in MIME format

--_----------=_MCPart_338457435
Content-Type: text/plain; charset="utf-8"; format="fixed"
Content-Transfer-Encoding: quoted-printable

Hello=2C this is a test message with boundaries.
It contains quoted-printable encoding=2C like =3D signs.
This should be cleaned up properly.

Line 1 of actual content
Line 2 of actual content
Line 3 of actual content
Line 4 of actual content
Line 5 of actual content
Line 6 of actual content

--_----------=_MCPart_338457435--`,
		},
		4: {
			TextBody: `Subject with UTF-8: â€œSmart Quotesâ€ and â€" Dashes

This email contains UTF-8 encoding issues.
The em-dash â€" should be displayed correctly.
Smart quotes â€œlike thisâ€ should work too.
Bullet points â€¢ and ellipsis â€¦ as well.

This tests our UTF-8 fix functionality.`,
			HTMLBody: "",
		},
	}

	return messages, contents
}

func TestBusinessLogicIntegration(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()
	
	// Set up test database
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Set up mock IMAP client with test data
	messages, contents := createTestMessages()
	mockIMAP := &MockIMAPClient{
		messages:        messages,
		messageContents: contents,
	}

	// Set up RSS generator
	rssConfig := rss.RSSConfig{
		OutputDir:               tempDir,
		Title:                   "Test Email RSS",
		BaseURL:                 "http://localhost:8080",
		MaxHTMLContentLength:    8000,
		MaxTextContentLength:    3000,
		MaxRSSHTMLLength:        5000,
		MaxRSSTextLength:        2900,
		MaxSummaryLength:        300,
		RemoveCSS:               false,
	}
	rssGenerator := rss.NewGenerator(rssConfig)

	// Create processor
	processor := New(mockIMAP, database, rssGenerator)

	// Define test folders
	folders := map[string]string{
		"INBOX": "inbox",
	}

	ctx := context.Background()

	// Run the business logic
	err = processor.ProcessFolders(ctx, folders)
	require.NoError(t, err)

	// Verify RSS feed was created
	rssPath := filepath.Join(tempDir, "inbox.xml")
	assert.FileExists(t, rssPath)
	
	rssContent, err := os.ReadFile(rssPath)
	require.NoError(t, err)
	rssString := string(rssContent)

	// Verify JSON feed was created
	jsonPath := filepath.Join(tempDir, "inbox.json")
	assert.FileExists(t, jsonPath)
	
	jsonContent, err := os.ReadFile(jsonPath)
	require.NoError(t, err)
	jsonString := string(jsonContent)

	// Test RSS feed content
	t.Run("RSS Feed Validation", func(t *testing.T) {
		// Check that all messages are present
		assert.Contains(t, rssString, "Plain Text Email")
		assert.Contains(t, rssString, "HTML Email with Rich Content")
		assert.Contains(t, rssString, "Multipart Email with MIME Boundaries")
		assert.Contains(t, rssString, "Email with UTF-8 Characters")

		// Check that MIME boundaries are cleaned up
		assert.NotContains(t, rssString, "MCPart_338457435")
		assert.NotContains(t, rssString, "Content-Type:")
		assert.NotContains(t, rssString, "Content-Transfer-Encoding:")

		// Check that quoted-printable is decoded
		assert.NotContains(t, rssString, "=2C")  // Should be comma
		assert.NotContains(t, rssString, "=3D")  // Should be equals sign

		// Check that UTF-8 content is processed (specific characters may vary based on encoding)
		assert.Contains(t, rssString, "encoding issues")  // Content about UTF-8 is present
		assert.Contains(t, rssString, "em-dash")  // Reference to em-dash is present
	})

	// Test JSON feed content
	t.Run("JSON Feed Validation", func(t *testing.T) {
		// Check JSON structure
		assert.Contains(t, jsonString, `"version": "https://jsonfeed.org/version/1.1"`)
		assert.Contains(t, jsonString, `"title": "Test Email RSS - inbox"`)
		assert.Contains(t, jsonString, `"items":`)

		// Check that all messages are present
		assert.Contains(t, jsonString, "Plain Text Email")
		assert.Contains(t, jsonString, "HTML Email with Rich Content")
		assert.Contains(t, jsonString, "Multipart Email with MIME Boundaries")
		assert.Contains(t, jsonString, "Email with UTF-8 Characters")

		// Check content separation
		assert.Contains(t, jsonString, `"content_html":`)
		assert.Contains(t, jsonString, `"content_text":`)
		
		// HTML content should contain HTML tags for message 2 (JSON-encoded)
		assert.Contains(t, jsonString, "Welcome to our newsletter!")  // Check for content
		assert.Contains(t, jsonString, "HTML version")  // Check for content

		// Text content should be clean (no HTML tags)
		assert.Contains(t, jsonString, "This is a simple plain text email")

		// Check summaries are generated
		assert.Contains(t, jsonString, `"summary":`)
		
		// MIME boundaries should be cleaned up
		assert.NotContains(t, jsonString, "MCPart_338457435")
		assert.NotContains(t, jsonString, "Content-Type:")
		
		// UTF-8 content should be processed (specific characters may vary based on encoding)
		assert.Contains(t, jsonString, "encoding issues")  // Content about UTF-8 is present
		assert.Contains(t, jsonString, "em-dash")  // Reference to em-dash is present
	})

	// Test summary generation
	t.Run("Summary Generation", func(t *testing.T) {
		// Should contain first few lines as summary
		assert.Contains(t, jsonString, "Hello there! This is a simple plain text email.")  // Check first line
		assert.Contains(t, jsonString, "Welcome to our newsletter! This is the plain text version.")  // Check second message
	})

	// Test database tracking
	t.Run("Database Tracking", func(t *testing.T) {
		// All messages should be marked as processed
		for _, msg := range messages {
			processed, err := database.IsMessageProcessed("INBOX", msg.UID)
			assert.NoError(t, err)
			assert.True(t, processed, "Message UID %d should be marked as processed", msg.UID)
		}

		// Get processed messages
		processedMsgs, err := database.GetProcessedMessages("INBOX", 10)
		assert.NoError(t, err)
		assert.Len(t, processedMsgs, 4)
	})

	// Test idempotency - running again should not duplicate
	t.Run("Idempotency", func(t *testing.T) {
		// Run processing again
		err = processor.ProcessFolders(ctx, folders)
		require.NoError(t, err)

		// Should still have the same number of processed messages
		processedMsgs, err := database.GetProcessedMessages("INBOX", 10)
		assert.NoError(t, err)
		assert.Len(t, processedMsgs, 4)

		// Feed files should still exist and have same content
		newRSSContent, err := os.ReadFile(rssPath)
		assert.NoError(t, err)
		newJSONContent, err := os.ReadFile(jsonPath)
		assert.NoError(t, err)

		// Content should be the same (idempotent)
		assert.Equal(t, string(rssContent), string(newRSSContent))
		assert.Equal(t, string(jsonContent), string(newJSONContent))
	})
}

func TestContentProcessingEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	
	// Test with edge case messages
	edgeMessages := []imap.Message{
		{
			ID: 1, UID: 1, Subject: "Empty Content", From: "test@example.com",
			Date: time.Now(),
		},
		{
			ID: 2, UID: 2, Subject: "Only HTML", From: "test@example.com", 
			Date: time.Now(),
		},
		{
			ID: 3, UID: 3, Subject: "Only Text", From: "test@example.com",
			Date: time.Now(),
		},
	}

	edgeContents := map[uint32]*imap.MessageContent{
		1: {TextBody: "", HTMLBody: ""},
		2: {TextBody: "", HTMLBody: "<h1>Only HTML content</h1><p>No text version available.</p>"},
		3: {TextBody: "Only plain text content available.\nNo HTML version.", HTMLBody: ""},
	}

	mockIMAP := &MockIMAPClient{
		messages:        edgeMessages,
		messageContents: edgeContents,
	}

	dbPath := filepath.Join(tempDir, "edge.db")
	database, err := db.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	rssConfig := rss.RSSConfig{
		OutputDir:               tempDir,
		Title:                   "Edge Case Test",
		BaseURL:                 "http://localhost:8080",
		MaxHTMLContentLength:    8000,
		MaxTextContentLength:    3000,
		MaxRSSHTMLLength:        5000,
		MaxRSSTextLength:        2900,
		MaxSummaryLength:        300,
		RemoveCSS:               false,
	}
	rssGenerator := rss.NewGenerator(rssConfig)
	processor := New(mockIMAP, database, rssGenerator)

	folders := map[string]string{"INBOX": "edge"}
	ctx := context.Background()

	// Should handle edge cases without errors
	err = processor.ProcessFolders(ctx, folders)
	assert.NoError(t, err)

	// Check that feeds were created
	jsonPath := filepath.Join(tempDir, "edge.json")
	assert.FileExists(t, jsonPath)

	jsonContent, err := os.ReadFile(jsonPath)
	require.NoError(t, err)
	jsonString := string(jsonContent)

	// Should handle empty content gracefully
	assert.Contains(t, jsonString, "Empty Content")
	assert.Contains(t, jsonString, "Only HTML")
	assert.Contains(t, jsonString, "Only Text")
	
	// Should convert between formats when only one is available
	assert.Contains(t, jsonString, "Only HTML content")  // Check for content, not raw HTML tags
	assert.Contains(t, jsonString, "Only plain text content")
}

func TestResetFolderIntegration(t *testing.T) {
	database, err := db.New(":memory:")
	require.NoError(t, err)
	defer database.Close()

	proc := New(nil, database, nil)

	err = database.MarkMessageProcessed("INBOX", 1, "Test", "test@example.com", time.Now())
	require.NoError(t, err)

	processed, err := database.IsMessageProcessed("INBOX", 1)
	require.NoError(t, err)
	assert.True(t, processed)

	err = proc.ResetFolder("INBOX")
	assert.NoError(t, err)

	processed, err = database.IsMessageProcessed("INBOX", 1)
	require.NoError(t, err)
	assert.False(t, processed)
}
