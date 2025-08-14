package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	config ServerConfig
}

type ServerConfig struct {
	Host     string
	Port     int
	FeedsDir string
}

func New(config ServerConfig) *Server {
	return &Server{
		config: config,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/feeds/", s.handleFeed)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	log.Printf("Starting server on %s", addr)
	log.Printf("Serving RSS feeds from %s", s.config.FeedsDir)

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	feeds, err := s.listFeeds()
	if err != nil {
		http.Error(w, "Failed to list feeds", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><head><title>EmailRSS Feeds</title></head><body>")
	fmt.Fprintf(w, "<h1>Available RSS Feeds</h1>")
	fmt.Fprintf(w, "<ul>")

	for _, feed := range feeds {
		feedURL := fmt.Sprintf("/feeds/%s", feed)
		fmt.Fprintf(w, "<li><a href=\"%s\">%s</a></li>", feedURL, feed)
	}

	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, "</body></html>")
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	feedName := strings.TrimPrefix(r.URL.Path, "/feeds/")
	if feedName == "" {
		http.Error(w, "Feed name required", http.StatusBadRequest)
		return
	}

	// Determine feed type and set appropriate extension
	var contentType string
	if strings.HasSuffix(feedName, ".json") {
		contentType = "application/feed+json"
	} else if strings.HasSuffix(feedName, ".xml") {
		contentType = "application/rss+xml"
	} else {
		// Default to XML if no extension specified
		feedName += ".xml"
		contentType = "application/rss+xml"
	}

	feedPath := filepath.Join(s.config.FeedsDir, feedName)

	if !s.isValidFeedPath(feedPath) {
		http.NotFound(w, r)
		return
	}

	feedData, err := os.ReadFile(feedPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Failed to read feed", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "max-age=3600")
	if _, err := w.Write(feedData); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "ok", "service": "emailrss"}`)
}

func (s *Server) listFeeds() ([]string, error) {
	entries, err := os.ReadDir(s.config.FeedsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var feeds []string
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".xml") || strings.HasSuffix(entry.Name(), ".json")) {
			feeds = append(feeds, entry.Name())
		}
	}

	return feeds, nil
}

func (s *Server) isValidFeedPath(feedPath string) bool {
	cleanPath := filepath.Clean(feedPath)
	expectedDir := filepath.Clean(s.config.FeedsDir)

	return strings.HasPrefix(cleanPath, expectedDir) &&
		(strings.HasSuffix(cleanPath, ".xml") || strings.HasSuffix(cleanPath, ".json")) &&
		!strings.Contains(feedPath, "..")
}
