# Plan: Dual-OS Build for avior-go (Windows + Linux)

## Context

avior-go currently only compiles for Windows because `encoder/encoder.go` directly uses
the Windows API (`golang.org/x/sys/windows`: `OpenProcess`, `SetPriorityClass`,
`CloseHandle`) to set the process priority of the spawned `ffmpeg` process.
Goal: The same code should optionally produce a Windows executable (`avior-go.exe`) or a
Linux binary (`avior-go`) via Go build tags / `GOOS`, so that avior-go instances can run
under Linux in Docker in the future.

Facts from the code (verified in this session):
- Only Windows dependency in the entire repo: `encoder/encoder.go` lines 176–188
  (`windows.OpenProcess` / `windows.SetPriorityClass` / `windows.CloseHandle`), plus the
  import `"golang.org/x/sys/windows"` on line 23.
- All path operations already use `filepath.Join` (app.go, config/config.go,
  api/api.go) — OS-agnostic. UNC paths exist only in config JSONs (user data, not a
  code problem).
- All Go dependencies in `go.mod` are pure Go (gorilla, redis, mongo-driver, glg,
  godirwalk, lumberjack …) → `CGO_ENABLED=0` cross-compile is possible, no C toolchain
  needed.
- CI: `.github/workflows/go.yml` currently builds only `goos: windows`, `goarch: amd64`
  via `wangyoucao577/go-release-action@v1.18`.
- Go version per go.mod: `go 1.25.0`.

End state: `GOOS=windows go build` and `GOOS=linux go build` both work; ffmpeg priority
is set per OS via build-tag files (Windows: PriorityClass, Linux: nice level); CI builds
both artifacts.

## Tasks (one MD file each)

0. `dual-os-task-00-branch-plan-files.md` — Create branch `feat/dual-os`; copy plan +
   all task MDs to `.cortex/plans/` in the repo and commit.
1. `dual-os-task-01-priority-build-tags.md` — Extract OS-specific priority setting
   from `encoder/encoder.go` into build-tag files.
2. `dual-os-task-02-build-switches.md` — Build switches (Makefile + scripts) for
   Windows and Linux executables.
3. `dual-os-task-03-ci-matrix.md` — Extend GitHub Actions workflow to a
   Windows+Linux matrix.
4. `dual-os-task-04-verification.md` — Verification: cross-compile both targets,
   tests, smoke check.
5. `dual-os-task-05-docker-compose.md` — Dockerfile + compose.yaml for Komodo deployment
   (mount `/mnt/user/media` → `/media`).

Dependencies: Task 00 first (branch + plan placement), then Task 01 (otherwise Linux
won't compile). Tasks 02 and 03 are independent of each other, but both require Task 01.
Task 05 requires Task 01 (uses the same Linux build). Task 04 last.
