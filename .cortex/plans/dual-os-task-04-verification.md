# Task 04: Verifikation des Dual-OS-Builds

## Ziel

Beweisen, dass beide Targets aus demselben Code kompilieren und das Verhalten der
geänderten Code-Pfade (Task 01) unverändert bzw. korrekt ist.

Arbeitsverzeichnis für alle Kommandos: Repo-Root `C:/repos/avior-go`. Keine Env-Vars
oder Fixtures nötig; ffmpeg/ffprobe werden für die reinen Build-Checks nicht gebraucht.

## Checks (in dieser Reihenfolge)

1. **Beide Targets kompilieren** (übt Task 01 + 02):
   ```
   CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/avior-go-windows-amd64.exe app.go
   CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o dist/avior-go-linux-amd64 app.go
   ```
   Erwartung: beide Befehle exit 0, beide Dateien existieren. Vor Task 01 schlug der
   Linux-Befehl mit `build constraints exclude all Go files ... x/sys/windows` fehl —
   genau dieser Fehler muss weg sein.

2. **Linux-Binary ist statisch** (Docker-Tauglichkeit):
   ```
   file dist/avior-go-linux-amd64
   ```
   Erwartung: `ELF 64-bit LSB executable, x86-64, ... statically linked`.
   (`file` via Git-Bash/WSL; falls nicht vorhanden: `go tool nm` nicht nötig, Schritt
   entfällt ersatzlos — der Build mit `CGO_ENABLED=0` garantiert statisch.)

3. **Vet + bestehende Tests** (Regression):
   ```
   go vet ./...
   go test ./...
   ```
   Erwartung: kein neuer Befund; vorhandene Tests (u. a. `config`-Package) grün.
   Windows-spezifische Pfade werden auf dem Dev-Windows getestet; der Linux-Pfad
   (`priority_linux.go`) hat keine Unit-Tests — bewusst, er wrappt nur einen Syscall
   mit Logging.

4. **Smoke-Test Windows (Verhalten unverändert)**: `avior-go-windows-amd64.exe` mit der
   bestehenden `config_dev.json` starten, einen Encode-Job laufen lassen und im Log
   (`log/main.log`) prüfen, dass KEINE neue Warnung
   `could not set priority ... for ffmpeg handle` erscheint — Prioritäts-Setzung
   funktioniert also wie vorher.

5. **Smoke-Test Linux (neues Verhalten)**: das Linux-Binary in einem Container oder per
   WSL starten (`./avior-go-linux-amd64` neben einer `config.json` mit Linux-Pfaden in
   `MediaPaths`/`OutDirectory` und erreichbarem MongoDB/Redis). Sobald ein Encode-Job
   läuft: im Log darf keine `could not set priority`-Warnung auftauchen (außer der
   Container läuft ohne Rechte für negative nice-Werte bei `HIGH`/`ABOVE_NORMAL` —
   dann ist genau eine Warnung akzeptabel und dokumentiert, Encoding läuft weiter).
   Zusätzlich während eines Encodes: `ps -o pid,ni,cmd -C ffmpeg` — das `NI`-Feld muss
   dem konfigurierten Mapping entsprechen (IDLE → 19, NORMAL → 0, BELOW_NORMAL → 10).

## Abbruchkriterien / Fallbacks

- Schlägt Check 1 für Linux fehl mit Import-Fehler auf `x/sys/windows`: Task 01 nicht
  vollständig (Import nicht entfernt oder Build-Tag fehlt/falsch geschrieben —
  Tag muss exakt `//go:build windows` bzw. `//go:build linux` als erste Zeile sein).
- Schlägt Check 3 in `encoder`-Package: Signatur `setProcessPriority(cmd *exec.Cmd, cfg *config.Data)`
  muss in beiden Dateien identisch sein.
