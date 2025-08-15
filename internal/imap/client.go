package imap

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

type Client struct {
	client      *imapclient.Client
	config      IMAPConfig
	debugConfig DebugConfig
}

type IMAPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      bool
	Timeout  int
}

type DebugConfig struct {
	Enabled         bool
	RawMessagesDir  string
	SaveRawMessages bool
	MaxRawMessages  int
}

type Message struct {
	ID      uint32
	UID     uint32
	Subject string
	From    string
	Date    time.Time
	Body    string
}

func NewClient(config IMAPConfig, debugConfig DebugConfig) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	timeout := time.Duration(config.Timeout) * time.Second
	if config.Timeout == 0 {
		timeout = 15 * time.Second
	}

	var conn net.Conn
	var err error

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()

	dialer := &net.Dialer{Timeout: timeout}

	if config.TLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			MinVersion: tls.VersionTLS12,
		})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %v", err)
	}

	client := imapclient.New(conn, &imapclient.Options{
		Dialer: dialer,
	})

	if err := client.Login(config.Username, config.Password).Wait(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to login: %v", err)
	}

	log.Printf("Connected to IMAP server %s as %s", config.Host, config.Username)

	return &Client{
		client:      client,
		config:      config,
		debugConfig: debugConfig,
	}, nil
}

func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *Client) ListFolders(ctx context.Context) ([]string, error) {
	mboxes := c.client.List("", "*", nil)

	var folders []string
	for {
		mbox := mboxes.Next()
		if mbox == nil {
			break
		}
		folders = append(folders, mbox.Mailbox)
	}

	return folders, nil
}

func (c *Client) GetMessages(ctx context.Context, folder string, since time.Time) ([]Message, error) {
	_, err := c.client.Select(folder, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select folder %s: %v", folder, err)
	}

	criteria := &imap.SearchCriteria{}
	if !since.IsZero() {
		log.Printf("Searching for messages since: %v", since)
		criteria.Since = since
	} else {
		log.Printf("Searching for all messages (no since date)")
	}

	data, err := c.client.Search(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %v", err)
	}

	seqNums := data.AllSeqNums()
	log.Printf("Found %d messages in folder %s", len(seqNums), folder)

	if len(seqNums) == 0 {
		return nil, nil
	}

	seqSet := imap.SeqSetNum(seqNums...)
	fetchOptions := &imap.FetchOptions{
		Flags:    true,
		Envelope: true,
		UID:      true,
	}

	msgs := c.client.Fetch(seqSet, fetchOptions)

	var messages []Message
	for {
		msg := msgs.Next()
		if msg == nil {
			break
		}

		buffer, err := msg.Collect()
		if err != nil {
			log.Printf("Failed to collect message: %v", err)
			continue
		}

		if buffer.Envelope == nil {
			log.Printf("Message has no envelope")
			continue
		}

		message := Message{
			ID:      buffer.SeqNum,
			UID:     uint32(buffer.UID),
			Subject: buffer.Envelope.Subject,
			Date:    buffer.Envelope.Date,
		}

		log.Printf("Message SeqNum=%d, UID=%d, Subject=%s", buffer.SeqNum, buffer.UID, buffer.Envelope.Subject)

		if len(buffer.Envelope.From) > 0 {
			addr := buffer.Envelope.From[0]
			if addr.Name != "" {
				message.From = fmt.Sprintf("%s <%s@%s>", addr.Name, addr.Mailbox, addr.Host)
			} else {
				message.From = fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
			}
		}

		messages = append(messages, message)
	}

	log.Printf("Successfully processed %d messages", len(messages))
	return messages, nil
}

type MessageContent struct {
	TextBody string
	HTMLBody string
}

func (c *Client) GetMessageBody(ctx context.Context, uid uint32) (string, error) {
	content, err := c.GetMessageContent(ctx, uid)
	if err != nil {
		return "", err
	}

	// For backward compatibility, prefer HTML over text if available
	if content.HTMLBody != "" {
		return content.HTMLBody, nil
	}
	return content.TextBody, nil
}

func (c *Client) GetMessageContent(ctx context.Context, uid uint32) (*MessageContent, error) {
	seqSet := imap.UIDSet{}
	seqSet.AddNum(imap.UID(uid))

	// Fetch both the body structure and various body sections
	// This code constructs fetchOptions to specify which parts of the email message to fetch from the IMAP server.
	// - BodySection: a list of body sections to fetch:
	//   - {Specifier: imap.PartSpecifierText} fetches the text/plain part of the message body.
	//   - {Specifier: imap.PartSpecifierHeader} fetches the message headers.
	//   - {Specifier: imap.PartSpecifierNone} fetches the entire message body (all parts).
	// - BodyStructure: requests the server to include the MIME structure of the message, which helps in parsing multipart messages.
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText},   // text/plain
			{Specifier: imap.PartSpecifierHeader}, // headers
			{Specifier: imap.PartSpecifierNone},   // full body
		},
		BodyStructure: &imap.FetchItemBodyStructure{},
	}

	msgs := c.client.Fetch(seqSet, fetchOptions)

	msg := msgs.Next()
	if msg == nil {
		return nil, fmt.Errorf("failed to fetch message")
	}

	buffer, err := msg.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to collect message data: %v", err)
	}

	// Save raw message data if debug mode is enabled
	if c.debugConfig.Enabled && c.debugConfig.SaveRawMessages {
		// Get the current folder name for debug context
		// For now, we'll use "INBOX" as default - this could be improved by passing folder name as parameter
		currentFolder := "INBOX"

		// Combine all body sections into raw data for debugging
		var rawData bytes.Buffer

		// Write headers first
		for _, section := range buffer.BodySection {
			if section.Section != nil && section.Section.Specifier == imap.PartSpecifierHeader {
				rawData.WriteString("=== HEADERS ===\n")
				rawData.Write(section.Bytes)
				rawData.WriteString("\n\n")
			}
		}

		// Write full body
		for _, section := range buffer.BodySection {
			if section.Section != nil && section.Section.Specifier == imap.PartSpecifierNone {
				rawData.WriteString("=== BODY ===\n")
				rawData.Write(section.Bytes)
				rawData.WriteString("\n\n")
			}
		}

		// Write text part if available
		for _, section := range buffer.BodySection {
			if section.Section != nil && section.Section.Specifier == imap.PartSpecifierText {
				rawData.WriteString("=== TEXT PART ===\n")
				rawData.Write(section.Bytes)
				rawData.WriteString("\n\n")
			}
		}

		// Save to file
		if err := c.saveRawMessage(uid, currentFolder, rawData.Bytes()); err != nil {
			log.Printf("Failed to save raw message for UID %d: %v", uid, err)
		}
	}

	content := &MessageContent{}

	var contentType string
	var boundary string
	for _, section := range buffer.BodySection {
		if section.Section != nil && section.Section.Specifier == imap.PartSpecifierHeader {
			headers, err := mail.ReadMessage(bytes.NewReader(section.Bytes))
			if err == nil {
				contentType = headers.Header.Get("Content-Type")
			}
			break
		}
	}

	if strings.Contains(contentType, "multipart/alternative") {
		_, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			fmt.Printf("Error parsing content type: %v\n", err)
			return content, nil
		}

		boundary = params["boundary"]
		// fmt.Printf("boundary: %s\n", boundary)
	}

	var part []byte
	buffer.BodyStructure.Walk(func(path []int, bs imap.BodyStructure) (walkChildren bool) {
		if strings.HasPrefix(bs.MediaType(), "text/plain") {
			part = buffer.FindBodySection(&imap.FetchItemBodySection{Specifier: imap.PartSpecifierText})
			if part != nil {
				if boundary != "" {
					mr := multipart.NewReader(bytes.NewReader(part), boundary)
					for {
						p, err := mr.NextPart()
						if err != nil {
							break
						}
						slurp, err := io.ReadAll(p)
						if err != nil {
							continue
						}
						ctype := p.Header.Get("Content-Type")
						if strings.HasPrefix(ctype, "text/plain") {
							content.TextBody = string(slurp)
							// fmt.Printf("text part: %s\n", content.TextBody)
						} else if strings.HasPrefix(ctype, "text/html") {
							content.HTMLBody = string(slurp)
							// fmt.Printf("html part: %s\n", content.HTMLBody)
						}
					}
				} else {
					content.TextBody = string(part)
				}
			}
			return false
		}
		return true
	})

	if content.HTMLBody != "" {
		content.HTMLBody = strings.ReplaceAll(content.HTMLBody, "=\r\n", "")
	}
	if content.TextBody != "" {
		content.TextBody = strings.ReplaceAll(content.TextBody, "=\r\n", "")
	}

	return content, nil
}

// saveRawMessage saves the raw message data to disk for debugging purposes
func (c *Client) saveRawMessage(uid uint32, folder string, rawData []byte) error {
	if !c.debugConfig.Enabled || !c.debugConfig.SaveRawMessages {
		return nil
	}

	// Create debug directory if it doesn't exist
	debugDir := filepath.Join(c.debugConfig.RawMessagesDir, folder)
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		log.Printf("Failed to create debug directory %s: %v", debugDir, err)
		return err
	}

	// Generate filename with timestamp and UID
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_uid_%d.eml", timestamp, uid)
	filepath := filepath.Join(debugDir, filename)

	// Write raw message to file
	if err := os.WriteFile(filepath, rawData, 0644); err != nil {
		log.Printf("Failed to save raw message to %s: %v", filepath, err)
		return err
	}

	log.Printf("Saved raw message UID %d to %s", uid, filepath)

	// Clean up old files if we exceed the maximum
	c.cleanupOldRawMessages(debugDir)

	return nil
}

// cleanupOldRawMessages removes old raw message files if we exceed MaxRawMessages
func (c *Client) cleanupOldRawMessages(debugDir string) {
	if c.debugConfig.MaxRawMessages <= 0 {
		return
	}

	// Read directory contents
	files, err := os.ReadDir(debugDir)
	if err != nil {
		log.Printf("Failed to read debug directory %s: %v", debugDir, err)
		return
	}

	// Filter .eml files only
	var emlFiles []os.DirEntry
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".eml") && !file.IsDir() {
			emlFiles = append(emlFiles, file)
		}
	}

	// If we don't exceed the limit, nothing to clean up
	if len(emlFiles) <= c.debugConfig.MaxRawMessages {
		return
	}

	// Sort files by modification time (oldest first) and remove excess
	// Note: For simplicity, we'll just remove files alphabetically since our naming convention
	// includes timestamps, so alphabetical order equals chronological order
	excessCount := len(emlFiles) - c.debugConfig.MaxRawMessages
	for i := 0; i < excessCount; i++ {
		filePath := filepath.Join(debugDir, emlFiles[i].Name())
		if err := os.Remove(filePath); err != nil {
			log.Printf("Failed to remove old raw message file %s: %v", filePath, err)
		} else {
			log.Printf("Removed old raw message file: %s", emlFiles[i].Name())
		}
	}
}
