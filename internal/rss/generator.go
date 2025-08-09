package rss

import (
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/feeds"
)

type Generator struct {
	config RSSConfig
}

type RSSConfig struct {
	OutputDir string
	Title     string
	BaseURL   string
}

type EmailMessage struct {
	UID     uint32
	Subject string
	From    string
	Date    time.Time
	Body    string
}

type AIHooks interface {
	SummarizeMessage(subject, body string) (string, error)
}

type stubAIHooks struct{}

func (s *stubAIHooks) SummarizeMessage(subject, body string) (string, error) {
	return body, nil
}

func NewGenerator(config RSSConfig) *Generator {
	return &Generator{
		config: config,
	}
}

func (g *Generator) GenerateFeed(folder, feedName string, messages []EmailMessage, aiHooks AIHooks) error {
	if aiHooks == nil {
		aiHooks = &stubAIHooks{}
	}

	feed := &feeds.Feed{
		Title:       fmt.Sprintf("%s - %s", g.config.Title, feedName),
		Link:        &feeds.Link{Href: g.config.BaseURL},
		Description: fmt.Sprintf("RSS feed for email folder: %s", folder),
		Created:     time.Now(),
	}

	for _, msg := range messages {
		log.Printf("Processing RSS item for UID %d, body length: %d", msg.UID, len(msg.Body))
		
		summary, err := aiHooks.SummarizeMessage(msg.Subject, msg.Body)
		if err != nil {
			log.Printf("AI summarization failed for UID %d: %v, using original body", msg.UID, err)
			summary = msg.Body
		}
		
		log.Printf("Summary for UID %d length: %d", msg.UID, len(summary))
		
		processedContent := g.processContent(summary)
		log.Printf("Processed content for UID %d length: %d", msg.UID, len(processedContent))

		item := &feeds.Item{
			Title:       msg.Subject,
			Link:        &feeds.Link{Href: fmt.Sprintf("%s/message/%d", g.config.BaseURL, msg.UID)},
			Description: processedContent,
			Author:      &feeds.Author{Name: msg.From, Email: msg.From},
			Created:     msg.Date,
			Id:          fmt.Sprintf("%s_%d", folder, msg.UID),
		}

		feed.Items = append(feed.Items, item)
	}

	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	feedPath := filepath.Join(g.config.OutputDir, fmt.Sprintf("%s.xml", feedName))

	rssXML, err := feed.ToRss()
	if err != nil {
		return fmt.Errorf("failed to generate RSS XML: %v", err)
	}

	if err := os.WriteFile(feedPath, []byte(rssXML), 0644); err != nil {
		return fmt.Errorf("failed to write RSS feed file: %v", err)
	}

	return nil
}

func (g *Generator) processContent(content string) string {
	log.Printf("processContent input: %d characters", len(content))
	
	if len(content) == 0 {
		log.Printf("processContent: empty content, returning empty string")
		return ""
	}
	
	// Clean MIME multipart headers and boundaries
	content = g.cleanMIMEContent(content)
	log.Printf("processContent: after MIME cleaning: %d characters", len(content))
	
	// Check if content is already HTML
	isHTML := strings.Contains(strings.ToLower(content), "<html") ||
		strings.Contains(strings.ToLower(content), "<body") ||
		strings.Contains(strings.ToLower(content), "<div") ||
		strings.Contains(strings.ToLower(content), "<p>")

	log.Printf("processContent: content detected as HTML: %v", isHTML)
	
	var result string
	if isHTML {
		// Content is already HTML, just truncate if needed
		if len(content) > 5000 { // Increase limit for HTML content
			result = content[:5000] + "..."
		} else {
			result = content
		}
		log.Printf("processContent: HTML content processed, output length: %d", len(result))
	} else {
		// Content is plain text, wrap in <pre> to preserve formatting
		escapedContent := html.EscapeString(content)
		result = fmt.Sprintf("<pre>%s</pre>", escapedContent)

		if len(result) > 3000 { // Increase limit since <pre> tags add overhead
			// Truncate the inner content, not the HTML tags
			innerContent := escapedContent
			if len(innerContent) > 2900 {
				innerContent = innerContent[:2900] + "..."
			}
			result = fmt.Sprintf("<pre>%s</pre>", innerContent)
		}
		log.Printf("processContent: text content processed, output length: %d", len(result))
	}
	
	return result
}

func (g *Generator) cleanMIMEContent(content string) string {
	lines := strings.Split(content, "\n")
	var cleanedLines []string
	var inMIMEHeaders bool = true
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip common MIME header patterns
		if inMIMEHeaders {
			// Skip MIME multipart introduction
			if strings.Contains(line, "This is a multi-part message in MIME format") ||
			   strings.Contains(line, "This is a multipart message in MIME format") {
				continue
			}
			
			// Skip MIME boundary markers
			if strings.HasPrefix(line, "--") && (strings.Contains(line, "=_") || strings.Contains(line, "_Part_")) {
				continue
			}
			
			// Skip Content-Type headers
			if strings.HasPrefix(line, "Content-Type:") {
				continue
			}
			
			// Skip Content-Transfer-Encoding headers
			if strings.HasPrefix(line, "Content-Transfer-Encoding:") {
				continue
			}
			
			// Skip Content-Disposition headers
			if strings.HasPrefix(line, "Content-Disposition:") {
				continue
			}
			
			// Skip charset and format specifications
			if strings.Contains(line, "charset=") || strings.Contains(line, "format=") {
				continue
			}
			
			// If we hit a line that's not empty and not a header, we're past the MIME headers
			if line != "" && !strings.Contains(line, ":") {
				inMIMEHeaders = false
				cleanedLines = append(cleanedLines, lines[i]) // Use original line with spacing
			}
		} else {
			// We're past headers, include all content but skip boundary markers
			if strings.HasPrefix(line, "--") && (strings.Contains(line, "=_") || strings.Contains(line, "_Part_")) {
				continue
			}
			cleanedLines = append(cleanedLines, lines[i]) // Use original line with spacing
		}
	}
	
	result := strings.Join(cleanedLines, "\n")
	
	// Decode quoted-printable encoding
	beforeDecode := len(result)
	result = g.decodeQuotedPrintable(result)
	log.Printf("cleanMIMEContent: quoted-printable decode: %d -> %d characters", beforeDecode, len(result))
	
	// Fix UTF-8 encoding issues
	beforeUTF8Fix := len(result)
	result = g.fixUTF8Encoding(result)
	log.Printf("cleanMIMEContent: UTF-8 fix: %d -> %d characters", beforeUTF8Fix, len(result))
	
	// Clean up excessive whitespace while preserving intentional line breaks
	result = strings.TrimSpace(result)
	
	return result
}

func (g *Generator) decodeQuotedPrintable(input string) string {
	// Handle soft line breaks (=\n or =\r\n)
	input = strings.ReplaceAll(input, "=\r\n", "")
	input = strings.ReplaceAll(input, "=\n", "")
	
	// Decode =XX hex sequences
	result := ""
	i := 0
	for i < len(input) {
		if i < len(input)-2 && input[i] == '=' {
			// Check if next two characters are hex digits
			hex := input[i+1 : i+3]
			if len(hex) == 2 && g.isHex(hex) {
				// Convert hex to ASCII
				if val, err := g.hexToInt(hex); err == nil {
					result += string(byte(val))
					i += 3
					continue
				}
			}
		}
		result += string(input[i])
		i++
	}
	
	return result
}

func (g *Generator) isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (g *Generator) hexToInt(hex string) (int, error) {
	result := 0
	for _, c := range hex {
		result *= 16
		if c >= '0' && c <= '9' {
			result += int(c - '0')
		} else if c >= 'A' && c <= 'F' {
			result += int(c - 'A' + 10)
		} else if c >= 'a' && c <= 'f' {
			result += int(c - 'a' + 10)
		} else {
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return result, nil
}

func (g *Generator) fixUTF8Encoding(input string) string {
	result := input
	
	// Most common UTF-8 encoding issues when UTF-8 is interpreted as Latin-1
	// Em dash: the mangled sequence "â€"" should become em dash
	result = strings.ReplaceAll(result, "\u00e2\u0080\u0094", "\u2014") // Em dash
	
	// En dash: different sequence "â€"" should become en dash  
	result = strings.ReplaceAll(result, "\u00e2\u0080\u0093", "\u2013") // En dash
	
	// Right single quotation mark
	result = strings.ReplaceAll(result, "\u00e2\u0080\u0099", "\u2019") // Right single quote
	
	// Left single quotation mark
	result = strings.ReplaceAll(result, "\u00e2\u0080\u0098", "\u2018") // Left single quote
	
	// Left double quotation mark
	result = strings.ReplaceAll(result, "\u00e2\u0080\u009c", "\u201c") // Left double quote
	
	// Right double quotation mark
	result = strings.ReplaceAll(result, "\u00e2\u0080\u009d", "\u201d") // Right double quote
	
	// Bullet
	result = strings.ReplaceAll(result, "\u00e2\u0080\u00a2", "\u2022") // Bullet
	
	// Horizontal ellipsis
	result = strings.ReplaceAll(result, "\u00e2\u0080\u00a6", "\u2026") // Ellipsis
	
	// Non-breaking space artifacts
	result = strings.ReplaceAll(result, "\u00c2\u00a0", " ")  // NBSP
	result = strings.ReplaceAll(result, "\u00c2", "")        // Standalone artifacts
	
	return result
}

func (g *Generator) GetFeedPath(feedName string) string {
	return filepath.Join(g.config.OutputDir, fmt.Sprintf("%s.xml", feedName))
}

func (g *Generator) FeedExists(feedName string) bool {
	feedPath := g.GetFeedPath(feedName)
	_, err := os.Stat(feedPath)
	return err == nil
}
