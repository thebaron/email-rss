package imap

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      IMAPConfig
		expectError bool
		skipReason  string
	}{
		{
			name: "invalid host",
			config: IMAPConfig{
				Host:     "nonexistent.invalid.domain",
				Port:     993,
				Username: "test@example.com",
				Password: "password",
				TLS:      true,
				Timeout:  1,
			},
			expectError: true,
		},
		{
			name: "invalid port",
			config: IMAPConfig{
				Host:     "localhost",
				Port:     99999,
				Username: "test@example.com",
				Password: "password",
				TLS:      false,
				Timeout:  1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			debugConfig := DebugConfig{
				Enabled:         false,
				RawMessagesDir:  "./debug",
				SaveRawMessages: false,
				MaxRawMessages:  100,
			}
			client, err := NewClient(tt.config, debugConfig)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				if client != nil {
					defer client.Close()
				}
			}
		})
	}
}

func TestIMAPConfig(t *testing.T) {
	config := IMAPConfig{
		Host:     "imap.example.com",
		Port:     993,
		Username: "test@example.com",
		Password: "password",
		TLS:      true,
		Timeout:  1,
	}

	assert.Equal(t, "imap.example.com", config.Host)
	assert.Equal(t, 993, config.Port)
	assert.Equal(t, "test@example.com", config.Username)
	assert.Equal(t, "password", config.Password)
	assert.True(t, config.TLS)
}

func TestClientClose(t *testing.T) {
	client := &Client{}

	err := client.Close()
	assert.NoError(t, err)
}

func TestListFolders_RequiresIntegration(t *testing.T) {
	t.Skip("Integration test - requires actual IMAP server")

	config := IMAPConfig{
		Host:     "imap.gmail.com",
		Port:     993,
		Username: "test@gmail.com",
		Password: "app-password",
		TLS:      true,
		Timeout:  1,
	}

	debugConfig := DebugConfig{
		Enabled:         false,
		RawMessagesDir:  "./debug",
		SaveRawMessages: false,
		MaxRawMessages:  100,
	}
	client, err := NewClient(config, debugConfig)
	if err != nil {
		t.Skip("Could not connect to IMAP server")
	}
	defer client.Close()

	ctx := context.Background()
	folders, err := client.ListFolders(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, folders)
	assert.Contains(t, folders, "INBOX")
}

func TestGetMessages_RequiresIntegration(t *testing.T) {
	t.Skip("Integration test - requires actual IMAP server")

	config := IMAPConfig{
		Host:     "imap.gmail.com",
		Port:     993,
		Username: "test@gmail.com",
		Password: "app-password",
		TLS:      true,
		Timeout:  1,
	}

	debugConfig := DebugConfig{
		Enabled:         false,
		RawMessagesDir:  "./debug",
		SaveRawMessages: false,
		MaxRawMessages:  100,
	}
	client, err := NewClient(config, debugConfig)
	if err != nil {
		t.Skip("Could not connect to IMAP server")
	}
	defer client.Close()

	ctx := context.Background()
	messages, err := client.GetMessages(ctx, "INBOX", time.Time{})

	assert.NoError(t, err)
	assert.IsType(t, []Message{}, messages)
}

func TestGetMessageBody_RequiresIntegration(t *testing.T) {
	t.Skip("Integration test - requires actual IMAP server and message UID")

	config := IMAPConfig{
		Host:     "imap.gmail.com",
		Port:     993,
		Username: "test@gmail.com",
		Password: "app-password",
		TLS:      true,
		Timeout:  1,
	}

	debugConfig := DebugConfig{
		Enabled:         false,
		RawMessagesDir:  "./debug",
		SaveRawMessages: false,
		MaxRawMessages:  100,
	}
	client, err := NewClient(config, debugConfig)
	if err != nil {
		t.Skip("Could not connect to IMAP server")
	}
	defer client.Close()

	ctx := context.Background()

	body, err := client.GetMessageBody(ctx, 1)

	if err != nil {
		t.Skip("Message UID 1 may not exist")
	}
	assert.IsType(t, "", body)
}

func TestConnectionErrors(t *testing.T) {
	tests := []struct {
		name   string
		config IMAPConfig
	}{
		{
			name: "connection timeout",
			config: IMAPConfig{
				Host:     "10.255.255.1",
				Port:     993,
				Username: "test@example.com",
				Password: "password",
				TLS:      true,
				Timeout:  1,
			},
		},
		{
			name: "invalid credentials",
			config: IMAPConfig{
				Host:     "imap.gmail.com",
				Port:     993,
				Username: "invalid@gmail.com",
				Password: "wrongpassword",
				TLS:      true,
				Timeout:  10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debugConfig := DebugConfig{
				Enabled:         false,
				RawMessagesDir:  "./debug",
				SaveRawMessages: false,
				MaxRawMessages:  100,
			}
			client, err := NewClient(tt.config, debugConfig)
			assert.Error(t, err)
			assert.Nil(t, client)
		})
	}
}

func TestNonTLSConnection(t *testing.T) {
	config := IMAPConfig{
		Host:     "nonexistent.example.com",
		Port:     143,
		Username: "test@example.com",
		Password: "password",
		TLS:      false,
		Timeout:  1,
	}

	debugConfig := DebugConfig{
		Enabled:         false,
		RawMessagesDir:  "./debug",
		SaveRawMessages: false,
		MaxRawMessages:  100,
	}
	client, err := NewClient(config, debugConfig)
	assert.Error(t, err)
	assert.Nil(t, client)
}
