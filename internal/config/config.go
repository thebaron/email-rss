package config

import (
	"fmt"
	"log"

	"github.com/knadh/koanf/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

type Config struct {
	IMAP     IMAPConfig     `koanf:"imap" yaml:"imap"`
	Database DatabaseConfig `koanf:"database" yaml:"database"`
	RSS      RSSConfig      `koanf:"rss" yaml:"rss"`
	Server   ServerConfig   `koanf:"server" yaml:"server"`
}

type IMAPConfig struct {
	Host     string            `koanf:"host" yaml:"host"`
	Port     int               `koanf:"port" yaml:"port"`
	Username string            `koanf:"username" yaml:"username"`
	Password string            `koanf:"password" yaml:"password"`
	TLS      bool              `koanf:"tls" yaml:"tls"`
	Folders  map[string]string `koanf:"folders" yaml:"folders"`
}

type DatabaseConfig struct {
	Path string `koanf:"path" yaml:"path"`
}

type RSSConfig struct {
	OutputDir string `koanf:"output_dir" yaml:"output_dir"`
	Title     string `koanf:"title" yaml:"title"`
	BaseURL   string `koanf:"base_url" yaml:"base_url"`
}

type ServerConfig struct {
	Host string `koanf:"host" yaml:"host"`
	Port int    `koanf:"port" yaml:"port"`
}

func Load(configPath string) (*Config, error) {
	k := koanf.New(".")
	
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("error loading config file: %v", err)
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
	return nil
}