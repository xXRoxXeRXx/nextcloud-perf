# nextcloud-perf

Ein Toolset zur Performance-Analyse und Benchmarking von Nextcloud-Instanzen.

## Features
- Netzwerk-Latenz- und Bandbreitenmessung
- Systeminformationen und Ressourcenüberwachung
- Benchmark-Runner für verschiedene Testszenarien
- Web-Oberfläche zur Auswertung

## Installation

1. Go 1.21 oder neuer installieren
2. Repository klonen:
   ```sh
   git clone https://github.com/xxroxxerxx/nextcloud-perf.git
   cd nextcloud-perf
   ```
3. Build:
   ```sh
   go build -o nextcloud-perf main.go
   ```


## Nutzung

### Kommandozeile

```sh
./nextcloud-perf --help
```

### Web-Oberfläche

Nach dem Start öffnet sich automatisch die Weboberfläche unter [http://localhost:3000](http://localhost:3000):

```sh
./nextcloud-perf
```
Die wichtigsten Funktionen sind dann über die Web-UI erreichbar.


## Build & Ausführung

### Voraussetzungen
- Go 1.21 oder neuer (empfohlen: 1.24)

### Build
Im Projektverzeichnis:
```sh
go build -o nextcloud-perf main.go
```

### Starten
```sh
./nextcloud-perf
```
Die Weboberfläche öffnet sich automatisch unter http://localhost:3000

## Projektstruktur
- `internal/` – Kernmodule für Benchmark, Netzwerk, System, Reporting, UI
- `web/` – Web-Frontend und Templates
- `main.go` – Einstiegspunkt
- `go.mod` – Go-Moduldefinition

## Lizenz
Siehe LICENSE-Datei.

---

> Bitte ergänze die README bei Bedarf um weitere Details oder Beispiele.
