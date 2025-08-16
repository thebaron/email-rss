package processor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"emailrss/internal/db"
	"emailrss/internal/imap"
	"emailrss/internal/rss"
)

// MockAsyncIMAPClient implements the IMAP interface for async testing
type MockAsyncIMAPClient struct {
	messages        []imap.Message
	messageContents map[uint32]*imap.MessageContent
	delay           time.Duration // Simulate network delay
}

func (m *MockAsyncIMAPClient) GetMessages(ctx context.Context, folder string, since time.Time) ([]imap.Message, error) {
	return m.messages, nil
}

func (m *MockAsyncIMAPClient) GetMessageContent(ctx context.Context, uid uint32) (*imap.MessageContent, error) {
	// Simulate network delay
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if content, exists := m.messageContents[uid]; exists {
		return content, nil
	}
	return &imap.MessageContent{}, nil
}

func TestAsyncMessageProcessing(t *testing.T) {
	// Create temporary database file for proper concurrency support
	tempFile := t.TempDir() + "/test.db"
	database, err := db.New(tempFile)
	require.NoError(t, err)
	defer database.Close()

	// Set up mock IMAP client with delay to test concurrency
	messages := []imap.Message{
		{ID: 1, UID: 1, Subject: "Message 1", From: "test1@example.com", Date: time.Now()},
		{ID: 2, UID: 2, Subject: "Message 2", From: "test2@example.com", Date: time.Now()},
		{ID: 3, UID: 3, Subject: "Message 3", From: "test3@example.com", Date: time.Now()},
		{ID: 4, UID: 4, Subject: "Message 4", From: "test4@example.com", Date: time.Now()},
		{ID: 5, UID: 5, Subject: "Message 5", From: "test5@example.com", Date: time.Now()},
	}

	contents := map[uint32]*imap.MessageContent{
		1: {TextBody: "Content 1", HTMLBody: "<p>Content 1</p>"},
		2: {TextBody: "Content 2", HTMLBody: "<p>Content 2</p>"},
		3: {TextBody: "Content 3", HTMLBody: "<p>Content 3</p>"},
		4: {TextBody: "Content 4", HTMLBody: "<p>Content 4</p>"},
		5: {TextBody: "Content 5", HTMLBody: "<p>Content 5</p>"},
	}

	mockIMAP := &MockAsyncIMAPClient{
		messages:        messages,
		messageContents: contents,
		delay:           50 * time.Millisecond, // Small delay to test concurrency
	}

	// Set up RSS generator
	rssConfig := rss.RSSConfig{
		OutputDir:            "/tmp",
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}
	rssGenerator := rss.NewGenerator(rssConfig)

	// Create processor with different worker counts
	t.Run("AsyncWithMultipleWorkers", func(t *testing.T) {
		processor := New(mockIMAP, database, rssGenerator)
		processor.SetMaxWorkers(3) // Use 3 workers for 5 messages

		ctx := context.Background()

		// Measure processing time
		start := time.Now()
		newMessages, err := processor.processMessagesAsync(ctx, "INBOX", messages)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Len(t, newMessages, 5)

		// With 3 workers and 50ms delay per message, processing should take around:
		// - Sequential: 5 * 50ms = 250ms
		// - Concurrent (3 workers): ceil(5/3) * 50ms = 2 * 50ms = 100ms
		// Allow some tolerance for test execution overhead
		assert.Less(t, duration, 200*time.Millisecond, "Async processing should be faster than sequential")

		// Verify all messages were processed
		for _, msg := range newMessages {
			assert.NotEmpty(t, msg.Subject)
			assert.NotEmpty(t, msg.From)
			assert.True(t, msg.UID > 0)
		}
	})

	t.Run("AsyncWithSingleWorker", func(t *testing.T) {
		// Clear database for fresh test
		err := database.ClearFolderHistory("INBOX2")
		require.NoError(t, err)

		processor := New(mockIMAP, database, rssGenerator)
		processor.SetMaxWorkers(1) // Use 1 worker (essentially sequential)

		ctx := context.Background()

		start := time.Now()
		newMessages, err := processor.processMessagesAsync(ctx, "INBOX2", messages)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Len(t, newMessages, 5)

		// With 1 worker, should take approximately sequential time
		assert.Greater(t, duration, 200*time.Millisecond, "Single worker should take longer")
	})
}

func TestAsyncFeedGeneration(t *testing.T) {
	tempDir := t.TempDir()

	// Set up RSS generator
	rssConfig := rss.RSSConfig{
		OutputDir:            tempDir,
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}
	rssGenerator := rss.NewGenerator(rssConfig)

	processor := New(nil, nil, rssGenerator) // Don't need IMAP or DB for this test

	testMessages := []rss.EmailMessage{
		{
			UID:      1,
			Subject:  "Test Message 1",
			From:     "test1@example.com",
			Date:     time.Now(),
			TextBody: "Test content 1",
			HTMLBody: "<p>Test content 1</p>",
		},
		{
			UID:      2,
			Subject:  "Test Message 2",
			From:     "test2@example.com",
			Date:     time.Now(),
			TextBody: "Test content 2",
			HTMLBody: "<p>Test content 2</p>",
		},
	}

	ctx := context.Background()

	// Test concurrent feed generation
	start := time.Now()
	err := processor.generateFeedsAsync(ctx, "INBOX", "test", testMessages)
	duration := time.Since(start)

	require.NoError(t, err)

	// Verify both feeds were created
	assert.FileExists(t, tempDir+"/test.xml")
	assert.FileExists(t, tempDir+"/test.json")

	t.Logf("Feed generation took: %v", duration)
}

func TestConcurrentFolderProcessing(t *testing.T) {
	// Create temporary database file for proper concurrency support
	tempFile := t.TempDir() + "/test.db"
	database, err := db.New(tempFile)
	require.NoError(t, err)
	defer database.Close()

	// Set up mock IMAP client
	messages := []imap.Message{
		{ID: 1, UID: 1, Subject: "Message 1", From: "test@example.com", Date: time.Now()},
		{ID: 2, UID: 2, Subject: "Message 2", From: "test@example.com", Date: time.Now()},
	}

	contents := map[uint32]*imap.MessageContent{
		1: {TextBody: "Content 1", HTMLBody: "<p>Content 1</p>"},
		2: {TextBody: "Content 2", HTMLBody: "<p>Content 2</p>"},
	}

	mockIMAP := &MockAsyncIMAPClient{
		messages:        messages,
		messageContents: contents,
		delay:           100 * time.Millisecond, // Longer delay to test folder concurrency
	}

	// Set up RSS generator
	tempDir := t.TempDir()
	rssConfig := rss.RSSConfig{
		OutputDir:            tempDir,
		Title:                "Test RSS",
		BaseURL:              "http://localhost:8080",
		MaxHTMLContentLength: 8000,
		MaxTextContentLength: 3000,
		MaxRSSHTMLLength:     5000,
		MaxRSSTextLength:     2900,
		MaxSummaryLength:     300,
		RemoveCSS:            false,
	}
	rssGenerator := rss.NewGenerator(rssConfig)

	processor := New(mockIMAP, database, rssGenerator)
	processor.SetMaxWorkers(2)

	folders := map[string]string{
		"INBOX":           "inbox",
		"INBOX/Important": "important",
		"INBOX/Work":      "work",
	}

	ctx := context.Background()

	// Test concurrent folder processing
	start := time.Now()
	err = processor.ProcessFolders(ctx, folders)
	duration := time.Since(start)

	require.NoError(t, err)

	// With concurrent processing, should be faster than sequential
	// Sequential would be: 3 folders * 2 messages * 100ms = 600ms
	// Concurrent should be faster
	assert.Less(t, duration, 500*time.Millisecond, "Concurrent folder processing should be faster")

	// Verify feeds were created for all folders
	assert.FileExists(t, tempDir+"/inbox.xml")
	assert.FileExists(t, tempDir+"/important.xml")
	assert.FileExists(t, tempDir+"/work.xml")

	t.Logf("Folder processing took: %v", duration)
}
