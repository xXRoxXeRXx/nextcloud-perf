# Nextcloud Perf

![Nextcloud Perf Logo](assets/logo.png)

**Ein leistungsstarkes Toolset zur detaillierten Performance-Analyse und Benchmarking von Nextcloud-Instanzen.**

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue?style=for-the-badge)
![Release](https://img.shields.io/github/v/release/xxroxxerxx/nextcloud-perf?style=for-the-badge)

---

## 🚀 Überblick

`nextcloud-perf` hilft dir dabei, Engpässe in deiner Nextcloud-Umgebung zu identifizieren. Ob Netzwerklatenz, langsame WebDAV-Operationen oder Ressourcenmangel auf dem Server – dieses Tool liefert dir die nötigen Daten direkt in einer übersichtlichen Weboberfläche.

## ✨ Kernfunktionen

| Kategorie | Features |
| :--- | :--- |
| **🌐 Netzwerk** | SSL/TLS Handshake, VPN/Proxy Detection, MTU Estimation & Latency/Packet Loss Analysis |
| **📁 WebDAV** | Upload/Download-Benchmark mit Chunking & Unterstützung für große Dateien |
| **💻 System** | Client-side Disk I/O Benchmarks & CPU Monitoring während der Transfers |
| **🧠 Analyse** | Automatische Qualitätsbewertung ("Exzellent", "Solide", "Optimierungsbedarf") |
| **📊 Reporting** | Interaktives Dashboard & detaillierte HTML-Reports (DE/EN) |

---

## 🛠️ Installation & Downloads

### 📦 Fertige Downloads (Empfohlen)

Lade die aktuellste Version für dein Betriebssystem von der [Releases-Seite](https://github.com/xxroxxerxx/nextcloud-perf/releases) herunter:

- **Windows**: `.exe` (Einfach doppelklicken)
- **macOS**: `.pkg` Installer
- **Linux**: `.AppImage` (Ausführbar machen und starten)

### 🧑‍💻 Manuell Bauen

1. **Repository klonen:**

   ```bash
   git clone https://github.com/xxroxxerxx/nextcloud-perf.git
   cd nextcloud-perf
   ```
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
3. Gib Nextcloud-URL, Benutzername und Passwort ein. (Credentials bleiben lokal).
4. Klicke auf "Start Benchmark" und analysiere die Ergebnisse.

---

## 🏗️ Architektur

Dieses Projekt ist in Go geschrieben und nutzt eine moderne, modulare Architektur:

- **Backend**: Go (net/http, native WebDAV implementation)
- **Frontend**: HTML5/CSS3 (Embedded Templates, Server-Sent Events)
- **Reporting**: HTML-Template Engine

---

## 📄 Lizenz

Dieses Projekt ist unter der MIT-Lizenz lizenziert. Weitere Details findest du in der [LICENSE](LICENSE)-Datei.
