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
