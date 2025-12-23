package ui

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"nextcloud-perf/internal/benchmark"
	"nextcloud-perf/internal/network"
	"nextcloud-perf/internal/report"
	"nextcloud-perf/internal/system"
	"nextcloud-perf/internal/webdav"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	Port         int
	LogChan      chan string
	ResultChan   chan report.ReportData
	LatestReport []byte
	ReportMu     sync.RWMutex
}

func NewServer(port int) *Server {
	return &Server{
		Port:       port,
		LogChan:    make(chan string, 100),
		ResultChan: make(chan report.ReportData, 1),
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
	io.WriteString(w, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nextcloud Performance Tool</title>
    <link rel="stylesheet" href="/static/style.css">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css">
    <style>
        .container { max-width: 1000px; }
        .log-output { 
            background: #1e1e1e; 
            color: #00ff00; 
            padding: 20px; 
            border-radius: 6px; 
            font-family: 'Courier New', monospace; 
            max-height: 300px; 
            overflow-y: auto; 
            margin-top: 15px;
            font-size: 0.85em;
        }
        .progress-stages {
            display: flex;
            justify-content: space-between;
            margin-bottom: 20px;
            padding: 10px 0;
        }
        .stage {
            flex: 1;
            text-align: center;
            padding: 10px;
            background: #f0f0f0;
            margin: 0 5px;
            border-radius: 8px;
            transition: all 0.3s;
            font-size: 0.85em;
        }
        .stage.active {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            transform: scale(1.05);
        }
        .stage.done {
            background: #2ecc71;
            color: white;
        }
        .stage i { font-size: 1.5em; display: block; margin-bottom: 5px; }
        .current-status {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 15px 20px;
            border-radius: 8px;
            font-size: 1.1em;
            margin-bottom: 15px;
            display: flex;
            align-items: center;
            gap: 15px;
        }
        .current-status .spinner {
            width: 24px;
            height: 24px;
            border: 3px solid rgba(255,255,255,0.3);
            border-top-color: white;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        .progress-bar { 
            height: 25px; 
            border-radius: 12px; 
            background: #e0e0e0; 
            margin: 15px 0; 
            overflow: hidden;
        }
        .progress-fill { 
            height: 100%; 
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%); 
            transition: width 0.3s; 
            display: flex; 
            align-items: center; 
            justify-content: center; 
            color: white; 
            font-weight: bold;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1><i class="fas fa-cloud"></i> Nextcloud Performance Check</h1>
            <div class="subtitle">System & Network Analysis Tool</div>
        </header>

        <!-- Login Card -->
        <div class="card" id="loginCard">
            <h2><i class="fas fa-sign-in-alt"></i> Connection Details</h2>
            <div class="form-group">
                <label for="url">Nextcloud URL</label>
                <input type="text" id="url" placeholder="https://cloud.example.com" value="https://">
            </div>
            <div class="form-group">
                <label for="user">Username</label>
                <input type="text" id="user" placeholder="Your Username">
            </div>
            <div class="form-group">
                <label for="pass">Password / App Token</label>
                <input type="password" id="pass" placeholder="Your Password" onkeypress="if(event.key==='Enter')startTest()">
            </div>
            <button class="btn-primary" onclick="startTest()">
                <i class="fas fa-tachometer-alt"></i> Start Benchmark
            </button>
        </div>

        <!-- Testing UI -->
        <div class="card" id="progressCard" style="display: none;">
            <h2><i class="fas fa-running"></i> Test in Progress</h2>
            
            <!-- Stage Indicators -->
            <div class="progress-stages">
                <div class="stage" id="stage-system"><i class="fas fa-desktop"></i> System</div>
                <div class="stage" id="stage-network"><i class="fas fa-network-wired"></i> Network</div>
                <div class="stage" id="stage-connect"><i class="fas fa-plug"></i> Connect</div>
                <div class="stage" id="stage-benchmark"><i class="fas fa-tachometer-alt"></i> Benchmark</div>
                <div class="stage" id="stage-report"><i class="fas fa-file-alt"></i> Report</div>
            </div>
            
            <!-- Current Status -->
            <div class="current-status">
                <div class="spinner"></div>
                <span id="currentStatus">Initializing...</span>
            </div>
            
            <!-- Progress Bar -->
            <div class="progress-bar">
                <div class="progress-fill" id="progressBar" style="width: 0%">0%</div>
            </div>
            
            <!-- Log Output -->
            <div class="log-output" id="log"></div>
        </div>

        <!-- Results Card -->
        <div class="card" id="resultsCard" style="display: none;">
             <div style="background: linear-gradient(135deg, #2ecc71 0%, #27ae60 100%); color: white; padding: 30px; border-radius: 12px; text-align: center; margin-bottom: 25px;">
                <i class="fas fa-check-circle" style="font-size: 50px; margin-bottom: 15px;"></i>
                <h2 style="margin: 0;">Benchmark Completed!</h2>
             </div>

             <h3 style="margin-bottom: 15px;"><i class="fas fa-tachometer-alt"></i> Transfer Speed Results</h3>
             <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 15px; margin-bottom: 25px;">
                <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 12px; text-align: center;">
                    <div style="font-size: 0.85em; opacity: 0.9; margin-bottom: 5px;">Small Files</div>
                    <div style="font-size: 0.7em; opacity: 0.7;">10 x 512KB</div>
                    <div style="font-size: 1.8em; font-weight: bold; margin: 10px 0;" id="resSmall">-- MB/s</div>
                    <div style="font-size: 0.75em; opacity: 0.8;" id="resSmallTime">--</div>
                </div>
                <div style="background: linear-gradient(135deg, #764ba2 0%, #667eea 100%); color: white; padding: 20px; border-radius: 12px; text-align: center;">
                    <div style="font-size: 0.85em; opacity: 0.9; margin-bottom: 5px;">Medium Files</div>
                    <div style="font-size: 0.7em; opacity: 0.7;">5 x 5MB</div>
                    <div style="font-size: 1.8em; font-weight: bold; margin: 10px 0;" id="resMedium">-- MB/s</div>
                    <div style="font-size: 0.75em; opacity: 0.8;" id="resMediumTime">--</div>
                </div>
                <div style="background: linear-gradient(135deg, #2c3e50 0%, #4ca1af 100%); color: white; padding: 20px; border-radius: 12px; text-align: center;">
                    <div style="font-size: 0.85em; opacity: 0.9; margin-bottom: 5px;">Large File</div>
                    <div style="font-size: 0.7em; opacity: 0.7;">512MB Chunked</div>
                    <div style="font-size: 1.8em; font-weight: bold; margin: 10px 0;" id="resLarge">-- MB/s</div>
                    <div style="font-size: 0.75em; opacity: 0.8;" id="resLargeTime">--</div>
                </div>
             </div>

             <h3 style="margin-bottom: 15px;"><i class="fas fa-network-wired"></i> Network Summary</h3>
             <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 15px; margin-bottom: 25px;">
                <div style="background: #f8f9ff; padding: 15px; border-radius: 8px; text-align: center; border: 1px solid #e0e0e0;">
                    <div style="font-size: 0.85em; color: #666;">Latency (Avg)</div>
                    <div style="font-size: 1.4em; font-weight: bold; color: #003d8f;" id="resPing">--</div>
                </div>
                <div style="background: #f8f9ff; padding: 15px; border-radius: 8px; text-align: center; border: 1px solid #e0e0e0;">
                    <div style="font-size: 0.85em; color: #666;">DNS Resolution</div>
                    <div style="font-size: 1.4em; font-weight: bold; color: #003d8f;" id="resDNS">--</div>
                </div>
                <div style="background: #f8f9ff; padding: 15px; border-radius: 8px; text-align: center; border: 1px solid #e0e0e0;">
                    <div style="font-size: 0.85em; color: #666;">Packet Loss</div>
                    <div style="font-size: 1.4em; font-weight: bold; color: #003d8f;" id="resLoss">--</div>
                </div>
             </div>

             <div style="text-align: center; margin-top: 30px;">
                <a href="/report/download" target="_blank" class="btn-primary" style="display: inline-block; text-decoration: none; padding: 18px 50px; font-size: 1.1em;">
                    <i class="fas fa-file-download"></i> Download Detailed Report
                </a>
             </div>
             
             <div style="margin-top: 20px; text-align: center;">
                <button class="btn-secondary" onclick="location.reload()" style="padding: 12px 30px;">
                    <i class="fas fa-redo"></i> Run New Test
                </button>
             </div>
        </div>
    </div>

    <script>
        const evtSource = new EventSource("/events");
        const logDiv = document.getElementById("log");
        const progressBar = document.getElementById("progressBar");
        const currentStatus = document.getElementById("currentStatus");
        
        let currentStage = '';
        
        function setStage(stage) {
            if (currentStage === stage) return;
            
            // Mark previous as done
            if (currentStage) {
                document.getElementById('stage-' + currentStage).classList.remove('active');
                document.getElementById('stage-' + currentStage).classList.add('done');
            }
            
            currentStage = stage;
            document.getElementById('stage-' + stage).classList.add('active');
        }
        
        function setProgress(percent) {
            progressBar.style.width = percent + "%";
            progressBar.innerText = percent + "%";
        }
        
        evtSource.addEventListener("message", function(event) {
            const msg = event.data;
            
            // Add to log
            logDiv.innerHTML += "<div>" + msg + "</div>";
            logDiv.scrollTop = logDiv.scrollHeight;
            
            // Update current status display
            currentStatus.innerText = msg;
            
            // Determine stage and progress based on message content
            if (msg.includes("System") || msg.includes("Collecting System")) {
                setStage('system');
                setProgress(10);
            }
            if (msg.includes("DNS") || msg.includes("Ping") || msg.includes("Traceroute")) {
                setStage('network');
                if (msg.includes("DNS")) setProgress(20);
                if (msg.includes("Ping")) setProgress(30);
                if (msg.includes("Traceroute")) setProgress(40);
            }
            if (msg.includes("Connecting") || msg.includes("Connected")) {
                setStage('connect');
                setProgress(50);
            }
            if (msg.includes("Small Files") || msg.includes("Medium Files") || msg.includes("Large File") || msg.includes("chunk")) {
                setStage('benchmark');
                if (msg.includes("Starting Small")) setProgress(52);
                if (msg.includes("Small Files:")) setProgress(58);
                if (msg.includes("Starting Medium")) setProgress(60);
                if (msg.includes("Medium Files:")) setProgress(68);
                if (msg.includes("Starting Large")) setProgress(70);
                if (msg.includes("chunk")) {
                    const match = msg.match(/chunk (\d+)/);
                    if (match) {
                        const chunkNum = parseInt(match[1]);
                        // 512MB / 10MB = ~52 chunks, map to 70-90%
                        setProgress(70 + Math.min(Math.floor(chunkNum * 0.4), 20));
                    }
                }
                if (msg.includes("Large File:")) setProgress(90);
            }
            if (msg.includes("Cleanup") || msg.includes("Generating Report") || msg.includes("Report Ready")) {
                setStage('report');
                if (msg.includes("Cleanup")) setProgress(92);
                if (msg.includes("Generating")) setProgress(95);
                if (msg.includes("Ready")) setProgress(98);
            }
        });

        evtSource.addEventListener("result", function(event) {
            const data = JSON.parse(event.data);
            
            setProgress(100);
            
            // Transition to Results
            document.getElementById('progressCard').style.display = 'none';
            document.getElementById('resultsCard').style.display = 'block';
            
            // Populate Speed Results
            document.getElementById('resSmall').innerText = data.SmallFiles.SpeedMBps.toFixed(2) + " MB/s";
            document.getElementById('resMedium').innerText = data.MediumFiles.SpeedMBps.toFixed(2) + " MB/s";
            document.getElementById('resLarge').innerText = data.LargeFile.SpeedMBps.toFixed(2) + " MB/s";
            
            // Duration (convert nanoseconds to readable format)
            const formatDuration = (ns) => {
                const ms = ns / 1000000;
                if (ms > 1000) return (ms / 1000).toFixed(1) + "s";
                return ms.toFixed(0) + "ms";
            };
            document.getElementById('resSmallTime').innerText = formatDuration(data.SmallFiles.Duration);
            document.getElementById('resMediumTime').innerText = formatDuration(data.MediumFiles.Duration);
            document.getElementById('resLargeTime').innerText = formatDuration(data.LargeFile.Duration);
            
            // Network Summary
            if (data.PingStats) {
                document.getElementById('resPing').innerText = data.PingStats.AvgMs.toFixed(1) + " ms";
                document.getElementById('resLoss').innerText = data.PingStats.PacketLoss.toFixed(1) + "%";
            }
            if (data.DNS) {
                document.getElementById('resDNS').innerText = data.DNS.ResolutionTime.toFixed(1) + " ms";
            }
        });

        async function startTest() {
            const url = document.getElementById('url').value;
            const user = document.getElementById('user').value;
            const pass = document.getElementById('pass').value;
            
            if (!url || !user || !pass) {
                alert("Please fill in all fields.");
                return;
            }

            document.getElementById('loginCard').style.display = 'none';
            document.getElementById('progressCard').style.display = 'block';
            
            try {
                await fetch('/run', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({url, user, pass})
                });
            } catch (e) {
                alert("Error: " + e);
                location.reload();
            }
        }
    </script>
</body>
</html>
`)
}

func (s *Server) HandleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok { return }

	// Keep connection open until client disconnects
	ctx := r.Context()
	
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-s.ResultChan:
			// Priority: Send result first
			b, _ := json.Marshal(res)
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
		
		// 1. SYSTEM INFO
		s.Broadcast("Collecting System Information...")
		sys, _ := system.GetSystemInfo()
		rpt.SystemOS = sys.OS
		rpt.CPU = report.CPUInfo{ Model: sys.CPUModel, Usage: sys.CPUUsage }
		rpt.RAM = report.RAMInfo{ 
			Total: formatBytes(sys.RAMTotal),
			Free: formatBytes(sys.RAMFree),
			Used: formatBytes(sys.RAMUsed),
			Usage: sys.RAMUsage,
		}
		s.Broadcast(fmt.Sprintf("System: %s | CPU: %.1f%% | RAM: %.1f%% used", sys.OS, sys.CPUUsage, sys.RAMUsage))
		
		// 1b. LOCAL NETWORK INFO
		s.Broadcast("Detecting Local Network...")
		localNet := network.GetLocalNetworkInfo()
		rpt.LocalNetwork = localNet
		if localNet.PrimaryIF != "" {
			s.Broadcast(fmt.Sprintf("Network: %s (%s)", localNet.ConnectionType, localNet.PrimaryIF))
		} else {
			s.Broadcast("Network: Could not detect local network")
		}
		
		// 2. NETWORK
		targetHost := req.URL 
		if len(targetHost) > 8 && targetHost[:8] == "https://" { targetHost = targetHost[8:] }
		if len(targetHost) > 7 && targetHost[:7] == "http://" { targetHost = targetHost[7:] }
		for i, c := range targetHost {
			if c == '/' { targetHost = targetHost[:i]; break }
		}
		hostOnly, _, err := net.SplitHostPort(targetHost)
		if err != nil { hostOnly = targetHost }

		// A. DNS Test
		s.Broadcast("Testing DNS Resolution...")
		dnsRes := network.MeasureDNS(hostOnly)
		rpt.DNS = dnsRes
		if dnsRes.Error != "" {
			s.Broadcast(fmt.Sprintf("DNS Error: %s", dnsRes.Error))
		} else {
			s.Broadcast(fmt.Sprintf("DNS: Resolved %s in %.2fms", hostOnly, dnsRes.ResolutionTime))
		}

		// B. Detailed Ping
		s.Broadcast("Running TCP Ping (10 packets)...")
		tcpTarget := targetHost
		if _, _, err := net.SplitHostPort(tcpTarget); err != nil { tcpTarget = fmt.Sprintf("%s:443", tcpTarget) }
		
		pingStats, err := network.MeasureDetailedTCPPing(tcpTarget, 10, 2*time.Second)
		if err != nil {
			s.Broadcast(fmt.Sprintf("Ping Error: %v", err))
		} else {
			rpt.PingStats = pingStats
			s.Broadcast(fmt.Sprintf("Ping: Avg=%.2fms | Min=%.2fms | Max=%.2fms | Loss=%.1f%%", 
				pingStats.AvgMs, pingStats.MinMs, pingStats.MaxMs, pingStats.PacketLoss))
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
		}

		// 3. WEBDAV
		s.Broadcast("Connecting to Nextcloud WebDAV...")
		client := webdav.NewClient(req.URL, req.User, req.Pass, func(msg string) {
			s.Broadcast(msg) // Forward WebDAV logs directly
		})
		caps, err := client.GetCapabilities()
		if err != nil {
			s.Broadcast(fmt.Sprintf("Error: %v", err))
			return
		}
		rpt.ServerVer = caps.Ocs.Data.Version.String
		s.Broadcast(fmt.Sprintf("Connected! Server: Nextcloud %s", rpt.ServerVer))

		testFolder := "perf-test"
		s.Broadcast("Creating test directory...")
		client.CreateDirectory(testFolder)

		// 4. BENCHMARKS
		// Small Files: 10 x 512KB
		s.Broadcast("Starting Small Files Test (10 x 512KB)...")
		resSmall, err := benchmark.RunSmallFiles(client, testFolder, 10, 512*1024, 5)
		if err != nil {
			rpt.SmallFiles.Errors = []error{err}
			s.Broadcast(fmt.Sprintf("Small Files Error: %v", err))
		} else {
			rpt.SmallFiles = report.SpeedResult{ SpeedMBps: resSmall.SpeedMBps, Duration: resSmall.Duration, Errors: resSmall.Errors }
			s.Broadcast(fmt.Sprintf("Small Files: %.2f MB/s (took %v)", resSmall.SpeedMBps, resSmall.Duration))
		}
		
		// Medium Files: 5 x 5MB (sequential for accurate speed measurement)
		s.Broadcast("Starting Medium Files Test (5 x 5MB)...")
		resMedium, err := benchmark.RunSmallFiles(client, testFolder, 5, 5*1024*1024, 1)
		if err != nil {
			rpt.MediumFiles.Errors = []error{err}
			s.Broadcast(fmt.Sprintf("Medium Files Error: %v", err))
		} else {
			rpt.MediumFiles = report.SpeedResult{ SpeedMBps: resMedium.SpeedMBps, Duration: resMedium.Duration, Errors: resMedium.Errors }
			s.Broadcast(fmt.Sprintf("Medium Files: %.2f MB/s (took %v)", resMedium.SpeedMBps, resMedium.Duration))
		}
		
		// Large File: 512MB with 10MB chunks
		s.Broadcast("Starting Large File Test (512MB with Chunking)...")
		resLarge, err := benchmark.RunLargeFile(client, testFolder, 512*1024*1024, true)
		rpt.LargeFile = report.SpeedResult{ SpeedMBps: resLarge.SpeedMBps, Duration: resLarge.Duration, Errors: resLarge.Errors }
		if resLarge.Errors != nil && len(resLarge.Errors) > 0 {
			s.Broadcast(fmt.Sprintf("Large File Warning: %v", resLarge.Errors))
		}
		s.Broadcast(fmt.Sprintf("Large File: %.2f MB/s (took %v)", resLarge.SpeedMBps, resLarge.Duration))

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
	
	url := fmt.Sprintf("http://localhost:%d", s.Port)
	log.Printf("UI starting at %s", url)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil))
}
