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
