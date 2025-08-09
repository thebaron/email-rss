package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLastProcessedDateParseError(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	_, err := database.conn.Exec(`INSERT INTO processed_messages (folder, uid, subject, from_addr, date) VALUES (?, ?, ?, ?, ?)`,
		"INBOX", 1, "Test", "test@example.com", "invalid-date-format")
	assert.NoError(t, err)

	lastDate, err := database.GetLastProcessedDate("INBOX")
	assert.Error(t, err)
	assert.True(t, lastDate.IsZero())
	assert.Contains(t, err.Error(), "failed to parse last processed date")
}
