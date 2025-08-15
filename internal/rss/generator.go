package rss

import (
	"encoding/json"
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
	OutputDir            string
	Title                string
	BaseURL              string
	MaxHTMLContentLength int
	MaxTextContentLength int
	MaxRSSHTMLLength     int
	MaxRSSTextLength     int
	MaxSummaryLength     int
	RemoveCSS            bool
}

type EmailMessage struct {
	UID      uint32
	Subject  string
	From     string
	Date     time.Time
	TextBody string
	HTMLBody string
}

type AIHooks interface {
	SummarizeMessage(subject, body string) (string, error)
}

type stubAIHooks struct{}

func (s *stubAIHooks) SummarizeMessage(subject, body string) (string, error) {
	return body, nil
}

// JSON Feed structures according to version 1.1 specification
type JSONFeed struct {
	Version     string     `json:"version"`
	Title       string     `json:"title"`
	HomePageURL string     `json:"home_page_url,omitempty"`
	FeedURL     string     `json:"feed_url,omitempty"`
	Description string     `json:"description,omitempty"`
	Items       []JSONItem `json:"items"`
}

type JSONItem struct {
	ID            string   `json:"id"`
	URL           string   `json:"url,omitempty"`
	Title         string   `json:"title,omitempty"`
	ContentHTML   string   `json:"content_html,omitempty"`
	ContentText   string   `json:"content_text,omitempty"`
	Summary       string   `json:"summary,omitempty"`
	DatePublished string   `json:"date_published,omitempty"`
	Authors       []Author `json:"authors,omitempty"`
}

type Author struct {
	Name string `json:"name,omitempty"`
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
		log.Printf("Processing RSS item for UID %d, text: %d chars, html: %d chars",
			msg.UID, len(msg.TextBody), len(msg.HTMLBody))

		// Choose the best content for RSS (prefer HTML if available)
		var contentForSummary string
		if msg.HTMLBody != "" {
			contentForSummary = msg.HTMLBody
		} else {
			contentForSummary = msg.TextBody
		}

		summary, err := aiHooks.SummarizeMessage(msg.Subject, contentForSummary)
		if err != nil {
			log.Printf("AI summarization failed for UID %d: %v, using original content", msg.UID, err)
			summary = contentForSummary
		}

		// log.Printf("Summary for UID %d length: %d", msg.UID, len(summary))

		processedContent := g.processContent(summary)
		// log.Printf("Processed content for UID %d length: %d", msg.UID, len(processedContent))

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

func (g *Generator) GenerateJSONFeed(folder, feedName string, messages []EmailMessage, aiHooks AIHooks) error {
	if aiHooks == nil {
		aiHooks = &stubAIHooks{}
	}

	jsonFeed := JSONFeed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       fmt.Sprintf("%s - %s", g.config.Title, feedName),
		HomePageURL: g.config.BaseURL,
		FeedURL:     fmt.Sprintf("%s/%s.json", g.config.BaseURL, feedName),
		Description: fmt.Sprintf("JSON feed for email folder: %s", folder),
		Items:       []JSONItem{},
	}

	for _, msg := range messages {
		log.Printf("Processing JSON feed item for UID %d, text: %d chars, html: %d chars",
			msg.UID, len(msg.TextBody), len(msg.HTMLBody))

		var contentHTML, contentText string

		// Process HTML content if available
		if msg.HTMLBody != "" {
			summary, err := aiHooks.SummarizeMessage(msg.Subject, msg.HTMLBody)
			if err != nil {
				log.Printf("AI summarization failed for HTML UID %d: %v, using original", msg.UID, err)
				summary = msg.HTMLBody
			}
			contentHTML = g.processHTMLContent(summary)
		}

		// Process text content if available
		if msg.TextBody != "" {
			summary, err := aiHooks.SummarizeMessage(msg.Subject, msg.TextBody)
			if err != nil {
				log.Printf("AI summarization failed for text UID %d: %v, using original", msg.UID, err)
				summary = msg.TextBody
			}
			contentText = g.processTextContent(summary)
		}

		// If we only have one type, derive the other
		if contentHTML == "" && contentText != "" {
			contentHTML = fmt.Sprintf("<pre>%s</pre>", html.EscapeString(contentText))
		} else if contentText == "" && contentHTML != "" {
			contentText = g.stripHTML(contentHTML)
		}

		// log.Printf("Final content for UID %d - HTML: %d chars, Text: %d chars",
		// 	msg.UID, len(contentHTML), len(contentText))

		item := JSONItem{
			ID:            fmt.Sprintf("%s_%d", folder, msg.UID),
			URL:           fmt.Sprintf("%s/message/%d", g.config.BaseURL, msg.UID),
			Title:         msg.Subject,
			ContentHTML:   contentHTML,
			ContentText:   contentText,
			DatePublished: msg.Date.Format(time.RFC3339),
			Authors:       []Author{{Name: msg.From}},
		}

		// Create summary from first 5 lines of text content
		if contentText != "" {
			item.Summary = g.createSummaryFromText(contentText)
		}

		jsonFeed.Items = append(jsonFeed.Items, item)
	}

	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	feedPath := filepath.Join(g.config.OutputDir, fmt.Sprintf("%s.json", feedName))

	jsonData, err := json.MarshalIndent(jsonFeed, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate JSON feed: %v", err)
	}

	if err := os.WriteFile(feedPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON feed file: %v", err)
	}

	log.Printf("Generated JSON feed with %d items at %s", len(jsonFeed.Items), feedPath)
	return nil
}

func (g *Generator) processContent(content string) string {
	// log.Printf("processContent input: %d characters", len(content))

	if len(content) == 0 {
		log.Printf("processContent: empty content, returning empty string")
		return ""
	}

	// Check if content is already HTML
	isHTML := strings.Contains(strings.ToLower(content), "<html") ||
		strings.Contains(strings.ToLower(content), "<body") ||
		strings.Contains(strings.ToLower(content), "<div") ||
		strings.Contains(strings.ToLower(content), "<p>")

	log.Printf("processContent: content detected as HTML: %v", isHTML)

	var result string
	if isHTML {
		// Content is already HTML, remove CSS if configured
		processedHTML := g.removeCSS(content)

		// Truncate if needed
		if len(processedHTML) > g.config.MaxRSSHTMLLength {
			result = processedHTML[:g.config.MaxRSSHTMLLength] + "..."
		} else {
			result = processedHTML
		}
		// log.Printf("processContent: HTML content processed, output length: %d", len(result))
	} else {
		// Content is plain text, wrap in <pre> to preserve formatting
		escapedContent := html.EscapeString(content)
		result = fmt.Sprintf("<pre>%s</pre>", escapedContent)

		maxTotalLength := g.config.MaxRSSTextLength + 11 // Add overhead for <pre></pre> tags
		if len(result) > maxTotalLength {
			// Truncate the inner content, not the HTML tags
			innerContent := escapedContent
			if len(innerContent) > g.config.MaxRSSTextLength {
				innerContent = innerContent[:g.config.MaxRSSTextLength] + "..."
			}
			result = fmt.Sprintf("<pre>%s</pre>", innerContent)
		}
		// log.Printf("processContent: text content processed, output length: %d", len(result))
	}

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
	result = strings.ReplaceAll(result, "\u00c2\u00a0", " ") // NBSP
	result = strings.ReplaceAll(result, "\u00c2", "")        // Standalone artifacts

	return result
}

func (g *Generator) processHTMLContent(content string) string {
	// log.Printf("processHTMLContent input: %d characters", len(content))

	if len(content) == 0 {
		return ""
	}

	// Remove CSS if configured
	content = g.removeCSS(content)

	// For HTML content, we want to preserve the HTML but ensure it's valid
	if len(content) > g.config.MaxHTMLContentLength {
		content = content[:g.config.MaxHTMLContentLength] + "..."
	}

	// log.Printf("processHTMLContent output: %d characters", len(content))
	return content
}

func (g *Generator) processTextContent(content string) string {
	// log.Printf("processTextContent input: %d characters", len(content))

	if len(content) == 0 {
		return ""
	}

	// For plain text, we don't need HTML escaping since it will be in content_text
	if len(content) > g.config.MaxTextContentLength {
		content = content[:g.config.MaxTextContentLength] + "..."
	}

	// log.Printf("processTextContent output: %d characters", len(content))
	return content
}

func (g *Generator) stripHTML(html string) string {
	// Simple HTML tag removal for plain text content
	result := html

	// Remove <pre> tags but keep content
	result = strings.ReplaceAll(result, "<pre>", "")
	result = strings.ReplaceAll(result, "</pre>", "")

	// Remove other common HTML tags
	result = strings.ReplaceAll(result, "<br>", "\n")
	result = strings.ReplaceAll(result, "<br/>", "\n")
	result = strings.ReplaceAll(result, "<br />", "\n")

	// Remove any remaining HTML tags using a simple approach
	for strings.Contains(result, "<") && strings.Contains(result, ">") {
		start := strings.Index(result, "<")
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}

	// Clean up excessive whitespace
	result = strings.TrimSpace(result)

	return result
}

func (g *Generator) createSummaryFromText(text string) string {
	if text == "" {
		return ""
	}

	// Split into lines and take first 5 non-empty lines
	lines := strings.Split(text, "\n")
	var summaryLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// Skip empty lines
		if trimmedLine != "" {
			summaryLines = append(summaryLines, trimmedLine)
			// Stop after 5 lines
			if len(summaryLines) >= 5 {
				break
			}
		}
	}

	// Join the lines back together
	summary := strings.Join(summaryLines, " ")

	// Add ellipsis if we truncated or if the summary is very long
	if len(summaryLines) >= 5 || len(summary) > g.config.MaxSummaryLength {
		if len(summary) > g.config.MaxSummaryLength {
			summary = summary[:g.config.MaxSummaryLength]
		}
		summary += "..."
	}

	return summary
}

// removeCSS removes CSS styles from HTML content
func (g *Generator) removeCSS(htmlContent string) string {
	if !g.config.RemoveCSS {
		return htmlContent
	}

	log.Printf("Removing CSS from HTML content (%d chars)", len(htmlContent))

	// Remove <style> tags and their content
	result := htmlContent

	// Remove <style>...</style> blocks (case insensitive, multiline)
	for {
		styleStart := strings.Index(strings.ToLower(result), "<style")
		if styleStart == -1 {
			styleStart = strings.Index(strings.ToLower(result), `\u003cstyle`)
			if styleStart == -1 {
				break
			}
		}

		// Find the end of the opening <style> tag
		tagEnd := strings.Index(result[styleStart:], ">")
		if tagEnd == -1 {
			tagEnd = strings.Index(strings.ToLower(result), `\u003e`)
			if tagEnd == -1 {
				break
			}
		}
		tagEnd += styleStart + 1

		// Find the closing </style> tag
		styleEnd := strings.Index(strings.ToLower(result[tagEnd:]), "</style>")
		tagEndLen := 8
		if styleEnd == -1 {
			styleEnd = strings.Index(strings.ToLower(result), `\u003c/style\003e`)
			tagEndLen = 18
			if styleEnd == -1 {
				// No closing tag found, remove from opening tag to end
				result = result[:styleStart] + result[tagEnd:]
				break
			}
		}

		styleEnd += tagEnd + tagEndLen // +8 for "</style>", +18 for "\u003c/style\003e"
		result = result[:styleStart] + result[styleEnd:]
	}

	// Remove inline style attributes from HTML tags
	// This regex matches style="..." attributes
	for {
		styleAttrStart := strings.Index(strings.ToLower(result), " style=")
		if styleAttrStart == -1 {
			break
		}

		// Find the quote character (either " or ')
		quoteStart := styleAttrStart + 7 // 7 = len(" style=")
		if quoteStart >= len(result) {
			break
		}

		quoteChar := result[quoteStart]
		if quoteChar != '"' && quoteChar != '\'' {
			// No quote found, skip this occurrence
			result = result[:styleAttrStart] + result[styleAttrStart+1:]
			continue
		}

		// Find the closing quote
		quoteEnd := strings.Index(result[quoteStart+1:], string(quoteChar))
		lineEnd := -1
		if quoteEnd == -1 {
			// No closing quote found, remove from style= to end of line
			lineEnd = strings.Index(result[styleAttrStart:], ">")
			if lineEnd == -1 {
				lineEnd = strings.Index(result[styleAttrStart:], "\u003e")
				if lineEnd == -1 {
					result = result[:styleAttrStart]
				}
			}
		}

		if lineEnd != -1 {
			result = result[:styleAttrStart] + result[styleAttrStart+lineEnd:]
			break
		}

		quoteEnd += quoteStart + 1 + 1 // +1 for closing quote
		result = result[:styleAttrStart] + result[quoteEnd:]
	}

	// Remove common CSS-related attributes and presentation attributes
	cssAttributes := []string{
		" class=",
		" id=",
		" bgcolor=",
		" width=",
		" height=",
		" align=",
		" valign=",
		" border=",
		" cellpadding=",
		" cellspacing=",
		" color=",
		" face=",
		" size=",
		" charset=",
	}

	for _, attr := range cssAttributes {
		for {
			attrStart := strings.Index(strings.ToLower(result), attr)
			if attrStart == -1 {
				break
			}

			// Find the quote character
			quoteStart := attrStart + len(attr)
			if quoteStart >= len(result) {
				break
			}

			quoteChar := result[quoteStart]
			if quoteChar != '"' && quoteChar != '\'' {
				// No quote found, skip this occurrence
				result = result[:attrStart] + result[attrStart+1:]
				continue
			}

			// Find the closing quote
			quoteEnd := strings.Index(result[quoteStart+1:], string(quoteChar))
			if quoteEnd == -1 {
				// No closing quote found, remove from attribute to end of tag
				tagEnd := strings.Index(result[attrStart:], ">")
				if tagEnd == -1 {
					result = result[:attrStart]
				} else {
					result = result[:attrStart] + result[attrStart+tagEnd:]
				}
				break
			}

			quoteEnd += quoteStart + 1 + 1 // +1 for closing quote
			result = result[:attrStart] + result[quoteEnd:]
		}
	}

	// Remove HTML comments
	for {
		commentStart := strings.Index(result, "<!--")
		if commentStart == -1 {
			break
		}

		commentEnd := strings.Index(result[commentStart:], "-->")
		if commentEnd == -1 {
			// No closing comment tag found, remove from comment start to end
			result = result[:commentStart]
			break
		}

		commentEnd += commentStart + 3 // +3 for "-->"
		result = result[:commentStart] + result[commentEnd:]
	}

	// Clean up any orphaned attribute values that might be left behind
	// This handles cases where MIME processing may have left orphaned parts
	orphanedPatterns := []string{
		`="utf-8"`,
		`="UTF-8"`,
		`="text/html"`,
		`="text/css"`,
		`="application/`,
	}

	for _, pattern := range orphanedPatterns {
		result = strings.ReplaceAll(result, pattern, "")
	}

	log.Printf("CSS removal complete, result: %d chars", len(result))
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
