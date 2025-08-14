# Changelog

All notable changes to EmailRSS will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.1.0] - 2025-08-14

### Added
- **Debug mode functionality**: Comprehensive debugging capabilities for troubleshooting email processing
  - Raw IMAP message storage with timestamp-based file naming (`YYYYMMDD_HHMMSS_uid_12345.eml`)
  - Configurable debug directory location and automatic file cleanup
  - Optional raw message saving with configurable retention limits
  - Debug configuration validation and example configuration file
- **CSS and styling removal**: Advanced HTML content cleaning for cleaner RSS feeds
  - Complete CSS removal including `<style>` blocks, inline `style` attributes, `class` and `id` attributes
  - HTML comment removal (`<!-- ... -->`) including multiline comments
  - Background color attribute removal (`bgcolor`) from all HTML elements
  - Configurable via `remove_css` flag with comprehensive test coverage
- **Enhanced content length controls**: Fine-grained content size management
  - Separate limits for HTML content processing, text content processing, RSS HTML output, RSS text output, and summaries
  - All limits configurable via YAML configuration with sensible defaults
  - Proper content truncation with ellipsis indicators
- **Improved validation testing**: Robust input/output validation system
  - Support for `.in` and `.out` file pairs for testing email processing pipelines
  - JSON-based test sample format with comprehensive validation
  - Automatic test discovery and execution for regression testing

### Enhanced
- **Content processing pipeline**: Significantly improved email content handling
  - Better MIME multipart parsing with proper HTML/text separation
  - Enhanced UTF-8 character correction for international content
  - Improved summary generation using first 5 lines of text content
  - More robust quoted-printable decoding with edge case handling
- **Configuration management**: Expanded configuration options
  - Debug settings with comprehensive validation
  - Content length limits with backward compatibility
  - CSS removal options with clear documentation
  - Enhanced example configuration with detailed comments
- **Test coverage**: Comprehensive testing improvements
  - Debug functionality validation with mock file systems
  - CSS removal testing with 19 different scenarios
  - Integration tests for full content processing pipeline
  - Edge case testing for malformed HTML and MIME content

### Fixed
- **Integration test stability**: Resolved test failures in content processing pipeline
- **MIME processing edge cases**: Better handling of malformed or incomplete MIME messages
- **CSS removal robustness**: Improved handling of mixed quote styles and nested CSS elements
- **Configuration validation**: Enhanced error handling for invalid debug and content settings

### Technical Details
- **New configuration options**:
  ```yaml
  debug:
    enabled: false
    raw_messages_dir: "./debug/raw_messages"
    save_raw_messages: false
    max_raw_messages: 100
  rss:
    max_html_content_length: 8000
    max_text_content_length: 3000
    max_rss_html_length: 5000
    max_rss_text_length: 2900
    max_summary_length: 300
    remove_css: false
  ```
- **New debug file structure**: Organized debug output with automatic cleanup
- **Enhanced content processing**: Multi-stage CSS and comment removal with preservation of semantic HTML
- **Improved test infrastructure**: 24 additional test cases covering new functionality

## [v1.0.0] - 2025-08-09

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
✅ **Debug capabilities**: Raw message storage and debugging tools for troubleshooting  
✅ **CSS/styling removal**: Clean HTML content by removing CSS, comments, and bgcolor attributes  
✅ **Configurable content limits**: Fine-grained control over content processing and output sizes  

---

## Future Enhancements

✅ **AI message summarization**: Hooks are implemented and ready for AI-powered content summarization  
🔮 **Message body storage**: Store email bodies in database for historical feed generation  
🔮 **Feed customization**: User-configurable feed titles, descriptions, and formatting options  
🔮 **Incremental updates**: More efficient processing of only new messages  
🔮 **Web interface**: Optional web UI for configuration and feed management  
🔮 **Multiple protocols**: Support for POP3 and Exchange in addition to IMAP  
🔮 **Advanced CSS removal**: Support for removing additional styling attributes like `align`, `valign`, `width`, `height`  
🔮 **Content filtering**: Configurable content filtering based on keywords or patterns  
🔮 **Feed encryption**: Optional encryption of RSS feed content for sensitive email data  