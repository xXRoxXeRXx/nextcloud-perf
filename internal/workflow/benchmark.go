package workflow

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"nextcloud-perf/internal/benchmark"
	"nextcloud-perf/internal/network"
	"nextcloud-perf/internal/report"
	"nextcloud-perf/internal/system"
	"nextcloud-perf/internal/webdav"
)

// Reporter defines the interface for communicating progress and results back to the UI/Caller.
type Reporter interface {
	Broadcast(msg string)
	SendResult(data report.ReportData)
	SaveReport(html []byte)
}

// BenchmarkOptions contains the necessary credentials and target for the benchmark.
type BenchmarkOptions struct {
	URL  string
	User string
	Pass string
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

// Run executes the full benchmark suite.
func Run(opts BenchmarkOptions, reporter Reporter) {
	rpt := report.ReportData{
		GeneratedAt: time.Now(),
		TargetURL:   opts.URL,
	}

	reporter.Broadcast("Starting Benchmark...")

	// Ensure we always send the result at the end, even on error
	defer func() {
		reporter.SendResult(rpt)
		reporter.Broadcast("Benchmark Logic Finished.")
	}()

	// 1. SYSTEM INFO
	reporter.Broadcast("Collecting System Information...")
	sys, err := system.GetSystemInfo()
	if err != nil {
		reporter.Broadcast(fmt.Sprintf("Warning: Could not get system info: %v", err))
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
		reporter.Broadcast(fmt.Sprintf("System: %s | CPU: %.1f%% | RAM: %.1f%% used", sys.OS, sys.CPUUsage, sys.RAMUsage))
		reporter.SendResult(rpt)
	}

	// 1b. LOCAL NETWORK INFO
	reporter.Broadcast("Detecting Local Network...")
	localNet := network.GetLocalNetworkInfo()
	rpt.LocalNetwork = localNet
	if localNet.PrimaryIF != "" {
		reporter.Broadcast(fmt.Sprintf("Network: %s (%s)", localNet.ConnectionType, localNet.PrimaryIF))
	} else {
		reporter.Broadcast("Network: Could not detect local network")
	}

	// 1c. REFERENCE SPEEDTEST
	reporter.Broadcast("Running Reference Speedtest (Speedtest.net)...")
	stRes, err := network.RunSpeedtest(func(msg string) {
		reporter.Broadcast("Speedtest: " + msg)
	})
	if err != nil {
		reporter.Broadcast(fmt.Sprintf("Speedtest Warning: %v", err))
		// Ensure we send an empty result with error so UI knows it finished/failed
		rpt.Speedtest = &network.SpeedtestResult{Error: err.Error()}
		reporter.SendResult(rpt)
	} else {
		rpt.Speedtest = stRes
		reporter.Broadcast(fmt.Sprintf("Ref Speed: %.2f Mbps Down / %.2f Mbps Up", stRes.DownloadSpeed, stRes.UploadSpeed))
		reporter.SendResult(rpt)
	}

	// 2. NETWORK - Parse URL properly using net/url
	parsedURL, err := url.Parse(opts.URL)
	var hostOnly string
	if err != nil {
		reporter.Broadcast(fmt.Sprintf("Warning: Could not parse URL: %v", err))
		hostOnly = opts.URL
	} else {
		hostOnly = parsedURL.Hostname()
	}

	// A. DNS Test
	reporter.Broadcast("Testing DNS Resolution...")
	dnsRes := network.MeasureDNS(hostOnly)
	rpt.DNS = dnsRes
	if dnsRes.Error != "" {
		reporter.Broadcast(fmt.Sprintf("DNS Error: %s", dnsRes.Error))
	} else {
		reporter.Broadcast(fmt.Sprintf("DNS: Resolved %s in %.2fms", hostOnly, dnsRes.ResolutionTime))
		reporter.SendResult(rpt)
	}

	// B. Detailed Ping
	reporter.Broadcast("Running TCP Ping (10 packets)...")
	var tcpTarget string
	if parsedURL.Port() != "" {
		tcpTarget = net.JoinHostPort(hostOnly, parsedURL.Port())
	} else {
		if parsedURL.Scheme == "http" {
			tcpTarget = net.JoinHostPort(hostOnly, "80")
		} else {
			tcpTarget = net.JoinHostPort(hostOnly, "443")
		}
	}

	reporter.Broadcast(fmt.Sprintf("Pinging %s...", tcpTarget))
	pingStats, err := network.MeasureDetailedTCPPing(tcpTarget, 10, 2*time.Second)
	if err != nil {
		reporter.Broadcast(fmt.Sprintf("Ping Error: %v", err))
		// Set 100% packet loss on error
		rpt.PingStats = network.DetailedPingStats{
			Host:       tcpTarget,
			PacketLoss: 100.0,
		}
		reporter.SendResult(rpt)
	} else {
		rpt.PingStats = pingStats
		reporter.Broadcast(fmt.Sprintf("Ping: Avg=%.2fms | Min=%.2fms | Max=%.2fms | Loss=%.1f%%",
			pingStats.AvgMs, pingStats.MinMs, pingStats.MaxMs, pingStats.PacketLoss))
		reporter.SendResult(rpt)
	}

	// C. Traceroute
	reporter.Broadcast("Running Traceroute (may require admin)...")
	hops, err := network.RunTraceroute(hostOnly, 15)
	if err != nil {
		reporter.Broadcast(fmt.Sprintf("Traceroute: Skipped (%v)", err))
	} else {
		reporter.Broadcast(fmt.Sprintf("Traceroute: Found %d hops", len(hops)))
		for _, h := range hops {
			hh := fmt.Sprintf("%d: %s (%v)", h.TTL, h.Address, h.RTT)
			rpt.Traceroute = append(rpt.Traceroute, hh)
		}
		reporter.SendResult(rpt)
	}

	// 3. WEBDAV
	reporter.Broadcast("Connecting to Nextcloud WebDAV...")
	client := webdav.NewClient(opts.URL, opts.User, opts.Pass, func(msg string) {
		reporter.Broadcast(msg) // Forward WebDAV logs directly
	})
	caps, err := client.GetCapabilities()
	if err != nil {
		reporter.Broadcast(fmt.Sprintf("Error: %v", err))
		rpt.Error = fmt.Sprintf("Failed to connect to Nextcloud: %v", err)
		return
	}
	rpt.ServerVer = caps.Ocs.Data.Version.String
	reporter.Broadcast(fmt.Sprintf("Connected! Server: Nextcloud %s", rpt.ServerVer))

	testFolder := fmt.Sprintf("perf-test-%d", time.Now().Unix())
	reporter.Broadcast("Creating test directory...")
	if err := client.CreateDirectory(testFolder); err != nil {
		reporter.Broadcast(fmt.Sprintf("Error creating folder: %v", err))
		rpt.Error = fmt.Sprintf("Failed to create test folder: %v", err)
		return
	}

	// 4. BENCHMARKS
	// Small Files: 5 x 512KB
	reporter.Broadcast("Starting Small Files Test (5 x 512KB)...")
	resSmall, err := benchmark.RunSmallFiles(client, testFolder, "test_small_", 5, 512*1024, 5)
	if err != nil {
		rpt.SmallFiles.Errors = []string{err.Error()}
		reporter.Broadcast(fmt.Sprintf("Small Files Error: %v", err))
	} else {
		rpt.SmallFiles = report.SpeedResult{SpeedMBps: resSmall.SpeedMBps, Duration: resSmall.Duration, Errors: errsToStrings(resSmall.Errors)}
		reporter.Broadcast(fmt.Sprintf("Small Files Upload: %.2f MB/s", resSmall.SpeedMBps))
	}

	// Small Files Download
	reporter.Broadcast("Starting Small Files Download (5 x 512KB)...")
	resSmallDown, err := benchmark.RunDownloadSmallFiles(client, testFolder, "test_small_", 5, 5)
	if err != nil {
		rpt.SmallFilesDown.Errors = []string{err.Error()}
		reporter.Broadcast(fmt.Sprintf("Download Error: %v", err))
	} else {
		rpt.SmallFilesDown = report.SpeedResult{SpeedMBps: resSmallDown.SpeedMBps, Duration: resSmallDown.Duration, Errors: errsToStrings(resSmallDown.Errors)}
		reporter.Broadcast(fmt.Sprintf("Small Files Download: %.2f MB/s", resSmallDown.SpeedMBps))
	}

	// Medium Files: 3 x 5MB (sequential for accurate speed measurement)
	reporter.Broadcast("Starting Medium Files Test (3 x 5MB)...")
	resMedium, err := benchmark.RunSmallFiles(client, testFolder, "test_medium_", 3, 5*1024*1024, 1)
	if err != nil {
		rpt.MediumFiles.Errors = []string{err.Error()}
		reporter.Broadcast(fmt.Sprintf("Medium Files Error: %v", err))
	} else {
		rpt.MediumFiles = report.SpeedResult{SpeedMBps: resMedium.SpeedMBps, Duration: resMedium.Duration, Errors: errsToStrings(resMedium.Errors)}
		reporter.Broadcast(fmt.Sprintf("Medium Files Upload: %.2f MB/s", resMedium.SpeedMBps))
	}

	// Medium Files Download
	reporter.Broadcast("Starting Medium Files Download (3 x 5MB)...")
	resMediumDown, err := benchmark.RunDownloadSmallFiles(client, testFolder, "test_medium_", 3, 1)
	if err != nil {
		rpt.MediumFilesDown.Errors = []string{err.Error()}
		reporter.Broadcast(fmt.Sprintf("Medium Download Error: %v", err))
	} else {
		rpt.MediumFilesDown = report.SpeedResult{SpeedMBps: resMediumDown.SpeedMBps, Duration: resMediumDown.Duration, Errors: errsToStrings(resMediumDown.Errors)}
		reporter.Broadcast(fmt.Sprintf("Medium Files Download: %.2f MB/s", resMediumDown.SpeedMBps))
	}

	// Large File: 256MB with Chunking
	reporter.Broadcast("Starting Large File Test (256MB with Chunking)...")
	resLarge, err := benchmark.RunLargeFile(client, testFolder, 256*1024*1024, true)
	if err != nil {
		rpt.LargeFile.Errors = []string{err.Error()}
		reporter.Broadcast(fmt.Sprintf("Large File Error: %v", err))
	}
	rpt.LargeFile = report.SpeedResult{SpeedMBps: resLarge.SpeedMBps, Duration: resLarge.Duration, Errors: errsToStrings(resLarge.Errors)}
	if len(resLarge.Errors) > 0 {
		reporter.Broadcast(fmt.Sprintf("Large File Warning: %v", resLarge.Errors))
	}
	reporter.Broadcast(fmt.Sprintf("Large File Upload: %.2f MB/s", resLarge.SpeedMBps))

	// Large File Download
	reporter.Broadcast("Starting Large File Download...")
	resLargeDown, err := benchmark.RunDownloadLargeFile(client, testFolder)
	if err != nil {
		rpt.LargeFileDown.Errors = []string{err.Error()}
		reporter.Broadcast(fmt.Sprintf("Large Download Error: %v", err))
	} else {
		rpt.LargeFileDown = report.SpeedResult{SpeedMBps: resLargeDown.SpeedMBps, Duration: resLargeDown.Duration, Errors: errsToStrings(resLargeDown.Errors)}
		reporter.Broadcast(fmt.Sprintf("Large File Download: %.2f MB/s", resLargeDown.SpeedMBps))
	}

	// CLEANUP FIRST (before report)
	reporter.Broadcast("Cleaning up test files...")
	if err := client.Delete(testFolder); err != nil {
		reporter.Broadcast(fmt.Sprintf("Warning: Cleanup failed: %v", err))
	}
	reporter.Broadcast("Cleanup complete.")

	// GENERATE REPORT
	reporter.Broadcast("Generating Report...")
	htmlBytes, err := report.GenerateHTML(rpt)
	if err != nil {
		reporter.Broadcast("Failed to generate report: " + err.Error())
	} else {
		reporter.SaveReport(htmlBytes)
		reporter.Broadcast("Report Ready!")

		// Small delay to ensure all log messages are flushed before result
		time.Sleep(100 * time.Millisecond)
		rpt.Completed = true
		reporter.SendResult(rpt)
	}
}
