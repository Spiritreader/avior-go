# avior-go - Agent Instructions

## Project

Go 1.25 media transcoding service. Monitors media directories, compares files via
pluggable comparator modules, and encodes/remuxes content with ffmpeg.

## Build

Single entrypoint: `app.go` in repo root (package `main`).

```
# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o dist/avior-go-windows-amd64.exe app.go

# Linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/avior-go-linux-amd64 app.go
```

Or use `make build-windows` / `make build-linux` / `make all` (requires `make`).

## Architecture

- `app.go` — Main entry, signal handling, logger setup.
- `api/` — HTTP API on port `10000 + Instance` (gorilla/mux, gorilla/websocket).
- `config/` — Config load/save (`config.json` next to binary), priority enums.
- `encoder/` — ffmpeg encoding. OS-specific priority in `priority_*.go` (build tags).
- `comparator/` — Pluggable comparison modules (legacy, resolution, audio, etc.).
- `media/` — File/model definitions.
- `worker/` — Background workers (encode queue, file walker, mover).
- `db/` — MongoDB access.
- `redis/` — Redis pubsub for cross-instance coordination.
- `globalstate/` — Singleton shared state. `ReflectionPath()` = dir of executable.
- `tools/` — Misc helpers (ffprobe, build scripts).
- `structs/`, `consts/`, `joblog/`, `cache/`, `log/` — Support packages.

## Key code conventions

- Config loaded from `filepath.Join(globalstate.ReflectionPath(), "config.json")`.
  The binary's directory IS the config/log root — this matters for Docker volumes.
- All path operations use `filepath.Join` (OS-agnostic).
- Only OS-specific code: `encoder/priority_windows.go` and `encoder/priority_linux.go`
  (build tags `//go:build windows` / `//go:build linux`). No other platform coupling.
- Logging via `github.com/kpango/glg` — log to `glg.FileWriter(...)` and
  `lumberjack.Logger` rotation.
- MongoDB via `go.mongodb.org/mongo-driver/v2`, Redis via `go-redis/v9`.
- Priority constants (`PRIORITY_IDLE`, `PRIORITY_NORMAL`, etc.) are Windows
  `PriorityClass` values; mapped to nice levels on Linux in `priority_linux.go`.
- godirwalk pinned below v1.17.0 (see `go.mod` exclude) — v1.17.0 reports io.EOF
  as a walk error on Windows.

## Docker

`compose.yaml` + `Dockerfile` in repo root for Komodo deployment.
Binary runs from `/data` inside container (entrypoint copies to persist config/logs).
Mount `/mnt/user/media:/media` for media access.
API on port `10000`.

## Language

All documentation, plans, and commit messages MUST be in English.
Code comments may be in either language, but public-facing and plan text is English only.

## Skills

### docs_agent — Technical Writer

Expert technical writer for this project. Read code from the repo and generate or
update documentation in `docs/`.

- **Tech stack:** Go 1.25, MongoDB, Redis, ffmpeg, Docker.
- **File structure:** `docs/` is the documentation root (create if missing).
- **Style:** Concise, specific, value-dense. Write so a new developer can
  understand — don't assume audience expertise in the topic.
- **Boundaries:**
  - Write new files to `docs/`.
  - Ask before major changes to existing documents.
  - Do not modify source code or config files.
  - Do not commit secrets.

## Never do

- Do not import `golang.org/x/sys/windows` in cross-platform files — use build-tagged
  files instead.
- Do not hardcode UNC paths (`\\UMS\...`) or drive letters in code — those are user
  config data.
- Do not upgrade `github.com/karrick/godirwalk` to v1.17.0 (see go.mod).
