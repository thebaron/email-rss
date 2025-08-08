package rss

import (
	"fmt"
	"html"
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
		summary, err := aiHooks.SummarizeMessage(msg.Subject, msg.Body)
		if err != nil {
			summary = msg.Body
		}

		item := &feeds.Item{
			Title:       msg.Subject,
			Link:        &feeds.Link{Href: fmt.Sprintf("%s/message/%d", g.config.BaseURL, msg.UID)},
			Description: g.processContent(summary),
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
	content = strings.ReplaceAll(content, "\r\n", "<br>")
	content = strings.ReplaceAll(content, "\n", "<br>")
	content = html.EscapeString(content)
	
	if len(content) > 1000 {
		content = content[:1000] + "..."
	}
	
	return content
}

func (g *Generator) GetFeedPath(feedName string) string {
	return filepath.Join(g.config.OutputDir, fmt.Sprintf("%s.xml", feedName))
}

func (g *Generator) FeedExists(feedName string) bool {
	feedPath := g.GetFeedPath(feedName)
	_, err := os.Stat(feedPath)
	return err == nil
}