package processor

import (
	"context"
	"fmt"
	"log"
	"time"

	"emailrss/internal/db"
	"emailrss/internal/imap"
	"emailrss/internal/rss"
)

// IMAPClient interface defines the methods needed from the IMAP client
type IMAPClient interface {
	GetMessages(ctx context.Context, folder string, since time.Time) ([]imap.Message, error)
	GetMessageContent(ctx context.Context, uid uint32) (*imap.MessageContent, error)
}

type Processor struct {
	imapClient   IMAPClient
	database     *db.DB
	rssGenerator *rss.Generator
	aiHooks      rss.AIHooks
}

func New(imapClient IMAPClient, database *db.DB, rssGenerator *rss.Generator) *Processor {
	return &Processor{
		imapClient:   imapClient,
		database:     database,
		rssGenerator: rssGenerator,
	}
}

func (p *Processor) SetAIHooks(hooks rss.AIHooks) {
	p.aiHooks = hooks
}

func (p *Processor) ProcessFolders(ctx context.Context, folders map[string]string) error {
	for folderPath, feedName := range folders {
		if err := p.processFolder(ctx, folderPath, feedName); err != nil {
			log.Printf("Failed to process folder %s: %v", folderPath, err)
			continue
		}
	}
	return nil
}

func (p *Processor) processFolder(ctx context.Context, folderPath, feedName string) error {
	log.Printf("Processing folder: %s -> %s", folderPath, feedName)

	lastProcessed, err := p.database.GetLastProcessedDate(folderPath)
	if err != nil {
		return fmt.Errorf("failed to get last processed date: %v", err)
	}

	messages, err := p.imapClient.GetMessages(ctx, folderPath, lastProcessed)
	if err != nil {
		return fmt.Errorf("failed to get messages: %v", err)
	}

	log.Printf("Retrieved %d messages from IMAP for processing", len(messages))

	var newMessages []rss.EmailMessage
	for _, msg := range messages {
		log.Printf("Processing message UID %d: %s", msg.UID, msg.Subject)
		processed, checkErr := p.database.IsMessageProcessed(folderPath, msg.UID)
		if checkErr != nil {
			log.Printf("Failed to check if message is processed: %v", checkErr)
			continue
		}

		if processed {
			log.Printf("Message UID %d already processed, skipping", msg.UID)
			continue
		}

		log.Printf("Message UID %d is new, processing", msg.UID)

		// Get both text and HTML content
		content, contentErr := p.imapClient.GetMessageContent(ctx, msg.UID)
		if contentErr != nil {
			log.Printf("Failed to get message content for UID %d: %v", msg.UID, contentErr)
			// Create empty content if error
			content = &imap.MessageContent{TextBody: "", HTMLBody: ""}
		}

		rssMsg := rss.EmailMessage{
			UID:      msg.UID,
			Subject:  msg.Subject,
			From:     msg.From,
			Date:     msg.Date,
			TextBody: content.TextBody,
			HTMLBody: content.HTMLBody,
		}

		// log.Printf("Created RSS message for UID %d with text: %d chars, HTML: %d chars",
		// 	msg.UID, len(content.TextBody), len(content.HTMLBody))

		newMessages = append(newMessages, rssMsg)

		if markErr := p.database.MarkMessageProcessed(folderPath, msg.UID, msg.Subject, msg.From, msg.Date); markErr != nil {
			log.Printf("Failed to mark message as processed: %v", markErr)
		}
	}

	if len(newMessages) == 0 {
		log.Printf("No new messages in folder %s", folderPath)
		return nil
	}

	// For now, just use the new messages with bodies for the RSS feed
	// TODO: In the future, we could store message bodies in the database
	// and retrieve them for older messages as well

	log.Printf("Generating RSS and JSON feeds with %d new messages (all have bodies)", len(newMessages))

	// Generate RSS feed
	if err := p.rssGenerator.GenerateFeed(folderPath, feedName, newMessages, p.aiHooks); err != nil {
		return fmt.Errorf("failed to generate RSS feed: %v", err)
	}

	// Generate JSON feed
	if err := p.rssGenerator.GenerateJSONFeed(folderPath, feedName, newMessages, p.aiHooks); err != nil {
		return fmt.Errorf("failed to generate JSON feed: %v", err)
	}

	log.Printf("Processed %d new messages for folder %s", len(newMessages), folderPath)
	return nil
}

func (p *Processor) ResetFolder(folderPath string) error {
	if err := p.database.ClearFolderHistory(folderPath); err != nil {
		return fmt.Errorf("failed to clear folder history: %v", err)
	}

	log.Printf("Reset history for folder: %s", folderPath)
	return nil
}
