const evtSource = new EventSource("/events");
const logDiv = document.getElementById("log");
const progressBar = document.getElementById("progressBar");
const currentStatus = document.getElementById("currentStatus");

let currentStage = '';

function setSafeText(id, text) {
    const el = document.getElementById(id);
    if (el) el.innerText = text;
}

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
    }
    const percEl = document.getElementById("progressPercentage");
    if (percEl) {
        percEl.innerText = percent + "%";
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
        if (data.completed || data.error) {
            setProgress(100);
        }

        if (data.error) {
            document.getElementById('progressCard').style.display = 'none';
            document.getElementById('resultsCard').style.display = 'block';

            const header = document.getElementById('resultHeader');
            header.classList.add('premium-header-fail'); // Optional CSS class if we want
            header.style.background = 'linear-gradient(135deg, #e74c3c 0%, #c0392b 100%)';
            const failText = translations[currentLang].benchmark_failed || "Benchmark Failed";
            const backText = translations[currentLang].btn_back || "Back to Connection Details";
            header.innerHTML = `
                <i class="fas fa-exclamation-circle" style="font-size: 60px; margin-bottom: 15px;"></i>
                <h2 style="margin: 0; font-size: 2em;">${failText}</h2>
                <div style="margin-top:20px; font-size: 1.1em; background: rgba(255,255,255,0.1); padding: 15px; border-radius: 12px; backdrop-filter: blur(5px);">${data.error}</div>
                <button class="btn-secondary" onclick="resetUI()" style="margin-top:25px; background: white; color: #c0392b; cursor: pointer; padding: 12px 25px; border-radius: 10px; border: none; font-weight: bold;">
                    <i class="fas fa-arrow-left"></i> <span>${backText}</span>
                </button>
            `;
            return;
        }

        if (data.completed) {
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

            header.style.background = 'linear-gradient(135deg, #27ae60 0%, #166534 100%)';
            const successText = translations[currentLang].benchmark_completed;
            header.innerHTML = '<i class="fas fa-check-circle" style="font-size: 60px; margin-bottom: 15px;"></i><h2 style="margin: 0; font-size: 2em;">' + successText + '</h2>';
        }

        // Populate results (Transfer Speeds)
        if (data.small_files) {
            setSafeText('resSmall', (data.small_files.speed_mbps > 0) ? (data.small_files.speed_mbps.toFixed(2) + " MB/s") : "--");
            if (data.small_files.duration > 0) setSafeText('durSmallUp', (data.small_files.duration / 1000000000).toFixed(1) + "s");
        }
        if (data.small_files_down) {
            setSafeText('resSmallDown', (data.small_files_down.speed_mbps > 0) ? (data.small_files_down.speed_mbps.toFixed(2) + " MB/s") : "--");
            if (data.small_files_down.duration > 0) setSafeText('durSmallDown', (data.small_files_down.duration / 1000000000).toFixed(1) + "s");
        }
        if (data.medium_files) {
            setSafeText('resMedium', (data.medium_files.speed_mbps > 0) ? (data.medium_files.speed_mbps.toFixed(2) + " MB/s") : "--");
            if (data.medium_files.duration > 0) setSafeText('durMedUp', (data.medium_files.duration / 1000000000).toFixed(1) + "s");
        }
        if (data.medium_files_down) {
            setSafeText('resMediumDown', (data.medium_files_down.speed_mbps > 0) ? (data.medium_files_down.speed_mbps.toFixed(2) + " MB/s") : "--");
            if (data.medium_files_down.duration > 0) setSafeText('durMedDown', (data.medium_files_down.duration / 1000000000).toFixed(1) + "s");
        }
        if (data.large_file) {
            setSafeText('resLarge', (data.large_file.speed_mbps > 0) ? (data.large_file.speed_mbps.toFixed(2) + " MB/s") : "--");
            if (data.large_file.duration > 0) setSafeText('durLargeUp', (data.large_file.duration / 1000000000).toFixed(1) + "s");
        }
        if (data.large_file_down) {
            setSafeText('resLargeDown', (data.large_file_down.speed_mbps > 0) ? (data.large_file_down.speed_mbps.toFixed(2) + " MB/s") : "--");
            if (data.large_file_down.duration > 0) setSafeText('durLargeDown', (data.large_file_down.duration / 1000000000).toFixed(1) + "s");
        }

        // Network Stats
        if (data.ping_stats) {
            const p = data.ping_stats;
            setSafeText('resPing', (p.avg_ms || 0).toFixed(2) + ' ms');
            setSafeText('resPacketLoss', (p.packet_loss || 0).toFixed(1) + '%');

            let pingQ = 'green';
            if (p.avg_ms > 60) pingQ = 'red';
            else if (p.avg_ms > 25) pingQ = 'yellow';
            const qPing = document.getElementById('qPing');
            if (qPing) qPing.className = `quality-indicator quality-${pingQ}`;

            let lossQ = 'green';
            if (p.packet_loss > 1) lossQ = 'red';
            else if (p.packet_loss > 0) lossQ = 'yellow';
            const qLoss = document.getElementById('qLoss');
            if (qLoss) qLoss.className = `quality-indicator quality-${lossQ}`;

            // Render Ping Table if provided
            const tbody = document.getElementById('pingTableBody');
            if (tbody && p.results) {
                tbody.innerHTML = '';
                p.results.forEach(r => {
                    const row = document.createElement('tr');
                    row.innerHTML = `<td>${r.seq}</td>
                                 <td>${r.success ? (r.time_ms || 0).toFixed(2) : '-'}</td>
                                 <td>${r.success ? '<span class="success-dot">OK</span>' : '<span class="fail-dot">' + (r.error_msg || 'Error') + '</span>'}</td>`;
                    tbody.appendChild(row);
                });
            }

            setSafeText('pingCount', p.count || "--");
            setSafeText('pingAvg', (p.avg_ms || 0).toFixed(2) + " ms");
            setSafeText('pingMin', (p.min_ms || 0).toFixed(2) + " ms");
            setSafeText('pingMax', (p.max_ms || 0).toFixed(2) + " ms");
            setSafeText('pingLoss', (p.packet_loss || 0).toFixed(1) + "%");
        }

        if (data.dns) {
            const d = data.dns;
            if (d.resolution_time > 0) {
                setSafeText('resDNS', (d.resolution_time || 0).toFixed(1) + " ms");
                setSafeText('dnsTime', (d.resolution_time || 0).toFixed(2) + " ms");
                const ipsDiv = document.getElementById('dnsIPs');
                if (ipsDiv && d.resolved_ips) {
                    ipsDiv.innerHTML = '';
                    d.resolved_ips.forEach(ip => {
                        const div = document.createElement('div');
                        div.innerText = "- " + ip;
                        ipsDiv.appendChild(div);
                    });
                }
            }
        }

        if (data.advanced_net) {
            const a = data.advanced_net;
            const advEl = document.getElementById('advNetStats');
            if (advEl) advEl.style.display = 'grid';
            setSafeText('valSSL', (a.tls_handshake_ms || 0).toFixed(1) + " ms");
            setSafeText('valMTU', a.mtu ? a.mtu + " B" : "Unknown");
            const vpnEl = document.getElementById('valVPN');
            if (a.vpn_detected) {
                vpnEl.innerText = "VPN: " + (a.vpn_type || "Detected");
                if (a.proxy_detected) vpnEl.innerText += " (Proxy)";
            } else if (a.proxy_detected) {
                vpnEl.innerText = "Proxy Detected";
            } else {
                vpnEl.innerText = "";
            }
        }

        // System & Server Info
        if (data.traceroute) {
            const tBox = document.getElementById('tracerouteBox');
            if (tBox) {
                tBox.innerHTML = '';
                data.traceroute.forEach(hop => {
                    const div = document.createElement('div');
                    div.innerText = hop;
                    tBox.appendChild(div);
                });
            }
        }

        if (data.system_os) document.getElementById('sysOS').innerText = data.system_os;

        if (data.cpu) {
            const cpu = data.cpu;
            if (document.getElementById('sysCPU')) document.getElementById('sysCPU').innerText = cpu.model || "--";
            if (document.getElementById('sysCPUUsage')) document.getElementById('sysCPUUsage').innerText = (cpu.usage || 0).toFixed(1) + "%";
        }
        if (data.peak_cpu_usage !== undefined) {
            const peakEl = document.getElementById('sysCPUPeak');
            if (peakEl) peakEl.innerText = data.peak_cpu_usage.toFixed(1) + "%";
        }

        if (data.ram) {
            const ram = data.ram;
            if (document.getElementById('sysRAMTotal')) document.getElementById('sysRAMTotal').innerText = ram.total || "--";
            if (document.getElementById('sysRAMUsed')) document.getElementById('sysRAMUsed').innerText = (ram.used || "--") + " (" + (ram.usage || 0).toFixed(1) + "%)";
            if (document.getElementById('sysRAMFree')) document.getElementById('sysRAMFree').innerText = ram.free || "--";
        }

        if (data.disk_io) {
            if (document.getElementById('diskWrite')) document.getElementById('diskWrite').innerText = (data.disk_io.write_mbps || 0).toFixed(1) + " MB/s";
            if (document.getElementById('diskRead')) document.getElementById('diskRead').innerText = (data.disk_io.read_mbps || 0).toFixed(1) + " MB/s";
        }

        if (data.local_network) {
            const ln = data.local_network;
            document.getElementById('netConnType').innerText = ln.connection_type || "--";
            document.getElementById('netPrimaryIF').innerText = ln.primary_if || "--";

            const listEl = document.getElementById('netInterfacesList');
            if (listEl && ln.interfaces) {
                listEl.innerHTML = '';
                ln.interfaces.forEach(iface => {
                    const row = document.createElement('div');
                    row.style.fontSize = '0.85em';
                    row.style.marginBottom = '8px';
                    row.style.padding = '5px';
                    row.style.borderRadius = '4px';
                    if (iface.name === ln.primary_if) {
                        row.style.background = '#e8f4fd';
                        row.style.borderLeft = '3px solid #003d8f';
                    }

                    const speedHtml = iface.link_speed && iface.link_speed !== 'Unknown'
                        ? `<div style="color: #27ae60; font-weight: bold;">${translations[currentLang].label_link_speed} ${iface.link_speed}</div>`
                        : '';

                    row.innerHTML = `
                        <div style="font-weight: bold;">${iface.name} (${iface.type})</div>
                        <div style="color: #666;">${iface.ip_address}</div>
                        ${speedHtml}
                    `;
                    listEl.appendChild(row);
                });
            }
        }

        if (data.cloud_check) {
            const cc = data.cloud_check;
            const detailEl = document.getElementById('ncStatusDetail');
            const badgeEl = document.getElementById('ncStatusBadge');

            if (detailEl) {
                let statusText = `Nextcloud ${cc.version || "--"}`;
                if (cc.edition) statusText += ` (${cc.edition})`;
                detailEl.innerText = statusText;
            }

            if (badgeEl) {
                if (cc.maintenance) {
                    badgeEl.innerText = "[Maintenance Mode]";
                    badgeEl.className = "badge badge-red";
                    badgeEl.style.display = 'inline-block';
                } else {
                    badgeEl.innerText = cc.status || "Connected";
                    badgeEl.className = "badge badge-blue";
                    badgeEl.style.display = 'inline-block';
                }
            }
        }

        if (data.traceroute) {
            const trBox = document.getElementById('tracerouteBox');
            if (trBox) {
                trBox.innerHTML = '';
                data.traceroute.forEach(line => {
                    const div = document.createElement('div');
                    div.innerText = line;
                    trBox.appendChild(div);
                });
            }
        }

        // Reference Speedtest logic
        if (data.speedtest) {
            const s = data.speedtest;
            if (s.error) {
                document.getElementById('refDown').innerText = "Error";
                document.getElementById('refUp').innerText = "Error";
            } else {
                const uMbps = s.upload_speed || 0;
                const dMbps = s.download_speed || 0;
                const uMBps = s.upload_mbps || 0;
                const dMBps = s.download_mbps || 0;
                document.getElementById('refUp').innerText = `${uMBps.toFixed(2)} MB/s (${uMbps.toFixed(2)} Mbps)`;
                document.getElementById('refDown').innerText = `${dMBps.toFixed(2)} MB/s (${dMbps.toFixed(2)} Mbps)`;

                const limitUp = Math.min(uMBps, 10);
                const limitDown = Math.min(dMBps, 50);

                if (data.small_files && data.small_files_down) {
                    const qSUp = updateQualityDot('qSmallUp', data.small_files.speed_mbps, limitUp, false);
                    const qSDown = updateQualityDot('qSmallDown', data.small_files_down.speed_mbps, limitDown, false);
                    updateConclusion('concSmall', qSUp, qSDown);
                }

                if (data.medium_files && data.medium_files_down) {
                    const qMUp = updateQualityDot('qMedUp', data.medium_files.speed_mbps, limitUp, false);
                    const qMDown = updateQualityDot('qMedDown', data.medium_files_down.speed_mbps, limitDown, false);
                    updateConclusion('concMed', qMUp, qMDown);
                }

                if (data.large_file && data.large_file_down) {
                    const qLUp = updateQualityDot('qLargeUp', data.large_file.speed_mbps, limitUp, true);
                    const qLDown = updateQualityDot('qLargeDown', data.large_file_down.speed_mbps, limitDown, true);
                    updateConclusion('concLarge', qLUp, qLDown);
                }

                if (s.isp) document.getElementById('resProvider').innerText = s.isp;
                if (s.server_name) document.getElementById('resStServer').innerText = s.server_name;
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

    // Reset Stages
    ['system', 'network', 'connect', 'benchmark', 'report'].forEach(s => {
        const el = document.getElementById('stage-' + s);
        if (el) el.classList.remove('active', 'done');
    });
    currentStage = '';

    // Clear lists & boxes
    const trBox = document.getElementById('tracerouteBox');
    if (trBox) trBox.innerHTML = '';
    const listEl = document.getElementById('netInterfacesList');
    if (listEl) listEl.innerHTML = '';
    const dnsIPs = document.getElementById('dnsIPs');
    if (dnsIPs) dnsIPs.innerHTML = '';

    // Reset labels to placeholder
    const labels = [
        'resURL', 'ncStatusDetail', 'resSmall', 'resSmallDown', 'resMedium', 'resMediumDown', 'resLarge', 'resLargeDown',
        'resPing', 'resPacketLoss', 'resDNS', 'diskWrite', 'diskRead',
        'sysOS', 'sysCPU', 'sysCPUUsage', 'sysCPUPeak', 'sysRAMTotal', 'sysRAMUsed', 'sysRAMFree',
        'resProvider', 'resStServer', 'refUp', 'refDown', 'netConnType', 'netPrimaryIF', 'valSSL', 'valMTU'
    ];
    labels.forEach(id => {
        const el = document.getElementById(id);
        if (el) el.innerText = '--';
    });

    const trac = document.getElementById('tracerouteBox');
    if (trac) trac.innerHTML = '';

    const cBtn = document.getElementById('cancelBtn');
    if (cBtn) {
        cBtn.disabled = false;
        cBtn.innerHTML = `<i class="fas fa-times-circle"></i> <span data-i18n="btn_cancel">${translations[currentLang].btn_cancel}</span>`;
    }

    const ncBadge = document.getElementById('ncStatusBadge');
    if (ncBadge) {
        ncBadge.innerText = '';
        ncBadge.style.display = 'none';
        ncBadge.className = 'badge';
    }

    // Reset quality dots
    document.querySelectorAll('.quality-indicator').forEach(dot => {
        dot.className = 'quality-indicator quality-none';
    });
    document.querySelectorAll('.conclusion-text').forEach(c => {
        c.innerHTML = '';
    });
}
