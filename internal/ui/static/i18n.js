const translations = {
    en: {
        title: "Nextcloud Performance Check",
        subtitle: "System & Network Analysis Tool",
        connection_details: "Connection Details",
        label_url: "Nextcloud URL",
        label_username: "Username",
        label_password: "Password / App Token",
        placeholder_username: "Your Username",
        placeholder_password: "Your Password",
        btn_start: "Start Benchmark",
        test_running: "Test in Progress",
        stage_system: "System",
        stage_network: "Network",
        stage_connect: "Connect",
        stage_benchmark: "Benchmark",
        stage_report: "Report",
        status_initializing: "Initializing...",
        benchmark_completed: "Benchmark Completed!",
        benchmark_failed: "Benchmark Failed",
        header_ref_speed: "Reference Speed (Speedtest.net)",
        label_isp: "Internet Service Provider",
        label_server: "Benchmark Server",
        label_upload_ref: "Upload (Ref)",
        label_download_ref: "Download (Ref)",
        header_transfer_speed: "Transfer Speed Results",
        label_small_files: "Small Files",
        label_medium_files: "Medium Files",
        label_large_file: "Large File",
        label_upload: "Upload:",
        label_download: "Download:",
        header_network_summary: "Network Summary",
        label_latency: "Avg Latency",
        label_packet_loss: "Packet Loss",
        label_dns: "DNS Resolution",
        btn_download_report: "Download Detailed Report",
        btn_run_new: "Run New Test",
        conc_excellent: "Excellent connection",
        conc_solid: "Solid performance",
        conc_optimize: "Needs optimization",
        please_fill: "Please fill in all fields.",
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
        label_tcp_connect: "TCP Connect",
        label_avg: "Avg:",
        label_min: "Min:",
        label_max: "Max:",
        summary_view_ping: "View Detailed Ping Results",
        th_seq: "Seq",
        th_time: "Time (ms)",
        th_status: "Status",
        btn_cancel: "Cancel Test",
        btn_back: "Back to Connection Details",
        label_write: "Write:",
        label_read: "Read:"
    },
    de: {
        title: "Nextcloud Performance Check",
        subtitle: "System- & Netzwerkanalyse-Tool",
        connection_details: "Verbindungsdetails",
        label_url: "Nextcloud URL",
        label_username: "Benutzername",
        label_password: "Passwort / App Token",
        placeholder_username: "Dein Benutzername",
        placeholder_password: "Dein Passwort",
        btn_start: "Benchmark Starten",
        test_running: "Test läuft",
        stage_system: "System",
        stage_network: "Netzwerk",
        stage_connect: "Verbinden",
        stage_benchmark: "Benchmark",
        stage_report: "Bericht",
        status_initializing: "Initialisiere...",
        benchmark_completed: "Benchmark Abgeschlossen!",
        benchmark_failed: "Benchmark Fehlgeschlagen",
        header_ref_speed: "Referenzgeschwindigkeit (Speedtest.net)",
        label_isp: "Internetanbieter",
        label_server: "Benchmark-Server",
        label_upload_ref: "Upload (Ref)",
        label_download_ref: "Download (Ref)",
        header_transfer_speed: "Übertragungsgeschwindigkeit",
        label_small_files: "Kleine Dateien",
        label_medium_files: "Mittlere Dateien",
        label_large_file: "Große Datei",
        label_upload: "Upload:",
        label_download: "Download:",
        header_network_summary: "Netzwerk-Zusammenfassung",
        label_latency: "Durchschn. Latenz",
        label_packet_loss: "Packet Loss",
        label_dns: "DNS-Auflösung",
        btn_download_report: "Detaillierten Bericht herunterladen",
        btn_run_new: "Neuen Test starten",
        conc_excellent: "Exzellente Verbindung",
        conc_solid: "Solide Leistung",
        conc_optimize: "Optimierungsbedarf",
        please_fill: "Bitte füllen Sie alle Felder aus.",
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
        label_tcp_connect: "TCP Verbindung",
        label_avg: "Durschn.:",
        label_min: "Min:",
        label_max: "Max:",
        summary_view_ping: "Detaillierte Ping-Ergebnisse anzeigen",
        th_seq: "Seq",
        th_time: "Zeit (ms)",
        th_status: "Status",
        btn_cancel: "Test abbrechen",
        btn_back: "Zurück zu den Verbindungsdetails",
        label_write: "Schreiben:",
        label_read: "Read:"
    }
};

// Determine default language (Browser -> de, else en)
const userLang = navigator.language || navigator.userLanguage;
let currentLang = localStorage.getItem('lang') || (userLang.startsWith('de') ? 'de' : 'en');

function updateLanguage(lang) {
    currentLang = lang;
    localStorage.setItem('lang', lang);
    const langEl = document.getElementById('currentLang');
    if (langEl) langEl.innerText = lang.toUpperCase();

    // Translate elements
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (translations[lang][key]) {
            el.innerText = translations[lang][key];
        }
    });

    // Translate placeholders
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.getAttribute('data-i18n-placeholder');
        if (translations[lang][key]) {
            el.placeholder = translations[lang][key];
        }
    });
}

function toggleLanguage() {
    const newLang = currentLang === 'en' ? 'de' : 'en';
    updateLanguage(newLang);
}

// Init language on load
document.addEventListener('DOMContentLoaded', () => {
    updateLanguage(currentLang);
});
