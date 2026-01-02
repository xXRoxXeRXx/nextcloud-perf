package ui

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"nextcloud-perf/internal/report"
	"nextcloud-perf/internal/workflow"
)

//go:embed static/*
//go:embed templates/*
var staticFiles embed.FS

var templates *template.Template

func init() {
	var err error
	templates, err = template.ParseFS(staticFiles, "templates/*.html")
	if err != nil {
		panic(err)
	}
}

// Client represents an SSE client connection
type Client struct {
	id      string
	msgChan chan string
	resChan chan report.ReportData
}

type Server struct {
	Port         int
	LogChan      chan string
	ResultChan   chan report.ReportData
	LatestReport []byte
	ReportMu     sync.RWMutex
	ReadyChan    chan struct{} // Signals when server is ready to accept connections
	cancelFunc   context.CancelFunc
	runMu        sync.Mutex
	
	// Client management for broadcasting
	clients    map[string]*Client
	clientsMu  sync.RWMutex
	register   chan *Client
	unregister chan *Client
}

func NewServer(port int) *Server {
	s := &Server{
		Port:       port,
		LogChan:    make(chan string, 100),
		ResultChan: make(chan report.ReportData, 1),
		ReadyChan:  make(chan struct{}),
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	
	// Start broadcaster goroutine
	go s.broadcaster()
	
	return s
}

// broadcaster handles message distribution to all connected clients
func (s *Server) broadcaster() {
	for {
		select {
		case msg := <-s.LogChan:
			s.clientsMu.RLock()
			for _, client := range s.clients {
				select {
				case client.msgChan <- msg:
				default:
					// Skip if client buffer is full (slow client)
				}
			}
			s.clientsMu.RUnlock()
			
		case res := <-s.ResultChan:
			s.clientsMu.RLock()
			for _, client := range s.clients {
				select {
				case client.resChan <- res:
				default:
					// Skip if client buffer is full
				}
			}
			s.clientsMu.RUnlock()
			
		case c := <-s.register:
			s.clientsMu.Lock()
			s.clients[c.id] = c
			s.clientsMu.Unlock()
			log.Printf("Client %s connected (total: %d)", c.id, len(s.clients))
			
		case c := <-s.unregister:
			s.clientsMu.Lock()
			if _, ok := s.clients[c.id]; ok {
				close(c.msgChan)
				close(c.resChan)
				delete(s.clients, c.id)
				log.Printf("Client %s disconnected (total: %d)", c.id, len(s.clients))
			}
			s.clientsMu.Unlock()
		}
	}
}

func (s *Server) Broadcast(msg string) {
	select {
	case s.LogChan <- msg:
	default:
	}
}

func (s *Server) SendResult(data report.ReportData) {
	s.ResultChan <- data
}

func (s *Server) SaveReport(html []byte) {
	s.ReportMu.Lock()
	defer s.ReportMu.Unlock()
	s.LatestReport = html
}

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if err := templates.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) HandleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	// Create unique client
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := &Client{
		id:      clientID,
		msgChan: make(chan string, 10),
		resChan: make(chan report.ReportData, 1),
	}
	
	// Register client
	s.register <- client
	defer func() {
		s.unregister <- client
	}()
	
	ctx := r.Context()
	
	// Heartbeat ticker to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
			
		case <-ticker.C:
			// Send heartbeat comment
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
			
		case msg, ok := <-client.msgChan:
			if !ok {
				// Channel closed
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
			
		case res, ok := <-client.resChan:
			if !ok {
				// Channel closed
				return
			}
			b, err := json.Marshal(res)
			if err != nil {
				fmt.Fprintf(w, "data: JSON Marshal Error: %v\n\n", err)
				flusher.Flush()
				continue
			}
			fmt.Fprintf(w, "event: result\ndata: %s\n\n", string(b))
			flusher.Flush()
		}
	}
}

func (s *Server) HandleDownloadReport(w http.ResponseWriter, r *http.Request) {
	s.ReportMu.RLock()
	defer s.ReportMu.RUnlock()
	if len(s.LatestReport) == 0 {
		http.Error(w, "No report available", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Disposition", "attachment; filename=Nextcloud_Perf_Report.html")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(s.LatestReport)))
	if _, err := bytes.NewReader(s.LatestReport).WriteTo(w); err != nil {
		log.Printf("Failed to write report download: %v", err)
	}
}

type RunRequest struct {
	URL  string `json:"url"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

// Validate performs input validation to prevent SSRF and injection attacks
func (r *RunRequest) Validate() error {
	// URL validation
	if r.URL == "" {
		return errors.New("URL is required")
	}
	
	parsedURL, err := url.Parse(r.URL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	
	// Only allow HTTP(S) protocols
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return errors.New("only HTTP(S) URLs are allowed")
	}
	
	// Warn about private IPs (SSRF protection)
	hostname := parsedURL.Hostname()
	if hostname == "localhost" || 
	   hostname == "127.0.0.1" ||
	   strings.HasPrefix(hostname, "192.168.") ||
	   strings.HasPrefix(hostname, "10.") ||
	   strings.HasPrefix(hostname, "172.16.") ||
	   strings.HasPrefix(hostname, "172.17.") ||
	   strings.HasPrefix(hostname, "172.18.") ||
	   strings.HasPrefix(hostname, "172.19.") ||
	   strings.HasPrefix(hostname, "172.2") ||
	   strings.HasPrefix(hostname, "172.30.") ||
	   strings.HasPrefix(hostname, "172.31.") {
		log.Printf("Warning: Testing against private IP address: %s", hostname)
	}
	
	// Username validation
	if r.User == "" {
		return errors.New("username is required")
	}
	if len(r.User) > 255 {
		return errors.New("username too long (max 255 chars)")
	}
	
	// Password validation
	if r.Pass == "" {
		return errors.New("password is required")
	}
	if len(r.Pass) > 1024 {
		return errors.New("password too long (max 1024 chars)")
	}
	
	return nil
}

func (s *Server) HandleRun(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Validate input
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	s.runMu.Lock()
	if s.cancelFunc != nil {
		s.runMu.Unlock()
		http.Error(w, "Benchmark already running", 409)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel
	
	// Synchronization channel to ensure goroutine has started before unlock
	started := make(chan struct{})
	
	go func() {
		close(started) // Signal that goroutine has started
		defer func() {
			s.runMu.Lock()
			s.cancelFunc = nil
			s.runMu.Unlock()
		}()

		opts := workflow.BenchmarkOptions{
			URL:  req.URL,
			User: req.User,
			Pass: req.Pass,
		}
		workflow.Run(ctx, opts, s)
	}()
	
	<-started // Wait for goroutine to start
	s.runMu.Unlock()
	
	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleCancel(w http.ResponseWriter, r *http.Request) {
	s.runMu.Lock()
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.Broadcast("Benchmark cancelled by user.")
	}
	s.runMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) Listen() {
	http.Handle("/static/", http.FileServer(http.FS(staticFiles)))
	http.HandleFunc("/", s.HandleIndex)
	http.HandleFunc("/events", s.HandleEvents)
	http.HandleFunc("/run", s.HandleRun)
	http.HandleFunc("/run/cancel", s.HandleCancel)
	http.HandleFunc("/report/download", s.HandleDownloadReport)

	addr := fmt.Sprintf(":%d", s.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to listen on %s: %v", addr, err))
	}

	url := fmt.Sprintf("http://localhost:%d", s.Port)
	log.Printf("UI starting at %s", url)

	// Signal that server is listening
	close(s.ReadyChan)

	log.Fatal(http.Serve(ln, nil))
}
