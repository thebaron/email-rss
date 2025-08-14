# EmailRSS

[![CI](https://github.com/thebaron/email-rss/actions/workflows/ci.yml/badge.svg)](https://github.com/thebaron/email-rss/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/thebaron/email-rss/graph/badge.svg?token=G0H7CCPB3B)](https://codecov.io/gh/thebaron/email-rss)
[![Go Report Card](https://goreportcard.com/badge/github.com/thebaron/email-rss)](https://goreportcard.com/report/github.com/thebaron/email-rss)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Convert email folders into RSS feeds for reading in RSS readers.

## Features

- **Multi-folder support**: Each IMAP folder becomes its own RSS and JSON feed
- **Modern feed formats**: Generates both RSS/XML and JSON Feed 1.1 formats
- **Rich content processing**: Advanced email body processing with MIME cleaning and UTF-8 fixes
- **IMAP integration**: Secure authentication with configurable settings and timeouts
- **Web server**: Serves feeds over HTTP with proper MIME types
- **Deduplication**: Tracks processed messages to avoid duplicates
- **Container ready**: Docker and Kubernetes deployment support
- **CLI interface**: Process emails once or run continuously
- **History reset**: Reset folder processing history when needed

## Quick Start

1. Copy `config.example.yaml` to `config.yaml` and configure your IMAP settings
2. Build: `go build -o emailrss ./cmd/emailrss`
3. Process emails: `./emailrss process --once`
4. Start server: `./emailrss serve`
5. View feeds at:
   - RSS feeds: `http://localhost:8080/feeds/inbox.xml`
   - JSON feeds: `http://localhost:8080/feeds/inbox.json`
   - Feed listing: `http://localhost:8080`

## Configuration

Edit `config.yaml`:

```yaml
imap:
  host: "imap.gmail.com"
  port: 993
  username: "your-email@gmail.com"
  password: "your-app-password"
  tls: true
  folders:
    "INBOX": "inbox"
    "INBOX/Important": "important"

database:
  path: "./data/emailrss.db"

rss:
  output_dir: "./feeds"
  title: "Email RSS Feeds"
  base_url: "http://localhost:8080"
  # Content length limits (optional)
  max_html_content_length: 8000      # Max chars for processing HTML content
  max_text_content_length: 3000      # Max chars for processing text content
  max_rss_html_length: 5000          # Max chars for RSS HTML content
  max_rss_text_length: 2900          # Max chars for RSS text content
  max_summary_length: 300            # Max chars for item summaries
  # CSS removal (optional, default: false)
  remove_css: false                  # Remove CSS styling, HTML comments, and bgcolor attributes from HTML emails

server:
  host: "0.0.0.0"
  port: 8080

# Debug options (optional, all default to false/disabled)
debug:
  enabled: false                     # Enable debug mode
  raw_messages_dir: "./debug/raw_messages"  # Directory for raw IMAP messages
  save_raw_messages: false           # Save raw IMAP messages to disk
  max_raw_messages: 100              # Maximum number of raw messages to keep
```

## Commands

- `emailrss process`: Continuously process emails every 5 minutes
- `emailrss process --once`: Process emails once and exit
- `emailrss serve`: Start the RSS web server
- `emailrss reset FOLDER`: Reset processing history for a folder

## Docker Deployment

```bash
docker build -t emailrss .
docker run -v $(pwd)/config.yaml:/data/config.yaml -v $(pwd)/data:/data -p 8080:8080 emailrss
```

## Kubernetes Deployment

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/pvc.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/cronjob.yaml
```

## Architecture

- **IMAP Client**: Connects to email servers and fetches messages with timeout support
- **SQLite Database**: Tracks processed messages to prevent duplicates
- **Feed Generator**: Converts email messages to both RSS/XML and JSON Feed formats
- **Content Processor**: Advanced MIME cleaning, quoted-printable decoding, and UTF-8 fixes
- **Web Server**: Serves feeds with proper MIME types and health checks
- **CLI Interface**: kong-based command line interface
- **Configuration**: koanf-based YAML configuration management

## CI/CD & Quality

This project uses GitHub Actions for continuous integration with:

- **Automated Testing**: Unit and integration tests on every push/PR
- **Code Coverage**: Coverage reports uploaded to Codecov
- **Code Quality**: Comprehensive linting with golangci-lint
- **Build Verification**: Multi-platform build testing
- **Artifact Generation**: Automated binary and coverage report generation

Current test coverage: **65.6%** across core business logic components.

## Future Enhancements

The codebase includes hooks for AI-powered message summarization that can be implemented later.