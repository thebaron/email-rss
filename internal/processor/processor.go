package processor

import (
	"context"
	"fmt"
	"log"
	"sync"
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
	maxWorkers   int // Maximum concurrent workers for message processing
}

func New(imapClient IMAPClient, database *db.DB, rssGenerator *rss.Generator) *Processor {
	return &Processor{
		imapClient:   imapClient,
		database:     database,
		rssGenerator: rssGenerator,
		maxWorkers:   5, // Default to 5 concurrent workers
	}
}

// SetMaxWorkers configures the maximum number of concurrent workers for message processing
func (p *Processor) SetMaxWorkers(workers int) {
	if workers <= 0 {
		workers = 1
	}
	p.maxWorkers = workers
}

func (p *Processor) SetAIHooks(hooks rss.AIHooks) {
	p.aiHooks = hooks
}

func (p *Processor) ProcessFolders(ctx context.Context, folders map[string]string) error {
	// Process folders concurrently but with limited concurrency
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, p.maxWorkers)
	
	for folderPath, feedName := range folders {
		wg.Add(1)
		go func(folderPath, feedName string) {
			defer wg.Done()
			
			// Acquire semaphore to limit concurrent folder processing
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			if err := p.processFolder(ctx, folderPath, feedName); err != nil {
				log.Printf("Failed to process folder %s: %v", folderPath, err)
			}
		}(folderPath, feedName)
	}
	
	wg.Wait()
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

	// Process messages concurrently
	newMessages, err := p.processMessagesAsync(ctx, folderPath, messages)
	if err != nil {
		return fmt.Errorf("failed to process messages: %v", err)
	}

	if len(newMessages) == 0 {
		log.Printf("No new messages in folder %s", folderPath)
		return nil
	}

	// For now, just use the new messages with bodies for the RSS feed
	// TODO: In the future, we could store message bodies in the database
	// and retrieve them for older messages as well

	log.Printf("Generating RSS and JSON feeds with %d new messages (all have bodies)", len(newMessages))

	// Generate RSS and JSON feeds concurrently
	err = p.generateFeedsAsync(ctx, folderPath, feedName, newMessages)
	if err != nil {
		return fmt.Errorf("failed to generate feeds: %v", err)
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

// processMessagesAsync processes messages concurrently with limited concurrency
func (p *Processor) processMessagesAsync(ctx context.Context, folderPath string, messages []imap.Message) ([]rss.EmailMessage, error) {
	// Channel to collect processed messages
	resultChan := make(chan rss.EmailMessage, len(messages))
	errorChan := make(chan error, len(messages))
	
	// Worker pool for message processing
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, p.maxWorkers)
	
	// Process each message concurrently
	for _, msg := range messages {
		wg.Add(1)
		go func(msg imap.Message) {
			defer wg.Done()
			
			// Acquire semaphore to limit concurrent operations
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Check if message is already processed
			processed, checkErr := p.database.IsMessageProcessed(folderPath, msg.UID)
			if checkErr != nil {
				log.Printf("Failed to check if message UID %d is processed: %v", msg.UID, checkErr)
				errorChan <- checkErr
				return
			}
			
			if processed {
				log.Printf("Message UID %d already processed, skipping", msg.UID)
				return
			}
			
			log.Printf("Processing message UID %d: %s", msg.UID, msg.Subject)
			
			// Get message content
			content, contentErr := p.imapClient.GetMessageContent(ctx, msg.UID)
			if contentErr != nil {
				log.Printf("Failed to get message content for UID %d: %v", msg.UID, contentErr)
				// Create empty content if error
				content = &imap.MessageContent{TextBody: "", HTMLBody: ""}
			}
			
			// Create RSS message
			rssMsg := rss.EmailMessage{
				UID:      msg.UID,
				Subject:  msg.Subject,
				From:     msg.From,
				Date:     msg.Date,
				TextBody: content.TextBody,
				HTMLBody: content.HTMLBody,
			}
			
			// Mark message as processed
			if markErr := p.database.MarkMessageProcessed(folderPath, msg.UID, msg.Subject, msg.From, msg.Date); markErr != nil {
				log.Printf("Failed to mark message UID %d as processed: %v", msg.UID, markErr)
				errorChan <- markErr
				return
			}
			
			// Send result
			resultChan <- rssMsg
			
		}(msg)
	}
	
	// Close channels when all workers are done
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()
	
	// Collect results
	var newMessages []rss.EmailMessage
	var errors []error
	
	for {
		select {
		case msg, ok := <-resultChan:
			if !ok {
				resultChan = nil
			} else {
				newMessages = append(newMessages, msg)
			}
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
			} else {
				errors = append(errors, err)
			}
		}
		
		if resultChan == nil && errorChan == nil {
			break
		}
	}
	
	// Log any errors but don't fail the entire operation
	for _, err := range errors {
		log.Printf("Error during message processing: %v", err)
	}
	
	log.Printf("Processed %d new messages concurrently", len(newMessages))
	return newMessages, nil
}

// generateFeedsAsync generates RSS and JSON feeds concurrently
func (p *Processor) generateFeedsAsync(ctx context.Context, folderPath, feedName string, messages []rss.EmailMessage) error {
	var wg sync.WaitGroup
	var rssErr, jsonErr error
	
	// Generate RSS feed
	wg.Add(1)
	go func() {
		defer wg.Done()
		rssErr = p.rssGenerator.GenerateFeed(folderPath, feedName, messages, p.aiHooks)
		if rssErr != nil {
			log.Printf("Failed to generate RSS feed for %s: %v", folderPath, rssErr)
		}
	}()
	
	// Generate JSON feed
	wg.Add(1)
	go func() {
		defer wg.Done()
		jsonErr = p.rssGenerator.GenerateJSONFeed(folderPath, feedName, messages, p.aiHooks)
		if jsonErr != nil {
			log.Printf("Failed to generate JSON feed for %s: %v", folderPath, jsonErr)
		}
	}()
	
	wg.Wait()
	
	// Return error if either feed generation failed
	if rssErr != nil {
		return fmt.Errorf("RSS feed generation failed: %v", rssErr)
	}
	if jsonErr != nil {
		return fmt.Errorf("JSON feed generation failed: %v", jsonErr)
	}
	
	log.Printf("Successfully generated both RSS and JSON feeds for %s", folderPath)
	return nil
}
