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
