# Task 02: Build Switches for Windows and Linux Executables

## Goal

One command per target platform produces the final binary. Go compiles cross-platform
natively via the `GOOS` switch; all dependencies are pure Go, therefore `CGO_ENABLED=0`
(static binary, ideal for slim Docker images).

## Edits

### 1. New `Makefile` in repo root

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

Notes:
- Entry point is `app.go` (package main in the repo root; verified via
  `.vscode/launch.json`, which launches `app.go` as the program).
- `-ldflags "-s -w"` strips symbols (smaller binary for containers).
- `main.buildVersion` is only set if the variable exists — unverified whether
  `buildVersion` exists in app.go. If `go build` fails with "no such variable":
  reduce ldflags to `-s -w` (fallback, no code requirement).

### 2. Alternative scripts for systems without make

`tools/build.ps1` (Windows development, reusing existing `tools/` convention):

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

### 3. Update `.gitignore`

Add `dist/` (the `*.exe` line already exists, but doesn't cover `dist/` on Linux where
binaries have no `.exe` extension).

## Usage (the "switches")

- Windows executable: `make build-windows`  or  `./tools/build.ps1 -Target windows`
- Linux binary:      `make build-linux`    or  `./tools/build.sh linux`

## Check

From repo root:
- `make all` produces `dist/avior-go-windows-amd64.exe` AND `dist/avior-go-linux-amd64`.
- `file dist/avior-go-linux-amd64` (under WSL/Git Bash) reports `ELF 64-bit ... statically linked`.
