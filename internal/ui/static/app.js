const evtSource = new EventSource("/events");
const logDiv = document.getElementById("log");
const progressBar = document.getElementById("progressBar");
const currentStatus = document.getElementById("currentStatus");

let currentStage = '';

const stages = ['system', 'network', 'connect', 'benchmark', 'report'];
function setStage(stage) {
    if (currentStage === stage) return;

    const currentIndex = stages.indexOf(currentStage);
    const newIndex = stages.indexOf(stage);
    if (newIndex < currentIndex && currentStage !== '') return; // Don't go backwards

    // Mark previous as done
    if (newIndex > 0) {
        for (let i = 0; i < newIndex; i++) {
            const prevEl = document.getElementById('stage-' + stages[i]);
            if (prevEl) {
                prevEl.classList.remove('active');
                prevEl.classList.add('done');
            }
        }
    }

    currentStage = stage;
    const stageEl = document.getElementById('stage-' + stage);
    if (stageEl) stageEl.classList.add('active');
}

function setProgress(percent) {
    if (progressBar) {
        progressBar.style.width = percent + "%";
        progressBar.innerText = percent + "%";
    }
}

evtSource.addEventListener("message", function (event) {
    const msg = event.data;

    // Add to log
    if (logDiv) {
        const msgDiv = document.createElement('div');
        msgDiv.textContent = msg;
        logDiv.appendChild(msgDiv);
        logDiv.scrollTop = logDiv.scrollHeight;
    }

    // Update current status display
    if (currentStatus) currentStatus.innerText = msg;

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
        setProgress(45);
    }
    // Speedtest Logic
    if (msg.includes("Reference Speedtest") || msg.includes("Ref Speed")) {
        setStage('network');
        if (msg.includes("Running")) setProgress(12);
        if (msg.includes("Speedtest:")) setProgress(15);
        if (msg.includes("Ref Speed")) setProgress(18);
    }
    if (msg.includes("Starting Small Files Test") || msg.includes("Medium Files") || msg.includes("Large File") || msg.includes("chunk")) {
        setStage('benchmark');
        if (msg.includes("Starting Small Files Test")) setProgress(52);
        if (msg.includes("Small Files Upload")) setProgress(55);
        if (msg.includes("Starting Small Files Download")) setProgress(57);
        if (msg.includes("Small Files Download")) setProgress(60);
        if (msg.includes("Starting Medium Files Test")) setProgress(62);
        if (msg.includes("Medium Files Upload")) setProgress(65);
        if (msg.includes("Starting Medium Files Download")) setProgress(67);
        if (msg.includes("Medium Files Download")) setProgress(70);
        if (msg.includes("Starting Large File Test")) setProgress(72);
        if (msg.includes("chunk")) {
            const match = msg.match(/chunk (\d+)/);
            if (match) {
                const chunkNum = parseInt(match[1]);
                setProgress(72 + Math.min(Math.floor(chunkNum * 0.2), 10));
            }
        }
        if (msg.includes("Large File Upload")) setProgress(82);
        if (msg.includes("Starting Large File Download")) setProgress(85);
        if (msg.includes("Large File Download")) setProgress(90);
    }

    if (msg.includes("Cleanup") || msg.includes("Generating Report") || msg.includes("Report Ready")) {
        setStage('report');
        if (msg.includes("Cleanup")) setProgress(92);
        if (msg.includes("Generating")) setProgress(95);
        if (msg.includes("Report Ready")) setProgress(98);
    }
});

function updateQualityDot(id, speed, limit, isLarge) {
    const dot = document.getElementById(id);
    if (!speed || speed <= 0 || !limit || limit <= 0) {
        if (dot) dot.className = 'quality-indicator quality-none';
        return;
    }
    const ratio = speed / limit;
    let quality = 'red';
    if (isLarge) {
        if (ratio > 0.70) quality = 'green';
        else if (ratio > 0.40) quality = 'yellow';
    } else {
        if (ratio > 0.15) quality = 'green';
        else if (ratio > 0.07) quality = 'yellow';
    }
    if (dot) dot.className = `quality-indicator quality-${quality}`;
    return quality;
}

function getConclusion(quality) {
    switch (quality) {
        case 'green': return `<span class="text-green" data-i18n="conc_excellent">${translations[currentLang].conc_excellent}</span>`;
        case 'yellow': return `<span class="text-yellow" data-i18n="conc_solid">${translations[currentLang].conc_solid}</span>`;
        case 'red': return `<span class="text-red" data-i18n="conc_optimize">${translations[currentLang].conc_optimize}</span>`;
        default: return '';
    }
}

function updateConclusion(id, q1, q2) {
    const el = document.getElementById(id);
    if (!el) return;
    let finalQ = 'green';
    if (q1 === 'red' || q2 === 'red') finalQ = 'red';
    else if (q1 === 'yellow' || q2 === 'yellow') finalQ = 'yellow';
    else if (q1 === 'none' || q2 === 'none') finalQ = '';

    el.innerHTML = getConclusion(finalQ);
}

evtSource.addEventListener("result", function (event) {
    try {
        let data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            console.error("JSON Parse Error", e, event.data);
            return;
        }

        console.log("Benchmark Result received", data);
        if (data.Completed || data.error) {
            setProgress(100);
        }

        if (data.error) {
            document.getElementById('progressCard').style.display = 'none';
            document.getElementById('resultsCard').style.display = 'block';

            const header = document.querySelector('#resultsCard > div:first-child');
            header.style.background = 'linear-gradient(135deg, #e74c3c 0%, #c0392b 100%)';
            const failText = translations[currentLang].benchmark_failed || "Benchmark Failed";
            const backText = translations[currentLang].btn_back || "Back to Connection Details";
            header.innerHTML = `
                <i class="fas fa-exclamation-circle" style="font-size: 50px; margin-bottom: 15px;"></i>
                <h2 style="margin: 0;">${failText}</h2>
                <div style="margin-top:10px; font-size: 1.1em; background: rgba(0,0,0,0.1); padding: 10px; border-radius: 5px;">${data.error}</div>
                <button class="btn-secondary" onclick="resetUI()" style="margin-top:20px; background: white; color: #c0392b; cursor: pointer;">
                    <i class="fas fa-arrow-left"></i> <span>${backText}</span>
                </button>
            `;
            return;
        }

        if (data.Completed) {
            document.getElementById('progressCard').style.display = 'none';
            document.getElementById('resultsCard').style.display = 'block';

            // Mark all stages as done
            stages.forEach(s => {
                const el = document.getElementById('stage-' + s);
                if (el) {
                    el.classList.remove('active');
                    el.classList.add('done');
                }
            });

            const header = document.querySelector('#resultsCard > div:first-child');
            header.style.background = 'linear-gradient(135deg, #2ecc71 0%, #27ae60 100%)';
            const successText = translations[currentLang].benchmark_completed;
            header.innerHTML = '<i class="fas fa-check-circle" style="font-size: 50px; margin-bottom: 15px;"></i><h2 style="margin: 0;">' + successText + '</h2>';
        }

        // Populate results
        document.getElementById('resSmall').innerText = (data.SmallFiles && data.SmallFiles.speed_mbps > 0) ? (data.SmallFiles.speed_mbps.toFixed(2) + " MB/s") : "--";
        document.getElementById('resMedium').innerText = (data.MediumFiles && data.MediumFiles.speed_mbps > 0) ? (data.MediumFiles.speed_mbps.toFixed(2) + " MB/s") : "--";
        document.getElementById('resLarge').innerText = (data.LargeFile && data.LargeFile.speed_mbps > 0) ? (data.LargeFile.speed_mbps.toFixed(2) + " MB/s") : "--";
        document.getElementById('resSmallDown').innerText = (data.SmallFilesDown && data.SmallFilesDown.speed_mbps > 0) ? (data.SmallFilesDown.speed_mbps.toFixed(2) + " MB/s") : "--";
        document.getElementById('resMediumDown').innerText = (data.MediumFilesDown && data.MediumFilesDown.speed_mbps > 0) ? (data.MediumFilesDown.speed_mbps.toFixed(2) + " MB/s") : "--";
        document.getElementById('resLargeDown').innerText = (data.LargeFileDown && data.LargeFileDown.speed_mbps > 0) ? (data.LargeFileDown.speed_mbps.toFixed(2) + " MB/s") : "--";

        if (data.PingStats) {
            document.getElementById('resPing').innerText = data.PingStats.avg_ms.toFixed(2) + ' ms';
            document.getElementById('resPacketLoss').innerText = data.PingStats.packet_loss.toFixed(1) + '%';

            let pingQ = 'green';
            if (data.PingStats.avg_ms > 60) pingQ = 'red';
            else if (data.PingStats.avg_ms > 25) pingQ = 'yellow';
            document.getElementById('qPing').className = `quality-indicator quality-${pingQ}`;

            let lossQ = 'green';
            if (data.PingStats.packet_loss > 1) lossQ = 'red';
            else if (data.PingStats.packet_loss > 0) lossQ = 'yellow';
            document.getElementById('qLoss').className = `quality-indicator quality-${lossQ}`;

            // Render Ping Table
            const tbody = document.getElementById('pingTableBody');
            if (tbody) {
                tbody.innerHTML = '';
                if (data.PingStats.results) {
                    data.PingStats.results.forEach(r => {
                        const row = document.createElement('tr');
                        row.innerHTML = `<td>${r.Seq}</td>
                                     <td>${r.Success ? r.TimeMs.toFixed(2) : '-'}</td>
                                     <td>${r.Success ? '<span class="success-dot">OK</span>' : '<span class="fail-dot">' + r.ErrorMsg + '</span>'}</td>`;
                        tbody.appendChild(row);
                    });
                }
            }

            document.getElementById('pingCount').innerText = data.PingStats.count;
            document.getElementById('pingAvg').innerText = data.PingStats.avg_ms.toFixed(2) + " ms";
            document.getElementById('pingMin').innerText = data.PingStats.min_ms.toFixed(2);
            document.getElementById('pingMax').innerText = data.PingStats.max_ms.toFixed(2);
            document.getElementById('pingLoss').innerText = data.PingStats.packet_loss.toFixed(1) + "%";
        }

        if (data.DNS) {
            if (data.DNS.resolution_time > 0) {
                document.getElementById('resDNS').innerText = data.DNS.resolution_time.toFixed(1) + " ms";
                const dnsTimeEl = document.getElementById('dnsTime');
                if (dnsTimeEl) dnsTimeEl.innerText = data.DNS.resolution_time.toFixed(2) + " ms";
                const ipsDiv = document.getElementById('dnsIPs');
                if (ipsDiv) {
                    ipsDiv.innerHTML = '';
                    if (data.DNS.resolved_ips) {
                        data.DNS.resolved_ips.forEach(ip => {
                            const d = document.createElement('div');
                            d.innerText = "- " + ip;
                            ipsDiv.appendChild(d);
                        });
                    }
                }
            }
        }

        if (data.advanced_net) {
            document.getElementById('advNetStats').style.display = 'block';
            document.getElementById('valSSL').innerText = data.advanced_net.tls_handshake_ms.toFixed(1) + " ms";
            document.getElementById('valMTU').innerText = data.advanced_net.mtu ? data.advanced_net.mtu + " B" : "Unknown";
            const vpnEl = document.getElementById('valVPN');
            if (data.advanced_net.vpn_detected) {
                vpnEl.innerText = "VPN: " + data.advanced_net.vpn_type;
                if (data.advanced_net.proxy_detected) vpnEl.innerText += " (Proxy)";
            } else if (data.advanced_net.proxy_detected) {
                vpnEl.innerText = "Proxy Detected";
            } else {
                vpnEl.innerText = "";
            }
        }

        if (data.SystemOS) {
            document.getElementById('sysOS').innerText = data.SystemOS;
            document.getElementById('sysCPU').innerText = data.CPU.Model;
            document.getElementById('sysCPUUsage').innerText = data.CPU.Usage.toFixed(1) + "%";
            document.getElementById('sysCPUPeak').innerText = data.peak_cpu_usage.toFixed(1) + "%";
            document.getElementById('sysRAMTotal').innerText = data.RAM.Total;
            document.getElementById('sysRAMUsed').innerText = data.RAM.Used + " (" + data.RAM.Usage.toFixed(1) + "%)";
            document.getElementById('sysRAMFree').innerText = data.RAM.Free;
        }

        if (data.disk_io) {
            document.getElementById('diskWrite').innerText = data.disk_io.write_mbps.toFixed(1) + " MB/s";
            document.getElementById('diskRead').innerText = data.disk_io.read_mbps.toFixed(1) + " MB/s";
        }

        if (data.LocalNetwork) {
            document.getElementById('netConnType').innerText = data.LocalNetwork.ConnectionType;
            document.getElementById('netPrimaryIF').innerText = data.LocalNetwork.PrimaryIF;
        }

        if (data.Traceroute) {
            const trBox = document.getElementById('tracerouteBox');
            if (trBox) {
                trBox.innerHTML = '';
                data.Traceroute.forEach(line => {
                    const d = document.createElement('div');
                    d.innerText = line;
                    trBox.appendChild(d);
                });
            }
        }

        if (data.Speedtest) {
            if (data.Speedtest.error) {
                document.getElementById('refDown').innerText = "Error";
                document.getElementById('refUp').innerText = "Error";
            } else {
                const uMbps = data.Speedtest.upload_speed || 0;
                const dMbps = data.Speedtest.download_speed || 0;
                const uMBps = data.Speedtest.upload_mbps || 0;
                const dMBps = data.Speedtest.download_mbps || 0;
                document.getElementById('refUp').innerText = `${uMBps.toFixed(2)} MB/s (${uMbps.toFixed(2)} Mbps)`;
                document.getElementById('refDown').innerText = `${dMBps.toFixed(2)} MB/s (${dMbps.toFixed(2)} Mbps)`;

                const limitUp = Math.min(uMBps, 10);
                const limitDown = Math.min(dMBps, 50);

                const qSUp = updateQualityDot('qSmallUp', data.SmallFiles.speed_mbps, limitUp, false);
                const qSDown = updateQualityDot('qSmallDown', data.SmallFilesDown.speed_mbps, limitDown, false);
                updateConclusion('concSmall', qSUp, qSDown);

                const qMUp = updateQualityDot('qMedUp', data.MediumFiles.speed_mbps, limitUp, false);
                const qMDown = updateQualityDot('qMedDown', data.MediumFilesDown.speed_mbps, limitDown, false);
                updateConclusion('concMed', qMUp, qMDown);

                const qLUp = updateQualityDot('qLargeUp', data.LargeFile.speed_mbps, limitUp, true);
                const qLDown = updateQualityDot('qLargeDown', data.LargeFileDown.speed_mbps, limitDown, true);
                updateConclusion('concLarge', qLUp, qLDown);

                if (data.Speedtest.isp) document.getElementById('resProvider').innerText = data.Speedtest.isp;
                if (data.Speedtest.server_name) document.getElementById('resStServer').innerText = data.Speedtest.server_name;
            }
        }
    } catch (err) {
        console.error("Result processing error", err);
    }
});

async function startTest() {
    const url = document.getElementById('url').value;
    const user = document.getElementById('user').value;
    const pass = document.getElementById('pass').value;

    if (!url || !user || !pass) {
        alert(translations[currentLang].please_fill);
        return;
    }

    document.getElementById('loginCard').style.display = 'none';
    document.getElementById('progressCard').style.display = 'block';

    try {
        await fetch('/run', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url, user, pass })
        });
    } catch (e) {
        alert("Error: " + e);
        location.reload();
    }
}

async function cancelTest() {
    try {
        await fetch('/run/cancel', { method: 'POST' });
        document.getElementById('cancelBtn').disabled = true;
        document.getElementById('cancelBtn').innerHTML = '<i class="fas fa-spinner fa-spin"></i> Cancelling...';
    } catch (e) {
        console.error("Cancel error", e);
    }
}

function resetUI() {
    document.getElementById('resultsCard').style.display = 'none';
    document.getElementById('progressCard').style.display = 'none';
    document.getElementById('loginCard').style.display = 'block';

    setProgress(0);
    if (logDiv) logDiv.innerHTML = '';
    if (currentStatus) currentStatus.innerText = translations[currentLang].status_initializing;

    ['system', 'network', 'connect', 'benchmark', 'report'].forEach(s => {
        const el = document.getElementById('stage-' + s);
        if (el) el.classList.remove('active', 'done');
    });
    currentStage = '';

    const cBtn = document.getElementById('cancelBtn');
    if (cBtn) {
        cBtn.disabled = false;
        cBtn.innerHTML = `<i class="fas fa-times-circle"></i> <span data-i18n="btn_cancel">${translations[currentLang].btn_cancel}</span>`;
    }
}
