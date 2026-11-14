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

## File permissions (all files the process creates)

Everything the app writes must be readable **and writable by every authorized host
user** — logs in `/data/log`, `.INFO.log` next to the media, `config.json`. The
container runs as `nobody:users` (PUID/PGID) and its output is shared with SMB
users and other tools (`add_rating_ui`, `avior-dedup`, scrapers). The matching rule
in `add_rating_ui` is `process.umask(0)` ("Dateien 666, world-writable").

- **Files: `0666`. Directories: `0775`/`0777`.** Do not ship anything stricter.
- `docker/entrypoint.sh` sets `umask 000` for that reason — do not lower it back.
- **A umask is not enough**: it only removes permission bits. Loggers that pass a
  mode to `os.OpenFile` keep it. `lumberjack` hardcodes `0600`, so every file it
  writes (`main.log`, rotated `main-<ts>.log`, `processed.log`, `skipped.log`,
  `.INFO.log`) is chmodded to `0666` explicitly.
- `app.go: ensureLogPermissions` re-applies the mode for `main.log`/`err.log`
  every 30 s because lumberjack rotates at 10 MB and would otherwise leave a
  `0600` file behind.
- When adding a new writer (logger, exporter, cache dump), set `0666` at creation
  **or** chmod after the first write — and add a test that asserts the mode.

## Docker

`compose.yaml` + `Dockerfile` in repo root for Komodo deployment.
Binary runs from `/data` inside container (entrypoint copies to persist config/logs).
Mount `/mnt/user/media:/media` for media access.
API on port `10000`.

## Language

All documentation, plans, and commit messages MUST be in English.
Code comments may be in either language, but public-facing and plan text is English only.

## Subagent routing

Project-level rules OVERRIDE the user-level global rules in
`~/.omp/agent/AGENTS.md` for this repository.

When delegating work via subagents (`task`), route by complexity:

- **Low complexity** (mechanical edits, config changes, file creation) →
  `deepseek-v4-flash` with high effort
- **Medium complexity** (refactoring, new features with existing patterns) →
  `deepseek-v4-flash` with max effort
- **High complexity** (novel design, architectural changes, debugging unknown issues) →
  `kimi-k3` with low effort


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
