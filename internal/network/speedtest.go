package network

import (
	"fmt"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

type SpeedtestResult struct {
	ServerID      string        `json:"server_id"`
	ServerName    string        `json:"server_name"`
	ServerCountry string        `json:"server_country"`
	Latency       time.Duration `json:"latency"`
	DownloadUnit  string        `json:"download_unit"`
	UploadUnit    string        `json:"upload_unit"`
	DownloadSpeed float64       `json:"download_speed"` // Mbps
	UploadSpeed   float64       `json:"upload_speed"`   // Mbps
	DownloadMBps  float64       `json:"download_mbps"`  // MB/s
	UploadMBps    float64       `json:"upload_mbps"`    // MB/s
	Error         string        `json:"error,omitempty"`
}

// RunSpeedtest performs a speedtest against the nearest server
func RunSpeedtest(logFunc func(string)) (*SpeedtestResult, error) {
	// Fetch servers
	logFunc("Updating server list...")
	serverList, err := speedtest.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server list: %v", err)
	}

	if len(serverList) == 0 {
		return nil, fmt.Errorf("no speedtest servers found")
	}

	logFunc("Finding best server...")
	targets, err := serverList.FindServer([]int{})
	if err != nil {
		return nil, fmt.Errorf("failed to find best server: %v", err)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no target server found")
	}

	target := targets[0]
	logFunc(fmt.Sprintf("Benchmarking against: %s (%s) - %s", target.Name, target.Country, target.Sponsor))

	// Ping
	err = target.PingTest(nil)
	if err != nil {
		return nil, fmt.Errorf("ping test failed: %v", err)
	}
	logFunc(fmt.Sprintf("Ping: %v", target.Latency))

	// Download
	logFunc("Running Download Test...")
	err = target.DownloadTest()
	if err != nil {
		return nil, fmt.Errorf("download test failed: %v", err)
	}
	dlMbps := (float64(target.DLSpeed) * 8) / 1000000.0
	dlMBps := float64(target.DLSpeed) / 1000000.0
	logFunc(fmt.Sprintf("Download: %.2f Mbps (%.2f MB/s)", dlMbps, dlMBps))

	// Upload
	logFunc("Running Upload Test...")
	err = target.UploadTest()
	if err != nil {
		return nil, fmt.Errorf("upload test failed: %v", err)
	}
	ulMbps := (float64(target.ULSpeed) * 8) / 1000000.0
	ulMBps := float64(target.ULSpeed) / 1000000.0
	logFunc(fmt.Sprintf("Upload: %.2f Mbps (%.2f MB/s)", ulMbps, ulMBps))

	return &SpeedtestResult{
		ServerID:      target.ID,
		ServerName:    fmt.Sprintf("%s (%s)", target.Sponsor, target.Name),
		ServerCountry: target.Country,
		Latency:       target.Latency,
		DownloadUnit:  "Mbps",
		UploadUnit:    "Mbps",
		DownloadSpeed: dlMbps,
		UploadSpeed:   ulMbps,
		DownloadMBps:  dlMBps,
		UploadMBps:    ulMBps,
	}, nil

}
