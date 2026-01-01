package ui

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"sync"

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

type Server struct {
	Port         int
	LogChan      chan string
	ResultChan   chan report.ReportData
	LatestReport []byte
	ReportMu     sync.RWMutex
	ReadyChan    chan struct{} // Signals when server is ready to accept connections
	cancelFunc   context.CancelFunc
	runMu        sync.Mutex
}

func NewServer(port int) *Server {
	return &Server{
		Port:       port,
		LogChan:    make(chan string, 100),
		ResultChan: make(chan report.ReportData, 1),
		ReadyChan:  make(chan struct{}),
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

	// Keep connection open until client disconnects
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case res := <-s.ResultChan:
			// Priority: Send result first
			b, err := json.Marshal(res)
			if err != nil {
				fmt.Fprintf(w, "data: JSON Marshal Error: %v\n\n", err)
				flusher.Flush()
				continue
			}
			fmt.Fprintf(w, "event: result\ndata: %s\n\n", string(b))
			flusher.Flush()
			// Don't return - keep connection for potential future results
		case msg := <-s.LogChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
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

func (s *Server) HandleRun(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	s.runMu.Unlock()

	go func() {
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
