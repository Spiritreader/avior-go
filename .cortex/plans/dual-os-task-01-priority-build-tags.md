# Task 01: OS-spezifische ffmpeg-Priorität per Build-Tags

## Ziel

`encoder/encoder.go` kompiliert unter Linux nicht (direkter Import von
`golang.org/x/sys/windows`). Die Prioritäts-Setzung wird in zwei Build-Tag-Dateien
ausgelagert; `encoder.go` ruft nur noch eine OS-neutrale Hilfsfunktion auf.

## Edits

### 1. Neue Datei `encoder/priority_windows.go`

```go
//go:build windows

package encoder

import (
	"os/exec"

	"github.com/Spiritreader/avior-go/config"
	"github.com/kpango/glg"
	"golang.org/x/sys/windows"
)

// setProcessPriority setzt die Windows-PriorityClass des gestarteten ffmpeg-Prozesses.
// Fehler werden nur geloggt (wie bisher), da das Encoding selbst davon nicht abhängt.
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

Verhaltensänderung gegenüber dem Original: `return` nach dem `OpenProcess`-Fehler
(vermeidet `SetPriorityClass` auf ungültigem Handle) und `CloseHandle` via `defer`.
Das ist die korrekte Version des bisherigen Codes — keine funktionale Änderung im
Erfolgsfall. Log-Strings bleiben identisch, außer dass im `SetPriorityClass`-Log
`cfg.Local.EncoderPriority` statt `cfg.Local.EncoderConfig` ausgegeben wird
(Fix eines Copy-Paste-Fehlers im Original, Zeile 183).

### 2. Neue Datei `encoder/priority_linux.go`

```go
//go:build linux

package encoder

import (
	"os/exec"
	"syscall"

	"github.com/Spiritreader/avior-go/config"
	"github.com/kpango/glg"
)

// niceLevel bildet die konfigurierte Windows-Prioritätsstufe auf ein Linux-nice-Level ab.
// Mapping: HIGH/ABOVE_NORMAL -> -5 (höhere Prio, erfordert root/CAP_SYS_NICE),
// NORMAL -> 0, BELOW_NORMAL -> 10, IDLE -> 19.
// Unbekannte Werte fallen auf 19 (idle) zurück — analog zum Windows-Fallback in
// config.PriorityUint32, das bei unbekannten Werten IDLE liefert.
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

// setProcessPriority setzt das nice-Level des gestarteten ffmpeg-Prozesses.
// Ein Fehler (z. B. fehlende Rechte für negative nice-Werte im Docker-Container)
// wird nur gewarnt geloggt; das Encoding läuft normal weiter.
func setProcessPriority(cmd *exec.Cmd, cfg *config.Data) {
	if err := syscall.Setpriority(syscall.PRIO_PROCESS, cmd.Process.Pid, niceLevel(cfg.Local.EncoderPriority)); err != nil {
		_ = glg.Warnf("could not set priority %s for ffmpeg process with pid %d, err: %s",
			cfg.Local.EncoderPriority, cmd.Process.Pid, err)
	}
}
```

Hinweis: In Docker ohne `CAP_SYS_NICE` schlagen negative nice-Werte fehl — das ist
akzeptabel (Warnung im Log, Encoding läuft mit Default-Priorität). In der
Container-Doku später ggf. `--cap-add SYS_NICE` empfehlen, ist aber nicht Teil dieses Plans.

### 3. `encoder/encoder.go` anpassen

- Import `"golang.org/x/sys/windows"` (Zeile 23) entfernen. `config` bleibt importiert
  (wird weiterhin anderswo in der Datei genutzt — verifiziert: `cfg.Local.*` überall).
- Den Block Zeilen 176–188 (die drei `windows.*`-Aufrufe samt Fehler-Logs) ersetzen durch:

```go
	setProcessPriority(cmd, cfg)
```

Der Aufruf steht direkt nach dem erfolgreichen `cmd.Start()`, exakt an der Stelle des
alten Blocks.

## Keine weiteren Callsites

`grep` nach `x/sys` und `windows\.` im gesamten Repo: nur `encoder/encoder.go`.
`config.PriorityUint32` bleibt unverändert (wird weiter von `priority_windows.go`
genutzt). Clean Cutover: kein Compatibility-Code.

## Check

- `GOOS=windows go build ./...` und `GOOS=linux go build ./...` aus dem Repo-Root
  kompilieren beide fehlerfrei.
- `go vet ./encoder` ohne Befund.
