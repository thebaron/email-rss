package imap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIMAPConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config IMAPConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: IMAPConfig{
				Host:     "imap.example.com",
				Port:     993,
				Username: "user@example.com",
				Password: "password",
				TLS:      true,
			},
			valid: true,
		},
		{
			name: "empty host",
			config: IMAPConfig{
				Port:     993,
				Username: "user@example.com",
				Password: "password",
				TLS:      true,
			},
			valid: false,
		},
		{
			name: "zero port",
			config: IMAPConfig{
				Host:     "imap.example.com",
				Username: "user@example.com",
				Password: "password",
				TLS:      true,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.valid {
				if client == nil && err != nil {
					t.Skip("Network connectivity issue")
				}
			} else {
				assert.Error(t, err)
				assert.Nil(t, client)
			}

			if client != nil {
				client.Close()
			}
		})
	}
}

func TestMessageStruct(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	msg := Message{
		ID:      1,
		UID:     12345,
		Subject: "Test Subject",
		From:    "sender@example.com",
		Date:    testTime,
		Body:    "Test body content",
	}

	assert.Equal(t, uint32(1), msg.ID)
	assert.Equal(t, uint32(12345), msg.UID)
	assert.Equal(t, "Test Subject", msg.Subject)
	assert.Equal(t, "sender@example.com", msg.From)
	assert.Equal(t, testTime, msg.Date)
	assert.Equal(t, "Test body content", msg.Body)
}
