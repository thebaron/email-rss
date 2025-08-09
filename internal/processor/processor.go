package processor

import (
	"context"
	"fmt"
	"log"

	"emailrss/internal/db"
	"emailrss/internal/imap"
	"emailrss/internal/rss"
)

type Processor struct {
	imapClient   *imap.Client
	database     *db.DB
	rssGenerator *rss.Generator
	aiHooks      rss.AIHooks
}

func New(imapClient *imap.Client, database *db.DB, rssGenerator *rss.Generator) *Processor {
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

	var newMessages []rss.EmailMessage
	for _, msg := range messages {
		processed, checkErr := p.database.IsMessageProcessed(folderPath, msg.UID)
		if checkErr != nil {
			log.Printf("Failed to check if message is processed: %v", checkErr)
			continue
		}

		if processed {
			continue
		}

		body, bodyErr := p.imapClient.GetMessageBody(ctx, msg.UID)
		if bodyErr != nil {
			log.Printf("Failed to get message body for UID %d: %v", msg.UID, bodyErr)
			body = ""
		}

		rssMsg := rss.EmailMessage{
			UID:     msg.UID,
			Subject: msg.Subject,
			From:    msg.From,
			Date:    msg.Date,
			Body:    body,
		}

		newMessages = append(newMessages, rssMsg)

		if markErr := p.database.MarkMessageProcessed(folderPath, msg.UID, msg.Subject, msg.From, msg.Date); markErr != nil {
			log.Printf("Failed to mark message as processed: %v", markErr)
		}
	}

	if len(newMessages) == 0 {
		log.Printf("No new messages in folder %s", folderPath)
		return nil
	}

	allMessages, err := p.database.GetProcessedMessages(folderPath, 50)
	if err != nil {
		return fmt.Errorf("failed to get processed messages: %v", err)
	}

	var feedMessages []rss.EmailMessage
	for _, dbMsg := range allMessages {
		feedMsg := rss.EmailMessage{
			UID:     dbMsg.UID,
			Subject: dbMsg.Subject,
			From:    dbMsg.From,
			Date:    dbMsg.Date,
		}
		feedMessages = append(feedMessages, feedMsg)
	}

	if err := p.rssGenerator.GenerateFeed(folderPath, feedName, feedMessages, p.aiHooks); err != nil {
		return fmt.Errorf("failed to generate RSS feed: %v", err)
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
