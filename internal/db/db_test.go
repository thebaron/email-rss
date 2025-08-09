package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		dbPath      string
		expectError bool
	}{
		{
			name:        "valid path",
			dbPath:      ":memory:",
			expectError: false,
		},
		{
			name:        "file path",
			dbPath:      filepath.Join(t.TempDir(), "test.db"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := New(tt.dbPath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, db)
				defer db.Close()
			}
		})
	}
}

func TestMigrate(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	row := db.conn.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='processed_messages';`)
	var tableName string
	err = row.Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "processed_messages", tableName)

	rows, err := db.conn.Query(`PRAGMA index_list(processed_messages);`)
	require.NoError(t, err)
	defer rows.Close()

	indexCount := 0
	for rows.Next() {
		indexCount++
	}
	assert.GreaterOrEqual(t, indexCount, 2)
}

func TestMarkMessageProcessed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	err := db.MarkMessageProcessed("INBOX", 12345, "Test Subject", "test@example.com", testTime)
	assert.NoError(t, err)

	err = db.MarkMessageProcessed("INBOX", 12345, "Updated Subject", "updated@example.com", testTime.Add(time.Hour))
	assert.NoError(t, err)

	var count int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM processed_messages WHERE folder = ? AND uid = ?`, "INBOX", 12345).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	var subject string
	err = db.conn.QueryRow(`SELECT subject FROM processed_messages WHERE folder = ? AND uid = ?`, "INBOX", 12345).Scan(&subject)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Subject", subject)
}

func TestIsMessageProcessed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	processed, err := db.IsMessageProcessed("INBOX", 12345)
	assert.NoError(t, err)
	assert.False(t, processed)

	err = db.MarkMessageProcessed("INBOX", 12345, "Test Subject", "test@example.com", testTime)
	assert.NoError(t, err)

	processed, err = db.IsMessageProcessed("INBOX", 12345)
	assert.NoError(t, err)
	assert.True(t, processed)

	processed, err = db.IsMessageProcessed("INBOX", 54321)
	assert.NoError(t, err)
	assert.False(t, processed)

	processed, err = db.IsMessageProcessed("Sent", 12345)
	assert.NoError(t, err)
	assert.False(t, processed)
}

func TestGetProcessedMessages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	for i := 1; i <= 5; i++ {
		err := db.MarkMessageProcessed("INBOX", uint32(i),
			"Subject "+string(rune('0'+i)),
			"user"+string(rune('0'+i))+"@example.com",
			baseTime.Add(time.Duration(i)*time.Hour))
		assert.NoError(t, err)
	}

	for i := 1; i <= 3; i++ {
		err := db.MarkMessageProcessed("Sent", uint32(i),
			"Sent Subject "+string(rune('0'+i)),
			"sender"+string(rune('0'+i))+"@example.com",
			baseTime.Add(time.Duration(i)*time.Hour))
		assert.NoError(t, err)
	}

	messages, err := db.GetProcessedMessages("INBOX", 3)
	assert.NoError(t, err)
	assert.Len(t, messages, 3)

	assert.Equal(t, uint32(5), messages[0].UID)
	assert.Equal(t, "Subject 5", messages[0].Subject)
	assert.Equal(t, "user5@example.com", messages[0].From)

	assert.Equal(t, uint32(4), messages[1].UID)
	assert.Equal(t, uint32(3), messages[2].UID)

	messages, err = db.GetProcessedMessages("Sent", 10)
	assert.NoError(t, err)
	assert.Len(t, messages, 3)

	messages, err = db.GetProcessedMessages("NonExistent", 10)
	assert.NoError(t, err)
	assert.Len(t, messages, 0)
}

func TestGetLastProcessedDate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	lastDate, err := db.GetLastProcessedDate("INBOX")
	assert.NoError(t, err)
	assert.True(t, lastDate.IsZero())

	testTime1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	testTime2 := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)
	testTime3 := time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC)

	err = db.MarkMessageProcessed("INBOX", 1, "Subject 1", "user1@example.com", testTime1)
	assert.NoError(t, err)

	err = db.MarkMessageProcessed("INBOX", 2, "Subject 2", "user2@example.com", testTime3)
	assert.NoError(t, err)

	err = db.MarkMessageProcessed("INBOX", 3, "Subject 3", "user3@example.com", testTime2)
	assert.NoError(t, err)

	lastDate, err = db.GetLastProcessedDate("INBOX")
	assert.NoError(t, err)
	assert.True(t, testTime3.Equal(lastDate))

	lastDate, err = db.GetLastProcessedDate("NonExistent")
	assert.NoError(t, err)
	assert.True(t, lastDate.IsZero())
}

func TestClearFolderHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	err := db.MarkMessageProcessed("INBOX", 1, "Subject 1", "user1@example.com", testTime)
	assert.NoError(t, err)
	err = db.MarkMessageProcessed("INBOX", 2, "Subject 2", "user2@example.com", testTime)
	assert.NoError(t, err)
	err = db.MarkMessageProcessed("Sent", 1, "Sent Subject 1", "sender1@example.com", testTime)
	assert.NoError(t, err)

	processed, err := db.IsMessageProcessed("INBOX", 1)
	assert.NoError(t, err)
	assert.True(t, processed)

	processed, err = db.IsMessageProcessed("Sent", 1)
	assert.NoError(t, err)
	assert.True(t, processed)

	err = db.ClearFolderHistory("INBOX")
	assert.NoError(t, err)

	processed, err = db.IsMessageProcessed("INBOX", 1)
	assert.NoError(t, err)
	assert.False(t, processed)

	processed, err = db.IsMessageProcessed("INBOX", 2)
	assert.NoError(t, err)
	assert.False(t, processed)

	processed, err = db.IsMessageProcessed("Sent", 1)
	assert.NoError(t, err)
	assert.True(t, processed)

	err = db.ClearFolderHistory("NonExistent")
	assert.NoError(t, err)
}

func TestClose(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)

	err = db.Close()
	assert.NoError(t, err)

	err = db.Close()
	assert.NoError(t, err)
}

func TestCloseNilConnection(t *testing.T) {
	db := &DB{}
	err := db.Close()
	assert.NoError(t, err)
}

func setupTestDB(t *testing.T) *DB {
	db, err := New(":memory:")
	require.NoError(t, err)
	return db
}
