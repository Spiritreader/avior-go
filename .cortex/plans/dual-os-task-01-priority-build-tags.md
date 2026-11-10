# Task 01: OS-specific ffmpeg Priority via Build Tags

## Goal

`encoder/encoder.go` won't compile under Linux (direct import of
`golang.org/x/sys/windows`). The priority setting is extracted into two build-tag files;
`encoder.go` only calls an OS-neutral helper function.

## Edits

### 1. New file `encoder/priority_windows.go`

```go
//go:build windows

package encoder

import (
	"os/exec"

	"github.com/Spiritreader/avior-go/config"
	"github.com/kpango/glg"
	"golang.org/x/sys/windows"
)

// setProcessPriority sets the Windows PriorityClass of the spawned ffmpeg process.
// Errors are only logged (as before), since the encoding itself doesn't depend on it.
func setProcessPriority(cmd *exec.Cmd, cfg *config.Data) {
	hProcess, err := windows.OpenProcess(0x0400|0x0200, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = glg.Warnf("could not get ffmpeg handle using pid %d, err: %s", cmd.Process.Pid, err)
		return
	}
	defer func() {
		if err := windows.CloseHandle(hProcess); err != nil {
			_ = glg.Errorf("could not close handle for pid %d, err: %s", cmd.Process.Pid, err)
		}
	}()
	if err := windows.SetPriorityClass(hProcess, config.PriorityUint32(cfg.Local.EncoderPriority)); err != nil {
		_ = glg.Warnf("could not set priority %s for ffmpeg handle using pid %d, err: %s",
			cfg.Local.EncoderPriority, cmd.Process.Pid, err)
	}
}
```

Behavior change over the original: `return` after `OpenProcess` error (avoids
`SetPriorityClass` on invalid handle) and `CloseHandle` via `defer`. This is the
correct version of the existing code — no functional change in the success case. Log
strings remain identical, except the `SetPriorityClass` log outputs
`cfg.Local.EncoderPriority` instead of `cfg.Local.EncoderConfig` (fix for a copy-paste
error in the original, line 183).

### 2. New file `encoder/priority_linux.go`

```go
//go:build linux

package encoder

import (
	"os/exec"
	"syscall"

	"github.com/Spiritreader/avior-go/config"
	"github.com/kpango/glg"
)

// niceLevel maps the configured Windows priority level to a Linux nice value.
// Mapping: HIGH/ABOVE_NORMAL -> -5 (higher priority, requires root/CAP_SYS_NICE),
// NORMAL -> 0, BELOW_NORMAL -> 10, IDLE -> 19.
// Unknown values fall back to 19 (idle) — analogous to the Windows fallback in
// config.PriorityUint32, which returns IDLE for unknown values.
func niceLevel(priority string) int {
	switch priority {
	case config.PRIORITY_HIGH.String(), config.PRIORITY_ABOVE_NORMAL.String():
		return -5
	case config.PRIORITY_NORMAL.String():
		return 0
	case config.PRIORITY_BELOW_NORMAL.String():
		return 10
	default:
		return 19
	}
}

// setProcessPriority sets the nice level of the spawned ffmpeg process.
// An error (e.g. missing permissions for negative nice values in a Docker container)
// is only logged as a warning; encoding proceeds normally.
func setProcessPriority(cmd *exec.Cmd, cfg *config.Data) {
	if err := syscall.Setpriority(syscall.PRIO_PROCESS, cmd.Process.Pid, niceLevel(cfg.Local.EncoderPriority)); err != nil {
		_ = glg.Warnf("could not set priority %s for ffmpeg process with pid %d, err: %s",
			cfg.Local.EncoderPriority, cmd.Process.Pid, err)
	}
}
```

Note: In Docker without `CAP_SYS_NICE`, negative nice values will fail — this is
acceptable (warning in log, encoding runs with default priority). The container docs
may later recommend `--cap-add SYS_NICE`, but that is not part of this plan.

### 3. Adapt `encoder/encoder.go`

- Remove import `"golang.org/x/sys/windows"` (line 23). `config` stays imported
  (still used elsewhere in the file — verified: `cfg.Local.*` throughout).
- Replace the block on lines 176–188 (the three `windows.*` calls with error logs) with:

```go
	setProcessPriority(cmd, cfg)
```

The call goes directly after the successful `cmd.Start()`, exactly where the old block
was.

## No further callsites

`grep` for `x/sys` and `windows\.` in the entire repo: only `encoder/encoder.go`.
`config.PriorityUint32` remains unchanged (still used by `priority_windows.go`).
Clean cutover: no compatibility code.

## Check

- `GOOS=windows go build ./...` and `GOOS=linux go build ./...` from the repo root
  both compile without errors.
- `go vet ./encoder` without findings.
