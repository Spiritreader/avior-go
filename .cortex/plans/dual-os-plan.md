# Plan: Dual-OS-Build für avior-go (Windows + Linux)

## Context

avior-go kompiliert aktuell nur für Windows, weil `encoder/encoder.go` die Windows-API
(`golang.org/x/sys/windows`: `OpenProcess`, `SetPriorityClass`, `CloseHandle`) direkt
verwendet, um die Prozess-Priorität des gestarteten `ffmpeg`-Prozesses zu setzen.
Ziel: Der gleiche Code soll per Schalter (Go-Build-Tags / `GOOS`) wahlweise ein
Windows-Executable (`avior-go.exe`) oder ein Linux-Binary (`avior-go`) erzeugen, damit
avior-go-Instanzen künftig unter Linux in Docker laufen können.

Fakten aus dem Code (verifiziert in dieser Session):
- Einzige Windows-Abhängigkeit im gesamten Repo: `encoder/encoder.go` Zeilen 176–188
  (`windows.OpenProcess` / `windows.SetPriorityClass` / `windows.CloseHandle`), plus Import
  `"golang.org/x/sys/windows"` in Zeile 23.
- Alle Pfad-Operationen nutzen bereits `filepath.Join` (app.go, config/config.go,
  api/api.go) — OS-agnostisch. UNC-Pfade existieren nur in config-JSONs (Benutzerdaten,
  kein Code-Problem).
- Alle Go-Dependencies in `go.mod` sind reines Go (gorilla, redis, mongo-driver, glg,
  godirwalk, lumberjack …) → `CGO_ENABLED=0`-Cross-Compile ist möglich, kein C-Toolchain nötig.
- CI: `.github/workflows/go.yml` baut via `wangyoucao577/go-release-action@v1.18` aktuell
  nur `goos: windows`, `goarch: amd64`.
- Go-Version laut go.mod: `go 1.25.0`.

Endzustand: `GOOS=windows go build` und `GOOS=linux go build` funktionieren beide;
die ffmpeg-Priorität wird pro OS über Build-Tag-Dateien gesetzt (Windows: PriorityClass,
Linux: nice-Level); CI baut beide Artefakte.

## Tasks (je ein MD-File)

0. `dual-os-task-00-branch-plan-files.md` — Branch `feat/dual-os` anlegen; Plan +
   alle Task-MDs unter `.cortex/plans/` ins Repo kopieren und committen.
1. `dual-os-task-01-priority-build-tags.md` — OS-spezifische Prioritäts-Setzung per
   Build-Tags aus `encoder/encoder.go` extrahieren.
2. `dual-os-task-02-build-switches.md` — Build-Schalter (Makefile + Skripte) für
   Windows- bzw. Linux-Executable.
3. `dual-os-task-03-ci-matrix.md` — GitHub-Actions-Workflow auf Windows+Linux-Matrix
   erweitern.
4. `dual-os-task-04-verification.md` — Verifikation: Cross-Compile beider Targets,
   Tests, Smoke-Check.
5. `dual-os-task-05-docker-compose.md` — Dockerfile + compose.yaml für Komodo-Deployment
   (Mount `/mnt/user/media` → `/media`).

Abhängigkeiten: Task 00 zuerst (Branch + Plan-Ablage), dann Task 01 (sonst kompiliert
Linux nicht). Tasks 02 und 03 sind unabhängig voneinander, setzen aber Task 01 voraus.
Task 05 setzt Task 01 voraus (nutzt denselben Linux-Build). Task 04 zuletzt.
