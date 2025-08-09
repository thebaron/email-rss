package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDefaults(t *testing.T) {
	config := &Config{
		IMAP: IMAPConfig{
			Host:     "imap.example.com",
			Username: "user@example.com",
			Password: "password123",
		},
	}

	err := validate(config)
	assert.NoError(t, err)

	assert.Equal(t, "./emailrss.db", config.Database.Path)
	assert.Equal(t, "./feeds", config.RSS.OutputDir)
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, 8080, config.Server.Port)
}

func TestValidateWithExistingValues(t *testing.T) {
	config := &Config{
		IMAP: IMAPConfig{
			Host:     "imap.example.com",
			Username: "user@example.com",
			Password: "password123",
		},
		Database: DatabaseConfig{
			Path: "/custom/db/path.db",
		},
		RSS: RSSConfig{
			OutputDir: "/custom/feeds",
		},
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 9090,
		},
	}

	err := validate(config)
	assert.NoError(t, err)

	assert.Equal(t, "/custom/db/path.db", config.Database.Path)
	assert.Equal(t, "/custom/feeds", config.RSS.OutputDir)
	assert.Equal(t, "127.0.0.1", config.Server.Host)
	assert.Equal(t, 9090, config.Server.Port)
}
