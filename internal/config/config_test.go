package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		validate    func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid config",
			configYAML: `
imap:
  host: "imap.example.com"
  port: 993
  username: "user@example.com"
  password: "password123"
  tls: true
  timeout: 45
  folders:
    "INBOX": "inbox"
    "Sent": "sent"

database:
  path: "/tmp/test.db"

rss:
  output_dir: "/tmp/feeds"
  title: "Test RSS"
  base_url: "http://localhost:8080"

server:
  host: "localhost"
  port: 8080
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "imap.example.com", cfg.IMAP.Host)
				assert.Equal(t, 993, cfg.IMAP.Port)
				assert.Equal(t, "user@example.com", cfg.IMAP.Username)
				assert.Equal(t, "password123", cfg.IMAP.Password)
				assert.True(t, cfg.IMAP.TLS)
				assert.Equal(t, 45, cfg.IMAP.Timeout)
				assert.Equal(t, "inbox", cfg.IMAP.Folders["INBOX"])
				assert.Equal(t, "sent", cfg.IMAP.Folders["Sent"])
				assert.Equal(t, "/tmp/test.db", cfg.Database.Path)
				assert.Equal(t, "/tmp/feeds", cfg.RSS.OutputDir)
				assert.Equal(t, "Test RSS", cfg.RSS.Title)
				assert.Equal(t, "http://localhost:8080", cfg.RSS.BaseURL)
				assert.Equal(t, "localhost", cfg.Server.Host)
				assert.Equal(t, 8080, cfg.Server.Port)
			},
		},
		{
			name: "minimal config with defaults",
			configYAML: `
imap:
  host: "imap.example.com"
  username: "user@example.com"
  password: "password123"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "imap.example.com", cfg.IMAP.Host)
				assert.Equal(t, "user@example.com", cfg.IMAP.Username)
				assert.Equal(t, "password123", cfg.IMAP.Password)
				assert.Equal(t, 30, cfg.IMAP.Timeout)
				assert.Equal(t, "./emailrss.db", cfg.Database.Path)
				assert.Equal(t, "./feeds", cfg.RSS.OutputDir)
				assert.Equal(t, "0.0.0.0", cfg.Server.Host)
				assert.Equal(t, 8080, cfg.Server.Port)
			},
		},
		{
			name: "missing host",
			configYAML: `
imap:
  username: "user@example.com"
  password: "password123"
`,
			expectError: true,
		},
		{
			name: "missing username",
			configYAML: `
imap:
  host: "imap.example.com"
  password: "password123"
`,
			expectError: true,
		},
		{
			name: "missing password",
			configYAML: `
imap:
  host: "imap.example.com"
  username: "user@example.com"
`,
			expectError: true,
		},
		{
			name:        "invalid YAML",
			configYAML:  `invalid: yaml: content: [`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempConfigFile(t, tt.configYAML)
			defer os.Remove(tmpFile)

			cfg, err := Load(tmpFile)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/file.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "error loading config file")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				IMAP: IMAPConfig{
					Host:     "imap.example.com",
					Username: "user@example.com",
					Password: "password123",
				},
			},
			expectError: false,
		},
		{
			name: "empty host",
			config: &Config{
				IMAP: IMAPConfig{
					Username: "user@example.com",
					Password: "password123",
				},
			},
			expectError: true,
			errorMsg:    "IMAP host is required",
		},
		{
			name: "empty username",
			config: &Config{
				IMAP: IMAPConfig{
					Host:     "imap.example.com",
					Password: "password123",
				},
			},
			expectError: true,
			errorMsg:    "IMAP username is required",
		},
		{
			name: "empty password",
			config: &Config{
				IMAP: IMAPConfig{
					Host:     "imap.example.com",
					Username: "user@example.com",
				},
			},
			expectError: true,
			errorMsg:    "IMAP password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func createTempConfigFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	return tmpFile
}
