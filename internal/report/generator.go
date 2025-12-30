package report

import (
	"bytes"
	"fmt"
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
	Error           string                   `json:"error,omitempty"` // Global error message if benchmark failed
}

type SpeedResult struct {
	SpeedMBps float64       `json:"speed_mbps"`
	Duration  time.Duration `json:"duration"`
	Errors    []string      `json:"errors"`
}

// GetPingQualityDot returns an HTML span with a colored dot indicating ping quality.
func GetPingQualityDot(p network.DetailedPingStats) template.HTML {
	color := "#2ecc71" // green
	if p.AvgMs > 60 {
		color = "#e74c3c" // red
	} else if p.AvgMs > 25 {
		color = "#f1c40f" // yellow
	}
	return template.HTML(fmt.Sprintf(`<span style="display:inline-block;width:10px;height:10px;border-radius:50%%;background-color:%s;margin-left:5px;vertical-align:middle;box-shadow:0 0 5px %s;"></span>`, color, color))
}

// GetLossQualityDot returns an HTML span with a colored dot indicating packet loss quality.
func GetLossQualityDot(p network.DetailedPingStats) template.HTML {
	color := "#2ecc71" // green
	if p.PacketLoss > 1.0 {
		color = "#e74c3c" // red
	} else if p.PacketLoss > 0.0 {
		color = "#f1c40f" // yellow
	}
	return template.HTML(fmt.Sprintf(`<span style="display:inline-block;width:10px;height:10px;border-radius:50%%;background-color:%s;margin-left:5px;vertical-align:middle;box-shadow:0 0 5px %s;"></span>`, color, color))
}

func (s SpeedResult) GetQualityColor(limitMBps float64, isLarge bool) string {
	if s.SpeedMBps <= 0 || limitMBps <= 0 {
		return "#bdc3c7" // Gray
	}
	ratio := s.SpeedMBps / limitMBps
	if isLarge {
		if ratio > 0.85 {
			return "#2ecc71"
		} // Green
		if ratio > 0.55 {
			return "#f1c40f"
		} // Yellow
	} else {
		if ratio > 0.50 {
			return "#2ecc71"
		} // Green
		if ratio > 0.30 {
			return "#f1c40f"
		} // Yellow
	}
	return "#e74c3c" // Red
}

func (s SpeedResult) GetQualityDot(limitMBps float64, isLarge bool) template.HTML {
	color := s.GetQualityColor(limitMBps, isLarge)
	return template.HTML(fmt.Sprintf(`<span style="display:inline-block;width:10px;height:10px;border-radius:50%%;background-color:%s;margin-left:5px;vertical-align:middle;box-shadow:0 0 5px %s;"></span>`, color, color))
}

func GetCombinedConclusion(up, down SpeedResult, limitUp, limitDown float64, isLarge bool) template.HTML {
	qUp := up.GetQualityColor(limitUp, isLarge)
	qDown := down.GetQualityColor(limitDown, isLarge)

	// Determine worst-case
	key := "conc_excellent"
	text := "Excellent connection"
	class := "text-green"

	if qUp == "#e74c3c" || qDown == "#e74c3c" {
		key = "conc_optimize"
		text = "Needs optimization"
		class = "text-red"
	} else if qUp == "#f1c40f" || qDown == "#f1c40f" {
		key = "conc_solid"
		text = "Solid performance"
		class = "text-yellow"
	} else if qUp == "#bdc3c7" || qDown == "#bdc3c7" {
		return ""
	}

	return template.HTML(fmt.Sprintf(`<div class="conclusion-text %s" data-i18n="%s">%s</div>`, class, key, text))
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
.conclusion-text {
    margin-top: 10px;
    font-size: 0.85em;
    font-weight: bold;
    padding-top: 8px;
    border-top: 1px solid rgba(0,0,0,0.05);
}
.text-green { color: #27ae60; }
.text-yellow { color: #d68910; }
.text-red { color: #c0392b; }
.lang-toggle {
    position: absolute;
    top: 20px;
    right: 20px;
    background: rgba(0, 61, 143, 0.1);
    padding: 5px 10px;
    border-radius: 20px;
    cursor: pointer;
    color: var(--global--color-ionos-blue);
    font-size: 0.9em;
    border: 1px solid rgba(0, 61, 143, 0.2);
    font-weight: bold;
}
.report-container { position: relative; }
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
            <div class="lang-toggle" onclick="toggleLanguage()">
                🌐 <span id="currentLang">EN</span>
            </div>
            <h1 data-i18n="report_title">Nextcloud Performance Report</h1>
            <div class="meta"><span data-i18n="meta_generated">Generated:</span> {{.Data.GeneratedAt.Format "2006-01-02 15:04:05"}}</div>
            <div class="meta"><span data-i18n="meta_target">Target:</span> {{.Data.TargetURL}} | <span data-i18n="meta_server">Server:</span> {{.Data.ServerVer}}</div>
        </header>

        <div class="section">
            <h2 data-i18n="section_system_info">System Information</h2>
            <div class="grid">
                <div class="card">
                    <div class="metric-label" data-i18n="label_client_os">Client OS</div>
                    <div>{{.Data.SystemOS}}</div>
                    <div class="metric-label" data-i18n="label_cpu_model">CPU Model</div>
                    <div style="font-size: 0.8em">{{.Data.CPU.Model}}</div>
                    <div class="metric-label"><span data-i18n="label_cpu_usage">CPU Usage:</span> {{printf "%.1f%%" .Data.CPU.Usage}}</div>
                </div>
                <div class="card">
                    <div class="metric-label" data-i18n="label_memory_ram">Memory (RAM)</div>
                    <div><span data-i18n="label_total">Total:</span> {{.Data.RAM.Total}}</div>
                    <div><span data-i18n="label_used">Used:</span> {{.Data.RAM.Used}} ({{printf "%.1f%%" .Data.RAM.Usage}})</div>
                    <div><span data-i18n="label_free">Free:</span> {{.Data.RAM.Free}}</div>
                </div>
                <div class="card">
                    <div class="metric-label" data-i18n="label_local_network">Local Network</div>
                    <div class="metric-value">{{.Data.LocalNetwork.ConnectionType}}</div>
                    <div class="metric-label"><span data-i18n="label_primary_if">Primary Interface:</span> {{.Data.LocalNetwork.PrimaryIF}}</div>
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
            <h2 data-i18n="section_network_diagnostics">Network Diagnostics</h2>
            <div class="grid">
                <div class="card">
                    <div class="metric-label" data-i18n="label_dns">DNS Resolution</div>
                    <div class="metric-value">{{printf "%.2f ms" .Data.DNS.ResolutionTime}}</div>
                    <div class="metric-label">Resolved IPs:</div>
                    {{range .Data.DNS.ResolvedIPs}}<div>- {{.}}</div>{{end}}
                    {{if .Data.DNS.Error}}<div class="error-box">{{.Data.DNS.Error}}</div>{{end}}
                </div>
                <div class="card">
                    <div class="metric-label"><span data-i18n="label_tcp_connect">TCP Connect</span> ({{.Data.PingStats.Count}} packets)</div>
                    <div class="metric-value"><span data-i18n="label_avg">Avg:</span> {{printf "%.2f ms" .Data.PingStats.AvgMs}} {{getPingQualityDot .Data.PingStats}}</div>
                    <div class="metric-label"><span data-i18n="label_min">Min:</span> {{printf "%.2f" .Data.PingStats.MinMs}} | <span data-i18n="label_max">Max:</span> {{printf "%.2f" .Data.PingStats.MaxMs}}</div>
                    <div class="metric-label"><span data-i18n="label_packet_loss">Loss:</span> {{printf "%.1f%%" .Data.PingStats.PacketLoss}} {{getLossQualityDot .Data.PingStats}}</div>
                </div>
            </div>

            <div style="margin-top:20px;">
                <details>
                    <summary style="cursor:pointer; color: #003d8f; font-weight:bold;" data-i18n="summary_view_ping">View Detailed Ping Results</summary>
                    <table>
                        <thead><tr><th data-i18n="th_seq">Seq</th><th data-i18n="th_time">Time (ms)</th><th data-i18n="th_status">Status</th></tr></thead>
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

        {{$limitUp := 0.0}}{{$limitDown := 0.0}}
        {{if .Data.Speedtest}}
            {{if gt .Data.Speedtest.UploadMBps 10.0}}{{$limitUp = 10.0}}{{else}}{{$limitUp = .Data.Speedtest.UploadMBps}}{{end}}
            {{if gt .Data.Speedtest.DownloadMBps 50.0}}{{$limitDown = 50.0}}{{else}}{{$limitDown = .Data.Speedtest.DownloadMBps}}{{end}}
        <div class="section">
        <div class="section">
            <h2 data-i18n="header_ref_speed">Reference Speed (Speedtest.net)</h2>
            <div class="card" style="background: #f0f4ff; border-color: #d1dbff; text-align: center; margin-bottom: 20px;">
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px;">
                    {{if .Data.Speedtest.ISP}}
                    <div>
                        <div class="metric-label" data-i18n="label_isp">Internet Service Provider</div>
                        <div class="metric-value" style="font-size: 1.1em;">{{.Data.Speedtest.ISP}}</div>
                    </div>
                    {{end}}
                    <div>
                        <div class="metric-label" data-i18n="label_server">Benchmark Server</div>
                        <div class="metric-value" style="font-size: 1.1em;">{{.Data.Speedtest.ServerName}}</div>
                    </div>
                </div>
            </div>
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
                        {{.Data.SmallFiles.GetQualityDot $limitUp false}}
                        <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.SmallFiles.Duration.Seconds}})</span>
                    </div>
                    <div style="margin-top: 5px;">
                        <span style="color: #003d8f; font-weight: bold;">Download:</span> {{printf "%.2f MB/s" .Data.SmallFilesDown.SpeedMBps}}
                        {{.Data.SmallFilesDown.GetQualityDot $limitDown false}}
                        <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.SmallFilesDown.Duration.Seconds}})</span>
                    </div>
                    {{getCombinedConclusion .Data.SmallFiles .Data.SmallFilesDown $limitUp $limitDown false}}

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
                        {{.Data.MediumFiles.GetQualityDot $limitUp false}}
                        <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.MediumFiles.Duration.Seconds}})</span>
                    </div>
                    <div style="margin-top: 5px;">
                        <span style="color: #003d8f; font-weight: bold;">Download:</span> {{printf "%.2f MB/s" .Data.MediumFilesDown.SpeedMBps}}
                        {{.Data.MediumFilesDown.GetQualityDot $limitDown false}}
                         <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.MediumFilesDown.Duration.Seconds}})</span>
                    </div>
                    {{getCombinedConclusion .Data.MediumFiles .Data.MediumFilesDown $limitUp $limitDown false}}

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
                        {{.Data.LargeFile.GetQualityDot $limitUp true}}
                        <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.LargeFile.Duration.Seconds}})</span>
                    </div>
                    <div style="margin-top: 5px;">
                         <span style="color: #003d8f; font-weight: bold;">Download:</span> {{printf "%.2f MB/s" .Data.LargeFileDown.SpeedMBps}}
                         {{.Data.LargeFileDown.GetQualityDot $limitDown true}}
                          <span style="font-size: 0.8em; color: #666;">({{printf "%.2fs" .Data.LargeFileDown.Duration.Seconds}})</span>
                    </div>
                    {{getCombinedConclusion .Data.LargeFile .Data.LargeFileDown $limitUp $limitDown true}}

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
            <small data-i18n="footer">Generated by Nextcloud Performance Tool (Open Source)</small>
        </footer>
    </div>
    <script>
        const translations = {
            en: {
                report_title: "Nextcloud Performance Report",
                meta_generated: "Generated:",
                meta_target: "Target:",
                meta_server: "Server:",
                section_system_info: "System Information",
                label_client_os: "Client OS",
                label_cpu_model: "CPU Model",
                label_cpu_usage: "CPU Usage:",
                label_memory_ram: "Memory (RAM)",
                label_total: "Total:",
                label_used: "Used:",
                label_free: "Free:",
                label_local_network: "Local Network",
                label_primary_if: "Primary Interface:",
                section_network_diagnostics: "Network Diagnostics",
                label_dns: "DNS Resolution",
                label_tcp_connect: "TCP Connect",
                label_avg: "Avg:",
                label_min: "Min:",
                label_max: "Max:",
                label_packet_loss: "Loss:",
                summary_view_ping: "View Detailed Ping Results",
                th_seq: "Seq",
                th_time: "Time (ms)",
                th_status: "Status",
                header_ref_speed: "Reference Speed (Speedtest.net)",
                label_isp: "Internet Service Provider",
                label_server: "Benchmark Server",
                label_upload_speed: "Upload Speed",
                label_download_speed: "Download Speed",
                section_webdav_benchmark: "WebDAV Benchmark",
                label_small_files: "Small Files (5 x 512KB)",
                label_upload: "Upload:",
                label_download: "Download:",
                label_medium_files: "Medium Files (3 x 5MB)",
                label_large_file: "Large File (256MB Chunked)",
                footer: "Generated by Nextcloud Performance Tool (Open Source)",
                conc_excellent: "Excellent connection",
                conc_solid: "Solid performance",
                conc_optimize: "Needs optimization"
            },
            de: {
                report_title: "Nextcloud Performance Bericht",
                meta_generated: "Generiert:",
                meta_target: "Ziel:",
                meta_server: "Server:",
                section_system_info: "Systeminformationen",
                label_client_os: "Client Betriebssystem",
                label_cpu_model: "CPU Modell",
                label_cpu_usage: "CPU Auslastung:",
                label_memory_ram: "Arbeitsspeicher (RAM)",
                label_total: "Gesamt:",
                label_used: "Belegt:",
                label_free: "Frei:",
                label_local_network: "Lokales Netzwerk",
                label_primary_if: "Primäre Schnittstelle:",
                section_network_diagnostics: "Netzwerkdiagnose",
                label_dns: "DNS-Auflösung",
                label_tcp_connect: "TCP Verbindung",
                label_avg: "Durschn.:",
                label_min: "Min:",
                label_max: "Max:",
                label_packet_loss: "Verlust:",
                summary_view_ping: "Detaillierte Ping-Ergebnisse anzeigen",
                th_seq: "Seq",
                th_time: "Zeit (ms)",
                th_status: "Status",
                header_ref_speed: "Referenzgeschwindigkeit (Speedtest.net)",
                label_isp: "Internetanbieter",
                label_server: "Benchmark-Server",
                label_upload_speed: "Upload Geschwindigkeit",
                label_download_speed: "Download Geschwindigkeit",
                section_webdav_benchmark: "WebDAV Benchmark",
                label_small_files: "Kleine Dateien (5 x 512KB)",
                label_upload: "Upload:",
                label_download: "Download:",
                label_medium_files: "Mittlere Dateien (3 x 5MB)",
                label_large_file: "Große Datei (256MB Chunked)",
                footer: "Generiert vom Nextcloud Performance Tool (Open Source)",
                conc_excellent: "Exzellente Verbindung",
                conc_solid: "Solide Leistung",
                conc_optimize: "Optimierungsbedarf"
            }
        };

        const userLang = navigator.language || navigator.userLanguage;
        let currentLang = localStorage.getItem('report_lang') || (userLang.startsWith('de') ? 'de' : 'en'); // Use separate/same key? separate might be safer for report context

        function updateLanguage(lang) {
            currentLang = lang;
            localStorage.setItem('report_lang', lang);
            document.getElementById('currentLang').innerText = lang.toUpperCase();
            
            document.querySelectorAll('[data-i18n]').forEach(el => {
                const key = el.getAttribute('data-i18n');
                if (translations[lang][key]) {
                    el.innerText = translations[lang][key];
                }
            });
        }

        function toggleLanguage() {
            const newLang = currentLang === 'en' ? 'de' : 'en';
            updateLanguage(newLang);
        }

        document.addEventListener('DOMContentLoaded', () => {
             updateLanguage(currentLang);
        });
    </script>
</body>
</html>
`

func GenerateHTML(data ReportData) ([]byte, error) {
	funcMap := template.FuncMap{
		"getPingQualityDot":     GetPingQualityDot,
		"getLossQualityDot":     GetLossQualityDot,
		"getCombinedConclusion": GetCombinedConclusion,
	}

	t, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
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
