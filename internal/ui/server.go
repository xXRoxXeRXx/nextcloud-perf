package ui

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"nextcloud-perf/internal/benchmark"
	"nextcloud-perf/internal/network"
	"nextcloud-perf/internal/report"
	"nextcloud-perf/internal/system"
	"nextcloud-perf/internal/webdav"
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

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
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
	bytes.NewReader(s.LatestReport).WriteTo(w)
}

type RunRequest struct {
	URL  string `json:"url"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

// Helper to convert []error to []string
func errsToStrings(errs []error) []string {
	var strs []string
	for _, e := range errs {
		if e != nil {
			strs = append(strs, e.Error())
		}
	}
	return strs
}

func (s *Server) HandleRun(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	go func() {
		rpt := report.ReportData{
			GeneratedAt: time.Now(),
			TargetURL:   req.URL,
		}

		s.Broadcast("Starting Benchmark...")

		// Ensure we always send the result at the end, even on error
		defer func() {
			s.SendResult(rpt)
			s.Broadcast("Benchmark Logic Finished.")
		}()

		// 1. SYSTEM INFO
		s.Broadcast("Collecting System Information...")
		sys, err := system.GetSystemInfo()
		if err != nil {
			s.Broadcast(fmt.Sprintf("Warning: Could not get system info: %v", err))
			rpt.SystemOS = "Unknown"
			rpt.CPU = report.CPUInfo{Model: "Unknown", Usage: 0}
			rpt.RAM = report.RAMInfo{Total: "Unknown", Free: "Unknown", Used: "Unknown", Usage: 0}
		} else {
			rpt.SystemOS = sys.OS
			rpt.CPU = report.CPUInfo{Model: sys.CPUModel, Usage: sys.CPUUsage}
			rpt.RAM = report.RAMInfo{
				Total: formatBytes(sys.RAMTotal),
				Free:  formatBytes(sys.RAMFree),
				Used:  formatBytes(sys.RAMUsed),
				Usage: sys.RAMUsage,
			}
			s.Broadcast(fmt.Sprintf("System: %s | CPU: %.1f%% | RAM: %.1f%% used", sys.OS, sys.CPUUsage, sys.RAMUsage))
			s.SendResult(rpt)
		}

		// 1b. LOCAL NETWORK INFO
		s.Broadcast("Detecting Local Network...")
		localNet := network.GetLocalNetworkInfo()
		rpt.LocalNetwork = localNet
		if localNet.PrimaryIF != "" {
			s.Broadcast(fmt.Sprintf("Network: %s (%s)", localNet.ConnectionType, localNet.PrimaryIF))
		} else {
			s.Broadcast("Network: Could not detect local network")
		}

		// 1c. REFERENCE SPEEDTEST
		s.Broadcast("Running Reference Speedtest (Speedtest.net)...")
		stRes, err := network.RunSpeedtest(func(msg string) {
			s.Broadcast("Speedtest: " + msg)
		})
		if err != nil {
			s.Broadcast(fmt.Sprintf("Speedtest Warning: %v", err))
			// Ensure we send an empty result with error so UI knows it finished/failed
			rpt.Speedtest = &network.SpeedtestResult{Error: err.Error()}
			s.SendResult(rpt)
		} else {
			rpt.Speedtest = stRes
			s.Broadcast(fmt.Sprintf("Ref Speed: %.2f Mbps Down / %.2f Mbps Up", stRes.DownloadSpeed, stRes.UploadSpeed))
			s.SendResult(rpt)
		}

		// 2. NETWORK - Parse URL properly using net/url
		parsedURL, err := url.Parse(req.URL)
		var hostOnly string
		if err != nil {
			s.Broadcast(fmt.Sprintf("Warning: Could not parse URL: %v", err))
			hostOnly = req.URL
		} else {
			hostOnly = parsedURL.Hostname()
		}

		// A. DNS Test
		s.Broadcast("Testing DNS Resolution...")
		dnsRes := network.MeasureDNS(hostOnly)
		rpt.DNS = dnsRes
		if dnsRes.Error != "" {
			s.Broadcast(fmt.Sprintf("DNS Error: %s", dnsRes.Error))
		} else {
			s.Broadcast(fmt.Sprintf("DNS: Resolved %s in %.2fms", hostOnly, dnsRes.ResolutionTime))
			s.SendResult(rpt)
		}

		// B. Detailed Ping
		s.Broadcast("Running TCP Ping (10 packets)...")
		tcpTarget := hostOnly
		if parsedURL.Port() != "" {
			tcpTarget = net.JoinHostPort(hostOnly, parsedURL.Port())
		} else {
			if parsedURL.Scheme == "http" {
				tcpTarget = net.JoinHostPort(hostOnly, "80")
			} else {
				tcpTarget = net.JoinHostPort(hostOnly, "443")
			}
		}

		s.Broadcast(fmt.Sprintf("Pinging %s...", tcpTarget))
		pingStats, err := network.MeasureDetailedTCPPing(tcpTarget, 10, 2*time.Second)
		if err != nil {
			s.Broadcast(fmt.Sprintf("Ping Error: %v", err))
			// Set 100% packet loss on error
			rpt.PingStats = network.DetailedPingStats{
				Host:       tcpTarget,
				PacketLoss: 100.0,
			}
			s.SendResult(rpt)
		} else {
			rpt.PingStats = pingStats
			s.Broadcast(fmt.Sprintf("Ping: Avg=%.2fms | Min=%.2fms | Max=%.2fms | Loss=%.1f%%",
				pingStats.AvgMs, pingStats.MinMs, pingStats.MaxMs, pingStats.PacketLoss))
			s.SendResult(rpt)
		}

		// C. Traceroute
		s.Broadcast("Running Traceroute (may require admin)...")
		hops, err := network.RunTraceroute(hostOnly, 15)
		if err != nil {
			s.Broadcast(fmt.Sprintf("Traceroute: Skipped (%v)", err))
		} else {
			s.Broadcast(fmt.Sprintf("Traceroute: Found %d hops", len(hops)))
			for _, h := range hops {
				hh := fmt.Sprintf("%d: %s (%v)", h.TTL, h.Address, h.RTT)
				rpt.Traceroute = append(rpt.Traceroute, hh)
			}
			s.SendResult(rpt)
		}

		// 3. WEBDAV
		s.Broadcast("Connecting to Nextcloud WebDAV...")
		client := webdav.NewClient(req.URL, req.User, req.Pass, func(msg string) {
			s.Broadcast(msg) // Forward WebDAV logs directly
		})
		caps, err := client.GetCapabilities()
		if err != nil {
			s.Broadcast(fmt.Sprintf("Error: %v", err))
			rpt.Error = fmt.Sprintf("Failed to connect to Nextcloud: %v", err)
			return
		}
		rpt.ServerVer = caps.Ocs.Data.Version.String
		s.Broadcast(fmt.Sprintf("Connected! Server: Nextcloud %s", rpt.ServerVer))

		testFolder := fmt.Sprintf("perf-test-%d", time.Now().Unix())
		s.Broadcast("Creating test directory...")
		if err := client.CreateDirectory(testFolder); err != nil {
			s.Broadcast(fmt.Sprintf("Error creating folder: %v", err))
			rpt.Error = fmt.Sprintf("Failed to create test folder: %v", err)
			return
		}

		// 4. BENCHMARKS
		// Small Files: 5 x 512KB
		s.Broadcast("Starting Small Files Test (5 x 512KB)...")
		resSmall, err := benchmark.RunSmallFiles(client, testFolder, "test_small_", 5, 512*1024, 5)
		if err != nil {
			rpt.SmallFiles.Errors = []string{err.Error()}
			s.Broadcast(fmt.Sprintf("Small Files Error: %v", err))
		} else {
			rpt.SmallFiles = report.SpeedResult{SpeedMBps: resSmall.SpeedMBps, Duration: resSmall.Duration, Errors: errsToStrings(resSmall.Errors)}
			s.Broadcast(fmt.Sprintf("Small Files Upload: %.2f MB/s", resSmall.SpeedMBps))
		}

		// Small Files Download
		s.Broadcast("Starting Small Files Download (5 x 512KB)...")
		resSmallDown, err := benchmark.RunDownloadSmallFiles(client, testFolder, "test_small_", 5, 5)
		if err != nil {
			rpt.SmallFilesDown.Errors = []string{err.Error()}
			s.Broadcast(fmt.Sprintf("Download Error: %v", err))
		} else {
			rpt.SmallFilesDown = report.SpeedResult{SpeedMBps: resSmallDown.SpeedMBps, Duration: resSmallDown.Duration, Errors: errsToStrings(resSmallDown.Errors)}
			s.Broadcast(fmt.Sprintf("Small Files Download: %.2f MB/s", resSmallDown.SpeedMBps))
		}

		// Medium Files: 3 x 5MB (sequential for accurate speed measurement)
		s.Broadcast("Starting Medium Files Test (3 x 5MB)...")
		resMedium, err := benchmark.RunSmallFiles(client, testFolder, "test_medium_", 3, 5*1024*1024, 1)
		if err != nil {
			rpt.MediumFiles.Errors = []string{err.Error()}
			s.Broadcast(fmt.Sprintf("Medium Files Error: %v", err))
		} else {
			rpt.MediumFiles = report.SpeedResult{SpeedMBps: resMedium.SpeedMBps, Duration: resMedium.Duration, Errors: errsToStrings(resMedium.Errors)}
			s.Broadcast(fmt.Sprintf("Medium Files Upload: %.2f MB/s", resMedium.SpeedMBps))
		}

		// Medium Files Download
		s.Broadcast("Starting Medium Files Download (3 x 5MB)...")
		resMediumDown, err := benchmark.RunDownloadSmallFiles(client, testFolder, "test_medium_", 3, 1)
		if err != nil {
			rpt.MediumFilesDown.Errors = []string{err.Error()}
			s.Broadcast(fmt.Sprintf("Medium Download Error: %v", err))
		} else {
			rpt.MediumFilesDown = report.SpeedResult{SpeedMBps: resMediumDown.SpeedMBps, Duration: resMediumDown.Duration, Errors: errsToStrings(resMediumDown.Errors)}
			s.Broadcast(fmt.Sprintf("Medium Files Download: %.2f MB/s", resMediumDown.SpeedMBps))
		}

		// Large File: 256MB with Chunking
		s.Broadcast("Starting Large File Test (256MB with Chunking)...")
		resLarge, err := benchmark.RunLargeFile(client, testFolder, 256*1024*1024, true)
		if err != nil {
			rpt.LargeFile.Errors = []string{err.Error()}
			s.Broadcast(fmt.Sprintf("Large File Error: %v", err))
		}
		rpt.LargeFile = report.SpeedResult{SpeedMBps: resLarge.SpeedMBps, Duration: resLarge.Duration, Errors: errsToStrings(resLarge.Errors)}
		if len(resLarge.Errors) > 0 {
			s.Broadcast(fmt.Sprintf("Large File Warning: %v", resLarge.Errors))
		}
		s.Broadcast(fmt.Sprintf("Large File Upload: %.2f MB/s", resLarge.SpeedMBps))

		// Large File Download
		s.Broadcast("Starting Large File Download...")
		resLargeDown, err := benchmark.RunDownloadLargeFile(client, testFolder)
		if err != nil {
			rpt.LargeFileDown.Errors = []string{err.Error()}
			s.Broadcast(fmt.Sprintf("Large Download Error: %v", err))
		} else {
			rpt.LargeFileDown = report.SpeedResult{SpeedMBps: resLargeDown.SpeedMBps, Duration: resLargeDown.Duration, Errors: errsToStrings(resLargeDown.Errors)}
			s.Broadcast(fmt.Sprintf("Large File Download: %.2f MB/s", resLargeDown.SpeedMBps))
		}

		// CLEANUP FIRST (before report)
		s.Broadcast("Cleaning up test files...")
		client.Delete(testFolder)
		s.Broadcast("Cleanup complete.")

		// GENERATE REPORT
		s.Broadcast("Generating Report...")
		htmlBytes, err := report.GenerateHTML(rpt)
		if err != nil {
			s.Broadcast("Failed to generate report: " + err.Error())
		} else {
			s.ReportMu.Lock()
			s.LatestReport = htmlBytes
			s.ReportMu.Unlock()
			s.Broadcast("Report Ready!")

			// Small delay to ensure all log messages are flushed before result
			time.Sleep(100 * time.Millisecond)
			rpt.Completed = true
			s.SendResult(rpt)
		}
	}()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) Listen() {
	http.Handle("/static/", http.FileServer(http.FS(staticFiles)))
	http.HandleFunc("/", s.HandleIndex)
	http.HandleFunc("/events", s.HandleEvents)
	http.HandleFunc("/run", s.HandleRun)
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
