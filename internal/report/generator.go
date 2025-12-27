package report

import (
	"bytes"
	"html/template"
	"time"

	"nextcloud-perf/internal/network"
)

type CPUInfo struct {
	Model string
	Usage float64
}

type RAMInfo struct {
	Total string
	Free  string
	Used  string
	Usage float64
}

type ReportData struct {
	GeneratedAt time.Time
	TargetURL   string
	ServerVer   string
	SystemOS    string
	CPU         CPUInfo
	RAM         RAMInfo
	Completed   bool // Signals if the benchmark is fully finished

	LocalNetwork network.LocalNetworkInfo
	PingStats    network.DetailedPingStats
	DNS          network.DNSResult
	Traceroute   []string

	SmallFiles      SpeedResult
	SmallFilesDown  SpeedResult
	MediumFiles     SpeedResult
	MediumFilesDown SpeedResult
	LargeFile       SpeedResult
	LargeFileDown   SpeedResult
	Speedtest       *network.SpeedtestResult `json:"Speedtest,omitempty"`
}

type SpeedResult struct {
	SpeedMBps float64
	Duration  time.Duration
	Errors    []string
}

// Embedded CSS to ensure the report is standalone
const cssStyle = `
:root {
    --global--color-ionos-blue: #003d8f;
    --global--color-dark-midnight: #001b41;
    --text-primary: #333333;
    --text-secondary: #666666;
    --bg-gradient-start: var(--global--color-ionos-blue);
    --bg-gradient-end: var(--global--color-dark-midnight);
}
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #f5f5f5;
    color: var(--text-primary);
    line-height: 1.6;
    margin: 0;
    padding: 20px;
}
.report-container {
    max-width: 900px;
    margin: 0 auto;
    background: white;
    padding: 40px;
    border-radius: 12px;
    box-shadow: 0 4px 20px rgba(0,0,0,0.1);
}
header {
    text-align: center;
    border-bottom: 2px solid var(--global--color-ionos-blue);
    padding-bottom: 20px;
    margin-bottom: 30px;
}
h1 { color: var(--global--color-ionos-blue); margin: 0; }
.meta { color: var(--text-secondary); font-size: 0.9em; margin-top: 5px; }
.section { margin-bottom: 30px; }
h2 { color: var(--global--color-dark-midnight); border-left: 5px solid var(--global--color-ionos-blue); padding-left: 10px; }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 20px; }
.card { background: #f8f9ff; padding: 20px; border-radius: 8px; border: 1px solid #e0e0e0; }
.metric-value { font-size: 1.3em; font-weight: bold; color: var(--global--color-ionos-blue); }
.metric-label { font-size: 0.9em; color: var(--text-secondary); }
.code-box { font-family: monospace; background: #2d2d2d; color: #00ff00; padding: 15px; border-radius: 8px; overflow-x: auto; font-size: 0.9em; }
.error-box { background: #fff0f0; border-left: 4px solid #ff4757; padding: 15px; margin-top: 10px; color: #d63031; }
table { width: 100%; border-collapse: collapse; margin-top: 10px; font-size: 0.9em; }
th, td { border: 1px solid #ddd; padding: 6px; text-align: left; }
th { background-color: #f2f2f2; }
.success-dot { color: green; }
.fail-dot { color: red; }
`

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Nextcloud Performance Report</title>
    <style>{{.Style}}</style>
</head>
<body>
    <div class="report-container">
        <header>
            <h1>Nextcloud Performance Report</h1>
            <div class="meta">Generated: {{.Data.GeneratedAt.Format "2006-01-02 15:04:05"}}</div>
            <div class="meta">Target: {{.Data.TargetURL}} | Server: {{.Data.ServerVer}}</div>
        </header>

        <div class="section">
            <h2>System Information</h2>
            <div class="grid">
                <div class="card">
                    <div class="metric-label">Client OS</div>
                    <div>{{.Data.SystemOS}}</div>
                    <div class="metric-label">CPU Model</div>
                    <div style="font-size: 0.8em">{{.Data.CPU.Model}}</div>
                    <div class="metric-label">CPU Usage: {{printf "%.1f%%" .Data.CPU.Usage}}</div>
                </div>
                <div class="card">
                    <div class="metric-label">Memory (RAM)</div>
                    <div>Total: {{.Data.RAM.Total}}</div>
                    <div>Used: {{.Data.RAM.Used}} ({{printf "%.1f%%" .Data.RAM.Usage}})</div>
                    <div>Free: {{.Data.RAM.Free}}</div>
                </div>
                <div class="card">
                    <div class="metric-label">Local Network</div>
                    <div class="metric-value">{{.Data.LocalNetwork.ConnectionType}}</div>
                    <div class="metric-label">Primary Interface: {{.Data.LocalNetwork.PrimaryIF}}</div>
                    {{range .Data.LocalNetwork.Interfaces}}
                    <div style="font-size: 0.85em; margin-top: 5px;">
                        <strong>{{.Name}}</strong> ({{.Type}}): {{.IPAddress}}
                        {{if .LinkSpeed}}<br>Speed: {{.LinkSpeed}}{{end}}
                    </div>
                    {{end}}
                </div>
            </div>
        </div>



        <div class="section">
            <h2>Network Diagnostics</h2>
            <div class="grid">
                <div class="card">
                    <div class="metric-label">DNS Resolution</div>
                    <div class="metric-value">{{printf "%.2f ms" .Data.DNS.ResolutionTime}}</div>
                    <div class="metric-label">Resolved IPs:</div>
                    {{range .Data.DNS.ResolvedIPs}}<div>- {{.}}</div>{{end}}
                    {{if .Data.DNS.Error}}<div class="error-box">{{.Data.DNS.Error}}</div>{{end}}
                </div>
                <div class="card">
                    <div class="metric-label">TCP Connect ({{.Data.PingStats.Count}} packets)</div>
                    <div class="metric-value">Avg: {{printf "%.2f ms" .Data.PingStats.AvgMs}}</div>
                    <div class="metric-label">Min: {{printf "%.2f" .Data.PingStats.MinMs}} | Max: {{printf "%.2f" .Data.PingStats.MaxMs}}</div>
                    <div class="metric-label">Loss: {{printf "%.1f%%" .Data.PingStats.PacketLoss}}</div>
                </div>
            </div>

            <div style="margin-top:20px;">
                <details>
                    <summary style="cursor:pointer; color: #003d8f; font-weight:bold;">View Detailed Ping Results</summary>
                    <table>
                        <thead><tr><th>Seq</th><th>Time (ms)</th><th>Status</th></tr></thead>
                        <tbody>
                            {{range .Data.PingStats.Results}}
                            <tr>
                                <td>{{.Seq}}</td>
                                <td>{{if .Success}}{{printf "%.2f" .TimeMs}}{{else}}-{{end}}</td>
                                <td>{{if .Success}}<span class="success-dot">OK</span>{{else}}<span class="fail-dot">{{.ErrorMsg}}</span>{{end}}</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </details>
            </div>
            
            {{if .Data.Traceroute}}
            <h3>Traceroute</h3>
            <div class="code-box">
                {{range .Data.Traceroute}}
                <div>{{.}}</div>
                {{end}}
            </div>
            {{end}}
        </div>

        {{if .Data.Speedtest}}
        <div class="section">
            <h2>Reference Speed (Speedtest.net)</h2>
            {{if .Data.Speedtest.ISP}}
            <div class="card" style="background: #f0f4ff; border-color: #d1dbff; text-align: center; margin-bottom: 20px;">
                <div class="metric-label">Internet Service Provider</div>
                <div class="metric-value" style="font-size: 1.1em;">{{.Data.Speedtest.ISP}}</div>
            </div>
            {{end}}
            <div class="grid">
                <div class="card" style="background: #f0fdf4; border-color: #bbf7d0;">
                    <div class="metric-label">Upload Speed</div>
                    <div class="metric-value">{{printf "%.2f MB/s" .Data.Speedtest.UploadMBps}}</div>
                    <div class="metric-label" style="font-size: 0.9em; color: #666;">({{printf "%.2f Mbps" .Data.Speedtest.UploadSpeed}})</div>
                    <div class="metric-label" style="margin-top:5px;">Latency: {{.Data.Speedtest.Latency}}</div>
                </div>
                <div class="card" style="background: #e8f4fd; border-color: #b6e0fe;">
                    <div class="metric-label">Download Speed</div>
                    <div class="metric-value">{{printf "%.2f MB/s" .Data.Speedtest.DownloadMBps}}</div>
                    <div class="metric-label" style="font-size: 0.9em; color: #666;">({{printf "%.2f Mbps" .Data.Speedtest.DownloadSpeed}})</div>
                    <div class="metric-label" style="margin-top:5px;">Server: {{.Data.Speedtest.ServerName}}</div>
                </div>
            </div>
        </div>
        {{end}}

        <div class="section">
            <h2>WebDAV Benchmark</h2>
            <div class="grid">
                <div class="card">
                    <div class="metric-label">Small Files (5 x 512KB)</div>
                    <div style="margin-top: 10px;">
                        <span style="color: #27ae60; font-weight: bold;">Upload:</span> {{printf "%.2f MB/s" .Data.SmallFiles.SpeedMBps}}
                        <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.SmallFiles.Duration.Seconds}})</span>
                    </div>
                    <div style="margin-top: 5px;">
                        <span style="color: #003d8f; font-weight: bold;">Download:</span> {{printf "%.2f MB/s" .Data.SmallFilesDown.SpeedMBps}}
                        <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.SmallFilesDown.Duration.Seconds}})</span>
                    </div>
                    {{if .Data.SmallFiles.Errors}}
                    <div class="error-box">
                        <strong>Up Errors:</strong><br>
                        {{range .Data.SmallFiles.Errors}}- {{.}}<br>{{end}}
                    </div>
                    {{end}}
                     {{if .Data.SmallFilesDown.Errors}}
                    <div class="error-box">
                        <strong>Down Errors:</strong><br>
                        {{range .Data.SmallFilesDown.Errors}}- {{.}}<br>{{end}}
                    </div>
                    {{end}}
                </div>
                <div class="card">
                    <div class="metric-label">Medium Files (3 x 5MB)</div>
                      <div style="margin-top: 10px;">
                        <span style="color: #27ae60; font-weight: bold;">Upload:</span> {{printf "%.2f MB/s" .Data.MediumFiles.SpeedMBps}}
                        <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.MediumFiles.Duration.Seconds}})</span>
                    </div>
                    <div style="margin-top: 5px;">
                        <span style="color: #003d8f; font-weight: bold;">Download:</span> {{printf "%.2f MB/s" .Data.MediumFilesDown.SpeedMBps}}
                         <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.MediumFilesDown.Duration.Seconds}})</span>
                    </div>
                    {{if .Data.MediumFiles.Errors}}
                    <div class="error-box">
                        <strong>Up Errors:</strong><br>
                        {{range .Data.MediumFiles.Errors}}- {{.}}<br>{{end}}
                    </div>
                    {{end}}
                     {{if .Data.MediumFilesDown.Errors}}
                    <div class="error-box">
                        <strong>Down Errors:</strong><br>
                        {{range .Data.MediumFilesDown.Errors}}- {{.}}<br>{{end}}
                    </div>
                    {{end}}
                </div>
                <div class="card">
                     <div class="metric-label">Large File (256MB Chunked)</div>
                      <div style="margin-top: 10px;">
                        <span style="color: #27ae60; font-weight: bold;">Upload:</span> {{printf "%.2f MB/s" .Data.LargeFile.SpeedMBps}}
                        <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.LargeFile.Duration.Seconds}})</span>
                    </div>
                    <div style="margin-top: 5px;">
                         <span style="color: #003d8f; font-weight: bold;">Download:</span> {{printf "%.2f MB/s" .Data.LargeFileDown.SpeedMBps}}
                          <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.LargeFileDown.Duration.Seconds}})</span>
                    </div>
                     {{if .Data.LargeFile.Errors}}
                    <div class="error-box">
                         <strong>Up Errors:</strong><br>
                        {{range .Data.LargeFile.Errors}}- {{.}}<br>{{end}}
                    </div>
                    {{end}}
                     {{if .Data.LargeFileDown.Errors}}
                    <div class="error-box">
                         <strong>Down Errors:</strong><br>
                        {{range .Data.LargeFileDown.Errors}}- {{.}}<br>{{end}}
                    </div>
                    {{end}}
                </div>
            </div>
        </div>
        
        <footer>
            <small>Generated by Nextcloud Performance Tool (Open Source)</small>
        </footer>
    </div>
</body>
</html>
`

func GenerateHTML(data ReportData) ([]byte, error) {
	t, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, struct {
		Style template.CSS
		Data  ReportData
	}{
		Style: template.CSS(cssStyle),
		Data:  data,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
