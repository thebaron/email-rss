package imap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClientTimeout(t *testing.T) {
	tests := []struct {
		name          string
		config        IMAPConfig
		expectedError bool
		skipReason    string
	}{
		{
			name: "custom timeout",
			config: IMAPConfig{
				Host:     "10.255.255.1", // Non-routable IP for timeout testing
				Port:     993,
				Username: "test@example.com",
				Password: "password",
				TLS:      false,
				Timeout:  1, // 1 second timeout
			},
			expectedError: true,
		},
		{
			name: "default timeout when zero",
			config: IMAPConfig{
				Host:     "10.255.255.1", // Non-routable IP for timeout testing
				Port:     993,
				Username: "test@example.com",
				Password: "password",
				TLS:      false,
				Timeout:  0, // Should default to 30 seconds
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			start := time.Now()
			client, err := NewClient(tt.config)
			elapsed := time.Since(start)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, client)

				if tt.config.Timeout > 0 {
					expectedTimeout := time.Duration(tt.config.Timeout) * time.Second
					assert.Less(t, elapsed, expectedTimeout+2*time.Second)
					assert.Greater(t, elapsed, expectedTimeout-1*time.Second)
				} else {
					assert.Less(t, elapsed, 17*time.Second)
					assert.Greater(t, elapsed, 13*time.Second)
				}
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

func TestIMAPConfigWithTimeout(t *testing.T) {
	config := IMAPConfig{
		Host:     "imap.example.com",
		Port:     993,
		Username: "test@example.com",
		Password: "password",
		TLS:      true,
		Timeout:  6,
	}

	assert.Equal(t, "imap.example.com", config.Host)
	assert.Equal(t, 993, config.Port)
	assert.Equal(t, "test@example.com", config.Username)
	assert.Equal(t, "password", config.Password)
	assert.True(t, config.TLS)
	assert.Equal(t, 6, config.Timeout)
}
