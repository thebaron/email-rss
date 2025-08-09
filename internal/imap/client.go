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
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{})
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
		criteria.Since = since
	}

	data, err := c.client.Search(criteria, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %v", err)
	}

	if len(data.AllSeqNums()) == 0 {
		return nil, nil
	}

	seqSet := imap.SeqSetNum(data.AllSeqNums()...)
	fetchOptions := &imap.FetchOptions{
		Flags:    true,
		Envelope: true,
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
			continue
		}

		if buffer.Envelope == nil {
			continue
		}

		message := Message{
			ID:      buffer.SeqNum,
			UID:     uint32(buffer.UID),
			Subject: buffer.Envelope.Subject,
			Date:    buffer.Envelope.Date,
		}

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

	body := buffer.FindBodySection(&imap.FetchItemBodySection{Specifier: imap.PartSpecifierText})
	if body != nil {
		return string(body), nil
	}

	return "", nil
}
