package imap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugFunctionality(t *testing.T) {
	// Create a temporary directory for debug output
	tempDir := t.TempDir()

	// Create debug configuration with debug enabled
	debugConfig := DebugConfig{
		Enabled:         true,
		RawMessagesDir:  tempDir,
		SaveRawMessages: true,
		MaxRawMessages:  5,
	}

	// Create client instance
	client := &Client{
		debugConfig: debugConfig,
	}

	// Test saveRawMessage functionality
	testUID := uint32(12345)
	testFolder := "INBOX"
	testData := []byte("Test raw message data\nWith multiple lines\nAnd some content")

	err := client.saveRawMessage(testUID, testFolder, testData)
	assert.NoError(t, err)

	// Check that debug directory was created
	debugDir := filepath.Join(tempDir, testFolder)
	assert.DirExists(t, debugDir)

	// Check that file was created
	files, err := os.ReadDir(debugDir)
	require.NoError(t, err)
	assert.Len(t, files, 1)

	// Check filename format (should contain timestamp and UID)
	filename := files[0].Name()
	assert.True(t, strings.Contains(filename, "uid_12345"))
	assert.True(t, strings.HasSuffix(filename, ".eml"))

	// Check file contents
	filePath := filepath.Join(debugDir, filename)
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, testData, content)
}

func TestDebugFunctionalityDisabled(t *testing.T) {
	// Create a temporary directory for debug output
	tempDir := t.TempDir()

	// Create debug configuration with debug disabled
	debugConfig := DebugConfig{
		Enabled:         false,
		RawMessagesDir:  tempDir,
		SaveRawMessages: true, // This should be ignored since Enabled is false
		MaxRawMessages:  5,
	}

	// Create client instance
	client := &Client{
		debugConfig: debugConfig,
	}

	// Test saveRawMessage functionality
	testUID := uint32(12345)
	testFolder := "INBOX"
	testData := []byte("Test raw message data")

	err := client.saveRawMessage(testUID, testFolder, testData)
	assert.NoError(t, err)

	// Check that no debug directory was created
	debugDir := filepath.Join(tempDir, testFolder)
	assert.NoDirExists(t, debugDir)
}

func TestDebugSaveRawMessagesDisabled(t *testing.T) {
	// Create a temporary directory for debug output
	tempDir := t.TempDir()

	// Create debug configuration with debug enabled but save messages disabled
	debugConfig := DebugConfig{
		Enabled:         true,
		RawMessagesDir:  tempDir,
		SaveRawMessages: false, // This should prevent saving
		MaxRawMessages:  5,
	}

	// Create client instance
	client := &Client{
		debugConfig: debugConfig,
	}

	// Test saveRawMessage functionality
	testUID := uint32(12345)
	testFolder := "INBOX"
	testData := []byte("Test raw message data")

	err := client.saveRawMessage(testUID, testFolder, testData)
	assert.NoError(t, err)

	// Check that no debug directory was created
	debugDir := filepath.Join(tempDir, testFolder)
	assert.NoDirExists(t, debugDir)
}

func TestDebugCleanupOldMessages(t *testing.T) {
	// Create a temporary directory for debug output
	tempDir := t.TempDir()

	// Create debug configuration with low max messages for testing cleanup
	debugConfig := DebugConfig{
		Enabled:         true,
		RawMessagesDir:  tempDir,
		SaveRawMessages: true,
		MaxRawMessages:  3, // Keep only 3 messages
	}

	// Create client instance
	client := &Client{
		debugConfig: debugConfig,
	}

	testFolder := "INBOX"
	testData := []byte("Test raw message data")

	// Save 5 messages (exceeds the limit of 3)
	for i := 1; i <= 5; i++ {
		err := client.saveRawMessage(uint32(i), testFolder, testData)
		assert.NoError(t, err)
	}

	// Check that debug directory exists
	debugDir := filepath.Join(tempDir, testFolder)
	assert.DirExists(t, debugDir)

	// Check that only 3 files remain (due to cleanup)
	files, err := os.ReadDir(debugDir)
	require.NoError(t, err)
	assert.Len(t, files, 3)

	// Verify all remaining files are .eml files
	for _, file := range files {
		assert.True(t, strings.HasSuffix(file.Name(), ".eml"))
	}
}

func TestDebugConfig(t *testing.T) {
	debugConfig := DebugConfig{
		Enabled:         true,
		RawMessagesDir:  "./debug/raw_messages",
		SaveRawMessages: true,
		MaxRawMessages:  100,
	}

	assert.True(t, debugConfig.Enabled)
	assert.Equal(t, "./debug/raw_messages", debugConfig.RawMessagesDir)
	assert.True(t, debugConfig.SaveRawMessages)
	assert.Equal(t, 100, debugConfig.MaxRawMessages)
}
