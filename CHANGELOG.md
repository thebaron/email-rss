# Changelog

All notable changes to EmailRSS will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Comprehensive email body content processing**: RSS feeds now include full email message bodies with proper formatting
- **MIME multipart content cleaning**: Automatically removes MIME headers, boundaries, and encoding artifacts from email content
- **Quoted-printable decoding**: Converts encoded sequences like `=2C` (comma), `=20` (space), `=3D` (equals) to proper characters
- **UTF-8 encoding fixes**: Corrects mangled UTF-8 sequences like `â€"` → `—` (em dash), `â€™` → `'` (smart quote)
- **HTML and text content detection**: Automatically handles both HTML and plain text email bodies appropriately
- **Text formatting preservation**: Plain text emails wrapped in `<pre>` tags to maintain original spacing and formatting
- **Configurable IMAP timeout**: Added timeout configuration for IMAP connections with sensible defaults
- **Enhanced message body retrieval**: Improved IMAP client to fetch both text/plain and alternative body parts
- **Comprehensive unit test suite**: Added 65.6% test coverage across core business logic components
- **GitHub Actions CI/CD pipeline**: Automated testing, building, and linting on every commit and PR
- **Code quality tools**: Integrated golangci-lint with essential linters for code quality
- **Documentation badges**: Added CI status, test coverage, Go Report Card, and license badges
- **MIT License**: Added proper open source licensing
- **Containerization support**: Docker and Kubernetes deployment configurations
- **Debug logging**: Extensive logging for troubleshooting IMAP connections, message processing, and content conversion

### Enhanced
- **RSS feed generation**: Now includes rich email content with proper HTML formatting
- **Message deduplication**: Improved UID-based tracking to prevent duplicate messages in feeds
- **Error handling**: Enhanced error reporting with detailed context for debugging
- **Content processing pipeline**: Multi-stage content cleaning and formatting system
- **IMAP client robustness**: Better handling of various email server configurations and message formats

### Fixed
- **Empty RSS descriptions**: Resolved issue where message bodies weren't being included in generated feeds
- **UID fetching**: Fixed IMAP client to properly retrieve message UIDs instead of returning zeros
- **Continuous processing**: Fixed issue where `process` command (without `--once`) wouldn't run immediately on startup
- **MIME boundary artifacts**: Cleaned up multipart message formatting that was cluttering email content
- **Character encoding issues**: Fixed display of special characters, punctuation, and international text
- **Variable shadowing**: Resolved linter warnings about variable name conflicts
- **Code formatting**: Applied consistent Go formatting across entire codebase

### Technical Improvements
- **Modular architecture**: Well-organized package structure with clear separation of concerns
- **Configuration management**: Robust YAML-based configuration with validation and defaults
- **Database layer**: SQLite-based message tracking with proper schema migrations
- **HTTP server**: Clean REST-like interface for serving RSS feeds with health checks
- **CLI interface**: Kong-based command-line interface with clear help and error messages
- **Memory management**: Efficient processing of large email datasets
- **Concurrent processing**: Async message processing for improved performance

### Dependencies
- Updated to Go 1.24 for latest language features and performance improvements
- Added comprehensive dependency management with `go.mod` and `go.sum`
- Integrated with modern Go libraries:
  - `github.com/emersion/go-imap/v2` for robust IMAP communication
  - `github.com/gorilla/feeds` for RSS/Atom feed generation
  - `github.com/alecthomas/kong` for CLI argument parsing
  - `github.com/knadh/koanf/v2` for configuration management
  - `modernc.org/sqlite` for embedded database functionality
  - `github.com/stretchr/testify` for comprehensive testing

### Development Workflow
- **Continuous Integration**: GitHub Actions workflow with parallel test, build, and lint jobs
- **Code Quality Gates**: Automated linting, formatting checks, and test coverage reporting
- **Badge Integration**: Live status indicators for build health and code quality
- **Development Guidelines**: Comprehensive development philosophy and best practices documentation
- **Incremental Development**: Staged implementation approach with clear progress tracking

---

## Project Goals Achieved

✅ **Multi-folder RSS feeds**: Each IMAP folder becomes its own RSS feed  
✅ **Rich email content**: Full message bodies with proper formatting in RSS descriptions  
✅ **IMAP integration**: Secure, configurable authentication with timeout controls  
✅ **Web server**: HTTP server for RSS feed delivery with health endpoints  
✅ **Message deduplication**: Reliable tracking to prevent duplicate entries  
✅ **Container deployment**: Docker and Kubernetes ready with proper configurations  
✅ **CLI interface**: User-friendly command-line operations with help and error handling  
✅ **Content processing**: Advanced email content cleaning and formatting pipeline  
✅ **Production ready**: Comprehensive testing, CI/CD, and quality assurance  

---

## Future Enhancements

🔮 **AI message summarization**: Hooks are in place for future AI-powered content summarization  
🔮 **Message body storage**: Store email bodies in database for historical feed generation  
🔮 **Feed customization**: User-configurable feed titles, descriptions, and formatting options  
🔮 **Incremental updates**: More efficient processing of only new messages  
🔮 **Web interface**: Optional web UI for configuration and feed management  
🔮 **Multiple protocols**: Support for POP3 and Exchange in addition to IMAP  