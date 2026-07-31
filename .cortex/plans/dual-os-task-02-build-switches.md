# Task 02: Build-Schalter für Windows- und Linux-Executable

## Ziel

Ein Befehl pro Zielplattform erzeugt das fertige Binary. Go kompiliert cross-plattform
nativ (`GOOS`-Schalter); alle Dependencies sind reines Go, daher `CGO_ENABLED=0`
(statisches Binary, ideal für schlanke Docker-Images).

## Edits

### 1. Neues `Makefile` im Repo-Root

```make
BINARY := avior-go
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.buildVersion=$(VERSION)

.PHONY: build-windows build-linux build all

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe app.go

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 app.go

build: build-windows

all: build-windows build-linux
```

Hinweise:
- Einstiegspunkt ist `app.go` (package main liegt im Repo-Root; verifiziert via
  `.vscode/launch.json`, das `app.go` als Programm startet).
- `-ldflags "-s -w"` strippt Symbole (kleineres Binary für Container).
- `main.buildVersion` wird nur gesetzt, wenn es die Variable gibt — unverified, ob
  `buildVersion` in app.go existiert. Falls `go build` mit "no such variable" scheitert:
  ldflags auf `-s -w` reduzieren (Fallback, kein Code-Zwang).

### 2. Alternativ-Skripte für Systeme ohne make

`tools/build.ps1` (Windows-Entwicklung, bestehende tools/-Konvention wiederverwenden):

```powershell
param([ValidateSet("windows","linux","all")][string]$Target = "windows")
$env:CGO_ENABLED = "0"
switch ($Target) {
  "windows" { $env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags "-s -w" -o dist/avior-go-windows-amd64.exe app.go }
  "linux"   { $env:GOOS="linux";   $env:GOARCH="amd64"; go build -ldflags "-s -w" -o dist/avior-go-linux-amd64 app.go }
  "all"     { & $PSCommandPath -Target windows; & $PSCommandPath -Target linux }
}
```

`tools/build.sh` (Linux/macOS):

```sh
#!/bin/sh
set -e
target="${1:-linux}"
export CGO_ENABLED=0 GOARCH=amd64
case "$target" in
  windows) GOOS=windows go build -ldflags "-s -w" -o dist/avior-go-windows-amd64.exe app.go ;;
  linux)   GOOS=linux   go build -ldflags "-s -w" -o dist/avior-go-linux-amd64 app.go ;;
  all)     "$0" windows && "$0" linux ;;
  *) echo "usage: $0 [windows|linux|all]" >&2; exit 1 ;;
esac
```

### 3. `.gitignore` ergänzen

`dist/` hinzufügen (Zeile mit `*.exe` existiert bereits, deckt aber `dist/` unter Linux
nicht ab, da dort die Binaries keine `.exe`-Endung haben).

## Verwendung (die "Schalter")

- Windows-Exe:  `make build-windows`  bzw.  `./tools/build.ps1 -Target windows`
- Linux-Binary: `make build-linux`    bzw.  `./tools/build.sh linux`

## Check

Aus dem Repo-Root:
- `make all` erzeugt `dist/avior-go-windows-amd64.exe` UND `dist/avior-go-linux-amd64`.
- `file dist/avior-go-linux-amd64` (unter WSL/Git-Bash) meldet `ELF 64-bit ... statically linked`.
