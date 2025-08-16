package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	IMAP       IMAPConfig       `koanf:"imap" yaml:"imap"`
	Database   DatabaseConfig   `koanf:"database" yaml:"database"`
	RSS        RSSConfig        `koanf:"rss" yaml:"rss"`
	Server     ServerConfig     `koanf:"server" yaml:"server"`
	Debug      DebugConfig      `koanf:"debug" yaml:"debug"`
	Processing ProcessingConfig `koanf:"processing" yaml:"processing"`
}

type IMAPConfig struct {
	Host     string            `koanf:"host" yaml:"host"`
	Port     int               `koanf:"port" yaml:"port"`
	Username string            `koanf:"username" yaml:"username"`
	Password string            `koanf:"password" yaml:"password"`
	TLS      bool              `koanf:"tls" yaml:"tls"`
	Timeout  int               `koanf:"timeout" yaml:"timeout"`
	Folders  map[string]string `koanf:"folders" yaml:"folders"`
}

type DatabaseConfig struct {
	Path string `koanf:"path" yaml:"path"`
}

type RSSConfig struct {
	OutputDir            string `koanf:"output_dir" yaml:"output_dir"`
	Title                string `koanf:"title" yaml:"title"`
	BaseURL              string `koanf:"base_url" yaml:"base_url"`
	MaxHTMLContentLength int    `koanf:"max_html_content_length" yaml:"max_html_content_length"`
	MaxTextContentLength int    `koanf:"max_text_content_length" yaml:"max_text_content_length"`
	MaxRSSHTMLLength     int    `koanf:"max_rss_html_length" yaml:"max_rss_html_length"`
	MaxRSSTextLength     int    `koanf:"max_rss_text_length" yaml:"max_rss_text_length"`
	MaxSummaryLength     int    `koanf:"max_summary_length" yaml:"max_summary_length"`
	RemoveCSS            bool   `koanf:"remove_css" yaml:"remove_css"`
}

type ServerConfig struct {
	Host string `koanf:"host" yaml:"host"`
	Port int    `koanf:"port" yaml:"port"`
}

type DebugConfig struct {
	Enabled         bool   `koanf:"enabled" yaml:"enabled"`
	RawMessagesDir  string `koanf:"raw_messages_dir" yaml:"raw_messages_dir"`
	SaveRawMessages bool   `koanf:"save_raw_messages" yaml:"save_raw_messages"`
	MaxRawMessages  int    `koanf:"max_raw_messages" yaml:"max_raw_messages"`
}

type ProcessingConfig struct {
	MaxWorkers int `koanf:"max_workers" yaml:"max_workers"`
}

func Load(configPath string) (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("error loading config file: %v", err)
	}

	// Support loading config from environment variables (env vars take precedence over file)
	// Use koanf's env.Provider to load env vars with prefix "EMAILRSS_" and map to struct fields
	envProvider := env.Provider(".", env.Opt{
		Prefix: "EMAILRSS_",
		TransformFunc: func(k, v string) (string, any) {
			return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(k, "EMAILRSS_")), "_", "."), v
		},
	})
	if err := k.Load(envProvider, nil); err != nil {
		return nil, fmt.Errorf("error loading config from environment: %v", err)
	}

	var config Config
	if err := k.Unmarshal("", &config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %v", err)
	}

	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return &config, nil
}

func validate(config *Config) error {
	if config.IMAP.Host == "" {
		return fmt.Errorf("IMAP host is required")
	}
	if config.IMAP.Username == "" {
		return fmt.Errorf("IMAP username is required")
	}
	if config.IMAP.Password == "" {
		return fmt.Errorf("IMAP password is required")
	}
	if config.Database.Path == "" {
		config.Database.Path = "./emailrss.db"
		log.Printf("Using default database path: %s", config.Database.Path)
	}
	if config.RSS.OutputDir == "" {
		config.RSS.OutputDir = "./feeds"
		log.Printf("Using default RSS output directory: %s", config.RSS.OutputDir)
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.IMAP.Timeout == 0 {
		config.IMAP.Timeout = 30
		log.Printf("Using default IMAP timeout: %d seconds", config.IMAP.Timeout)
	}

	// Set default content length limits
	if config.RSS.MaxHTMLContentLength == 0 {
		config.RSS.MaxHTMLContentLength = 8000
	}
	if config.RSS.MaxTextContentLength == 0 {
		config.RSS.MaxTextContentLength = 3000
	}
	if config.RSS.MaxRSSHTMLLength == 0 {
		config.RSS.MaxRSSHTMLLength = 5000
	}
	if config.RSS.MaxRSSTextLength == 0 {
		config.RSS.MaxRSSTextLength = 2900
	}
	if config.RSS.MaxSummaryLength == 0 {
		config.RSS.MaxSummaryLength = 300
	}

	// Set default debug configuration values
	if config.Debug.RawMessagesDir == "" {
		config.Debug.RawMessagesDir = "./debug/raw_messages"
	}
	if config.Debug.MaxRawMessages == 0 {
		config.Debug.MaxRawMessages = 100 // Default to keeping last 100 raw messages
	}

	// Set default processing configuration values
	if config.Processing.MaxWorkers == 0 {
		config.Processing.MaxWorkers = 5 // Default to 5 concurrent workers
		log.Printf("Using default max workers: %d", config.Processing.MaxWorkers)
	}
	if config.Processing.MaxWorkers > 20 {
		config.Processing.MaxWorkers = 20 // Cap at 20 workers to avoid resource exhaustion
		log.Printf("Capped max workers to: %d", config.Processing.MaxWorkers)
	}

	return nil
}
