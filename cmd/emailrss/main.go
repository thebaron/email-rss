package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"

	"emailrss/internal/config"
	"emailrss/internal/db"
	"emailrss/internal/imap"
	"emailrss/internal/processor"
	"emailrss/internal/rss"
	"emailrss/internal/server"
)

type CLI struct {
	Config string `short:"c" long:"config" default:"config.yaml" help:"Configuration file path"`

	Serve   ServeCmd   `cmd:"" help:"Start the RSS server"`
	Process ProcessCmd `cmd:"" help:"Process emails and generate RSS feeds"`
	Reset   ResetCmd   `cmd:"" help:"Reset folder history"`
}

type ServeCmd struct{}

type ProcessCmd struct {
	Once bool `short:"o" long:"once" help:"Process once and exit"`
}

type ResetCmd struct {
	Folder string `arg:"" required:"" help:"Folder path to reset"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli)

	cfg, err := config.Load(cli.Config)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	switch ctx.Command() {
	case "serve":
		err = runServe(cfg)
	case "process":
		err = runProcess(cfg, database, cli.Process.Once)
	case "reset <folder>":
		err = runReset(cfg, database, cli.Reset.Folder)
	default:
		log.Fatalf("Unknown command: %s", ctx.Command())
	}

	if err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

func runServe(cfg *config.Config) error {
	srv := server.New(server.ServerConfig{
		Host:     cfg.Server.Host,
		Port:     cfg.Server.Port,
		FeedsDir: cfg.RSS.OutputDir,
	})

	return srv.Start()
}

func runProcess(cfg *config.Config, database *db.DB, once bool) error {
	imapConfig := imap.IMAPConfig{
		Host:     cfg.IMAP.Host,
		Port:     cfg.IMAP.Port,
		Username: cfg.IMAP.Username,
		Password: cfg.IMAP.Password,
		TLS:      cfg.IMAP.TLS,
		Timeout:  cfg.IMAP.Timeout,
	}

	debugConfig := imap.DebugConfig{
		Enabled:         cfg.Debug.Enabled,
		RawMessagesDir:  cfg.Debug.RawMessagesDir,
		SaveRawMessages: cfg.Debug.SaveRawMessages,
		MaxRawMessages:  cfg.Debug.MaxRawMessages,
	}

	imapClient, err := imap.NewClient(imapConfig, debugConfig)
	if err != nil {
		return err
	}
	defer imapClient.Close()

	rssConfig := rss.RSSConfig{
		OutputDir:            cfg.RSS.OutputDir,
		Title:                cfg.RSS.Title,
		BaseURL:              cfg.RSS.BaseURL,
		MaxHTMLContentLength: cfg.RSS.MaxHTMLContentLength,
		MaxTextContentLength: cfg.RSS.MaxTextContentLength,
		MaxRSSHTMLLength:     cfg.RSS.MaxRSSHTMLLength,
		MaxRSSTextLength:     cfg.RSS.MaxRSSTextLength,
		MaxSummaryLength:     cfg.RSS.MaxSummaryLength,
		RemoveCSS:            cfg.RSS.RemoveCSS,
	}

	rssGenerator := rss.NewGenerator(rssConfig)
	proc := processor.New(imapClient, database, rssGenerator)
	proc.SetMaxWorkers(cfg.Processing.MaxWorkers)

	ctx := context.Background()

	if once {
		return proc.ProcessFolders(ctx, cfg.IMAP.Folders)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("Starting email processing loop...")

	// Process immediately on startup
	if err := proc.ProcessFolders(ctx, cfg.IMAP.Folders); err != nil {
		log.Printf("Initial processing failed: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := proc.ProcessFolders(ctx, cfg.IMAP.Folders); err != nil {
				log.Printf("Processing failed: %v", err)
			}
		case <-sigChan:
			log.Println("Shutting down...")
			return nil
		}
	}
}

func runReset(cfg *config.Config, database *db.DB, folderPath string) error {
	imapConfig := imap.IMAPConfig{
		Host:     cfg.IMAP.Host,
		Port:     cfg.IMAP.Port,
		Username: cfg.IMAP.Username,
		Password: cfg.IMAP.Password,
		TLS:      cfg.IMAP.TLS,
		Timeout:  cfg.IMAP.Timeout,
	}

	debugConfig := imap.DebugConfig{
		Enabled:         cfg.Debug.Enabled,
		RawMessagesDir:  cfg.Debug.RawMessagesDir,
		SaveRawMessages: cfg.Debug.SaveRawMessages,
		MaxRawMessages:  cfg.Debug.MaxRawMessages,
	}

	imapClient, err := imap.NewClient(imapConfig, debugConfig)
	if err != nil {
		return err
	}
	defer imapClient.Close()

	rssConfig := rss.RSSConfig{
		OutputDir:            cfg.RSS.OutputDir,
		Title:                cfg.RSS.Title,
		BaseURL:              cfg.RSS.BaseURL,
		MaxHTMLContentLength: cfg.RSS.MaxHTMLContentLength,
		MaxTextContentLength: cfg.RSS.MaxTextContentLength,
		MaxRSSHTMLLength:     cfg.RSS.MaxRSSHTMLLength,
		MaxRSSTextLength:     cfg.RSS.MaxRSSTextLength,
		MaxSummaryLength:     cfg.RSS.MaxSummaryLength,
		RemoveCSS:            cfg.RSS.RemoveCSS,
	}

	rssGenerator := rss.NewGenerator(rssConfig)
	proc := processor.New(imapClient, database, rssGenerator)
	proc.SetMaxWorkers(cfg.Processing.MaxWorkers)

	return proc.ResetFolder(folderPath)
}
