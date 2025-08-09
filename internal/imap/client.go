package imap

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

type Client struct {
	client *imapclient.Client
	config IMAPConfig
}

type IMAPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      bool
	Timeout  int
}

type Message struct {
	ID      uint32
	UID     uint32
	Subject string
	From    string
	Date    time.Time
	Body    string
}

func NewClient(config IMAPConfig) (*Client, error) {
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
		client: client,
		config: config,
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

func (c *Client) GetMessageBody(ctx context.Context, uid uint32) (string, error) {
	seqSet := imap.UIDSet{}
	seqSet.AddNum(imap.UID(uid))

	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierText},
		},
	}

	msgs := c.client.Fetch(seqSet, fetchOptions)

	msg := msgs.Next()
	if msg == nil {
		return "", fmt.Errorf("failed to fetch message body")
	}

	buffer, err := msg.Collect()
	if err != nil {
		return "", fmt.Errorf("failed to collect message data: %v", err)
	}

	// Try to get text/plain body
	textBody := buffer.FindBodySection(&imap.FetchItemBodySection{Specifier: imap.PartSpecifierText})
	
	if textBody != nil && len(textBody) > 0 {
		finalBody := string(textBody)
		log.Printf("Retrieved text body for UID %d: %d characters", uid, len(finalBody))
		return finalBody, nil
	}

	// If no text body, try to get any body content
	for i := 1; i <= 3; i++ {
		bodySection := buffer.FindBodySection(&imap.FetchItemBodySection{
			Specifier: imap.PartSpecifierNone,
			Part:      []int{i},
		})
		if bodySection != nil && len(bodySection) > 0 {
			finalBody := string(bodySection)
			log.Printf("Retrieved body part %d for UID %d: %d characters", i, uid, len(finalBody))
			return finalBody, nil
		}
	}

	log.Printf("No body found for UID %d", uid)
	return "", nil
}

