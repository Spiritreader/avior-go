# Task 00: Branch `feat/dual-os` + Commit Plan Files to Repo

## Goal

All work of the Dual-OS plan happens on a dedicated branch `feat/dual-os`.
The plan itself (index + all task MDs) is versioned as part of the repo under
`.cortex/plans/` so it doesn't exist only session-locally.

## Edits

1. Create and check out branch (from current `master` state):
   ```
   git checkout -b feat/dual-os
   ```
2. Create directory `.cortex/plans/` in the repo root and copy the plan files from the
   session directory there, with exactly these filenames:
   - `.cortex/plans/dual-os-plan.md` (index)
   - `.cortex/plans/dual-os-task-00-branch-plan-files.md` (this file)
   - `.cortex/plans/dual-os-task-01-priority-build-tags.md`
   - `.cortex/plans/dual-os-task-02-build-switches.md`
   - `.cortex/plans/dual-os-task-03-ci-matrix.md`
   - `.cortex/plans/dual-os-task-04-verification.md`
   - `.cortex/plans/dual-os-task-05-docker-compose.md`

   Source of content: the session-local `local://dual-os-*.md` artifacts
   (on disk at
   `C:\Users\MM\.omp\agent\sessions\--C--repos-avior-go--\…\local\`).
   Copy content 1:1, no content changes.
3. Commit `.cortex/plans/`:
   ```
   git add .cortex/plans
   git commit -m "docs: add dual-os build plan"
   ```
   No push — that's up to the user.

## Check

- `git branch --show-current` → `feat/dual-os`.
- `git status` → clean (all 7 files committed).
- All subsequent tasks (01–05) also commit their changes on `feat/dual-os`.
