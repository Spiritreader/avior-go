# Task 04: Verification of the Dual-OS Build

## Goal

Prove that both targets compile from the same code and that the behavior of the
changed code paths (Task 01) is unchanged or correct.

Working directory for all commands: repo root `C:/repos/avior-go`. No env vars or
fixtures needed; ffmpeg/ffprobe are not required for the pure build checks.

## Checks (in this order)

1. **Both targets compile** (exercises Tasks 01 + 02):
   ```
   CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/avior-go-windows-amd64.exe app.go
   CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o dist/avior-go-linux-amd64 app.go
   ```
   Expected: both commands exit 0, both files exist. Before Task 01, the Linux command
   failed with `build constraints exclude all Go files ... x/sys/windows` — exactly
   this error must be gone.

2. **Linux binary is static** (Docker suitability):
   ```
   file dist/avior-go-linux-amd64
   ```
   Expected: `ELF 64-bit LSB executable, x86-64, ... statically linked`.
   (`file` via Git Bash/WSL; if unavailable: skip — building with `CGO_ENABLED=0`
   guarantees static linking.)

3. **Vet + existing tests** (regression):
   ```
   go vet ./...
   go test ./...
   ```
   Expected: no new findings; existing tests (including the `config` package) pass.
   Windows-specific paths are tested on the dev Windows machine; the Linux path
   (`priority_linux.go`) has no unit tests — intentional, it only wraps a syscall
   with logging.

4. **Windows smoke test (behavior unchanged)**:
   Start `avior-go-windows-amd64.exe` with the existing `config_dev.json`, run an
   encode job, and verify in the log (`log/main.log`) that NO new warning
   `could not set priority ... for ffmpeg handle` appears — priority setting works
   as before.

5. **Linux smoke test (new behavior)**:
   Start the Linux binary in a container or via WSL (`./avior-go-linux-amd64` alongside
   a `config.json` with Linux paths in `MediaPaths`/`OutDirectory` and reachable
   MongoDB/Redis). Once an encode job runs: the log must not contain a
   `could not set priority` warning (unless the container runs without permissions
   for negative nice values at `HIGH`/`ABOVE_NORMAL` — then exactly one warning is
   acceptable and documented; encoding continues).
   Additionally during an encode: `ps -o pid,ni,cmd -C ffmpeg` — the `NI` field must
   match the configured mapping (IDLE → 19, NORMAL → 0, BELOW_NORMAL → 10).

## Abort criteria / fallbacks

- Check 1 fails for Linux with import error on `x/sys/windows`: Task 01 incomplete
  (import not removed or build tag missing/misspelled — tag must be exactly
  `//go:build windows` or `//go:build linux` as the first line).
- Check 3 fails in `encoder` package: signature
  `setProcessPriority(cmd *exec.Cmd, cfg *config.Data)` must be identical in both files.
