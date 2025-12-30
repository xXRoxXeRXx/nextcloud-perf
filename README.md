<p align="center">
  <img src="assets/logo.png" alt="Nextcloud Perf Logo" width="400">
</p>

# Nextcloud Perf

<p align="center">
  <strong>Ein leistungsstarkes Toolset zur detaillierten Performance-Analyse und Benchmarking von Nextcloud-Instanzen.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=for-the-badge" alt="Platform">
  <img src="https://img.shields.io/github/v/release/xxroxxerxx/nextcloud-perf?style=for-the-badge" alt="Release">
</p>

---

## 🚀 Überblick

`nextcloud-perf` hilft dir dabei, Engpässe in deiner Nextcloud-Umgebung zu identifizieren. Ob Netzwerklatenz, langsame WebDAV-Operationen oder Ressourcenmangel auf dem Server – dieses Tool liefert dir die nötigen Daten direkt in einer übersichtlichen Weboberfläche.

## ✨ Kernfunktionen (v2.2.0)

| Kategorie | Features |
| :--- | :--- |
| **🌐 Netzwerk** | **Neu**: Referenz-Speedtest (Speedtest.net) & Ampelsystem für Latenz/Packet Loss |
| **📁 WebDAV** | Upload/Download-Benchmark (Chunked Uploads 25MB, Unique Folders) |
| **🧠 Analyse** | **Neu**: Automatische Qualitätsbewertung ("Exzellent", "Solide", "Optimierungsbedarf") |
| **🛡️ Stabilität** | **Neu**: Robustes "Fail-Fast" Error Handling bei Verbindungsproblemen |
| **🌍 Sprache** | **Neu (v2.3.0)**: Vollständige Übersetzung (DE/EN) mit Auto-Detection |
| **📊 Reporting** | HTML-Report Generator mit detaillierten Metriken & Conclusion-Texten |

---

## 🆕 Was ist neu in v2.3.0?

*   **Internationalisierung (i18n)**:
    *   Das Tool spricht jetzt **Deutsch & Englisch**.
    *   **Auto-Detection**: Startet automatisch in deiner Browsersprache.
    *   **Manueller Switch**: Oben rechts kannst du jederzeit umschalten.
    *   Auch der **HTML-Report** ist vollständig übersetzt.
*   **Verbesserte UI**: Optimierter Kontrast für den Language-Switch und verfeinertes Layout.

## 🆕 Was war neu in v2.2.0?

*   **Robustes Error Handling**: Keine "hängenden" Benchmarks mehr. Bei falschen Credentials oder Verbindungsfehlern bricht das Tool sofort ab und zeigt den Fehler an.
*   **Performance Optimierung**: WebDAV-Uploads nutzen nun **25MB Chunks** für bessere Performance bei großen Dateien.
*   **Qualitäts-Ampel**: Ping und Packet Loss werden automatisch bewertet (Grün/Gelb/Rot) und mit einem textuellen Fazit versehen.
*   **Verbesserte UI**: Übersichtlicheres Dashboard mit logischerer Anordnung (Reference Speed oben) und deutlicherer Fehlerdarstellung.

---

## 🛠️ Installation & Downloads

### 📦 Fertige Downloads (Empfohlen)
Lade die aktuellste Version für dein Betriebssystem von der [Releases-Seite](https://github.com/xxroxxerxx/nextcloud-perf/releases) herunter:

*   **Windows**: `.exe` (Einfach doppelklicken)
*   **macOS**: `.pkg` Installer (Signierter Installer für einfache Installation)
*   **Linux**: `.AppImage` (Ausführbar machen und starten)

### 🧑‍💻 Manuell Bauen

1. **Repository klonen:**
   ```bash
   git clone https://github.com/xxroxxerxx/nextcloud-perf.git
   cd nextcloud-perf
   ```

2. **Binary bauen:**
   ```bash
   go build -o nextcloud-perf .
   ```

3. **Starten:**
   ```bash
   ./nextcloud-perf
   ```

---

## 📖 Nutzung

1. Starte das Tool (`./nextcloud-perf` oder Doppelklick).
2. Öffne den Browser unter `http://localhost:3000`.
3. Gib deine Nextcloud-URL, Benutzername und Passwort ein. (Keine Sorge, Credentials bleiben lokal).
4. Klicke auf "Start Benchmark" und warte auf die Ergebnisse.

---

## 🏗️ Architektur

Dieses Projekt ist in Go geschrieben und nutzt eine moderne, modulare Architektur:

*   **Backend**: Go (net/http, native WebDAV implementation)
*   **Frontend**: HTML5/CSS3 (Embedded Templates, Server-Sent Events)
*   **Reporting**: HTML-Template Engine

---

## 📄 Lizenz

Dieses Projekt ist unter der MIT-Lizenz lizenziert. Weitere Details findest du in der [LICENSE](LICENSE)-Datei.

<p align="center">
  <sub>Entwickelt mit ❤️ für die Nextcloud-Community.</sub>
</p>