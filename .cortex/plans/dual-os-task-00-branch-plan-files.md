# Task 00: Branch `feat/dual-os` + Plan-Dateien ins Repo ablegen

## Ziel

Alle Arbeiten des Dual-OS-Plans laufen auf einem eigenen Branch `feat/dual-os`.
Der Plan selbst (Index + alle Task-MDs) wird als Teil des Repos unter
`.cortex/plans/` versioniert, damit er nicht nur session-lokal existiert.

## Edits

1. Branch anlegen und auschecken (vom aktuellen `master`-Stand):
   ```
   git checkout -b feat/dual-os
   ```
2. Verzeichnis `.cortex/plans/` im Repo-Root anlegen und die Plan-Dateien aus dem
   Session-Verzeichnis dorthin kopieren, mit exakt diesen Dateinamen:
   - `.cortex/plans/dual-os-plan.md` (Index)
   - `.cortex/plans/dual-os-task-00-branch-plan-files.md` (dieses File)
   - `.cortex/plans/dual-os-task-01-priority-build-tags.md`
   - `.cortex/plans/dual-os-task-02-build-switches.md`
   - `.cortex/plans/dual-os-task-03-ci-matrix.md`
   - `.cortex/plans/dual-os-task-04-verification.md`
   - `.cortex/plans/dual-os-task-05-docker-compose.md`

   Quelle der Inhalte: die session-lokalen `local://dual-os-*.md`-Artefakte
   (auf Disk unter
   `C:\Users\MM\.omp\agent\sessions\--C--repos-avior-go--\2026-07-31T07-42-50-771Z_019fb720-3b93-7000-b9d1-372cb7334f9e\local\`).
   Inhalt 1:1 übernehmen, keine inhaltlichen Änderungen.
3. `.cortex/plans/` committen:
   ```
   git add .cortex/plans
   git commit -m "docs: add dual-os build plan"
   ```
   Kein Push — das obliegt dem Benutzer.

## Check

- `git branch --show-current` → `feat/dual-os`.
- `git status` → clean (alle 7 Dateien committet).
- Alle folgenden Tasks (01–05) committen ihre Änderungen ebenfalls auf `feat/dual-os`.
