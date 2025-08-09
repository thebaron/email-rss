package processor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"emailrss/internal/db"
	"emailrss/internal/rss"
)

func TestProcessFoldersIntegration(t *testing.T) {
	t.Skip("Integration test - requires IMAP setup")

	database, err := db.New(":memory:")
	require.NoError(t, err)
	defer database.Close()

	rssConfig := rss.RSSConfig{
		OutputDir: t.TempDir(),
		Title:     "Test RSS",
		BaseURL:   "http://localhost:8080",
	}
	rssGenerator := rss.NewGenerator(rssConfig)

	proc := New(nil, database, rssGenerator)

	folders := map[string]string{
		"TestFolder": "test",
	}

	err = proc.ProcessFolders(context.Background(), folders)
	assert.Error(t, err)
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
