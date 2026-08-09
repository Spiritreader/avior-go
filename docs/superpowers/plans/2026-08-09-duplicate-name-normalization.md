# Duplicate Name Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make duplicate lookup treat media names as equal when their normalized comparison keys contain the same Unicode letters and digits, regardless of whitespace or punctuation.

**Architecture:** Add a pure helper in `media` that derives a comparison key without mutating a `File` or its path. Use that helper in both worker duplicate lookup paths, so fresh filesystem scans and the in-memory cache apply exactly the same normalization. Keep configured extensions as a separate exact comparison component.

**Tech Stack:** Go 1.25, standard-library `unicode`, `strings`, existing `media` and `worker` packages, Go tests.

## Global Constraints

- Only comparison keys are normalized; stored paths, displayed names, output names, and cache entries remain unchanged.
- Retain Unicode letters and Unicode digits; remove whitespace, punctuation, symbols, hyphens, underscores, and all other non-letter/non-digit runes.
- Compare case-insensitively.
- Preserve configured media-extension handling; do not make `.mkv` and another extension equal.
- Do not alter duplicate modules after a match is found.
- Do not add dependencies or change configuration formats.

---

### Task 1: Add pure duplicate-name comparison key

**Files:**
- Modify: `media/media.go` near `OutName()` or another name helper
- Test: `media/media_test.go`

**Interfaces:**
- Produces: `func DuplicateNameKey(name string) string`, returning a case-folded key containing only Unicode letters and digits.

- [ ] **Step 1: Write failing tests**

Add a table-driven test for basenames (the worker compares the extension separately):

```go
func TestDuplicateNameKey(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"aktiv und gesund - Faszientherapie - Poolkeime - Stand-up-Paddling", "aktivundgesundfaszientherapiepoolkeimestanduppaddling"},
		{"aktiv und gesund   Faszientherapie   Poolkeime   Stand-up-Paddling", "aktivundgesundfaszientherapiepoolkeimestanduppaddling"},
		{"aktiv und gesund _ Faszientherapie _ Poolkeime _ Stand-up-Paddling", "aktivundgesundfaszientherapiepoolkeimestanduppaddling"},
		{"Ä Ö Ü ß 2024", "äöüß2024"},
		{"A/B", "ab"},
		{"AB", "ab"},
		{"Different", "different"},
	}
	for _, tt := range tests {
		original := tt.name
		if got := DuplicateNameKey(tt.name); got != tt.want {
			t.Errorf("DuplicateNameKey(%q) = %q, want %q", tt.name, got, tt.want)
		}
		if tt.name != original {
			t.Errorf("DuplicateNameKey mutated input %q", original)
		}
	}
}
```

The first three cases are the supplied production variants. The `A/B` and `AB` cases document the accepted trade-off that punctuation-only differences compare equal.

- [ ] **Step 2: Run the focused test and verify it fails**

Run:

```text
go test ./media/ -run TestDuplicateNameKey -v
```

Expected: FAIL because `DuplicateNameKey` does not yet exist.

- [ ] **Step 3: Implement the minimal pure helper**

Implement the helper using a `strings.Builder` and rune iteration. For each rune, keep it when `unicode.IsLetter(r)` or `unicode.IsDigit(r)`, write `unicode.ToLower(r)`, and skip everything else. Do not mutate any `File` field.

- [ ] **Step 4: Run focused tests**

Run:

```text
go test ./media/ -run TestDuplicateNameKey -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```text
git add media/media.go media/media_test.go
git commit -m "feat: add normalized duplicate name keys"
```

---

### Task 2: Apply the same key to fresh and cached duplicate scans

**Files:**
- Modify: `worker/worker.go:631-645` (`traverseMemCache`)
- Modify: `worker/worker.go:648-685` (`traverseDir`)
- Test: `worker/worker_test.go` or an existing worker test file, using the repository's current test conventions

**Interfaces:**
- Consumes: `media.DuplicateNameKey(string)` from Task 1.
- Produces: both duplicate lookup paths compare normalized basenames plus the configured extension, while returning the original path unchanged.

- [ ] **Step 1: Write failing comparison tests**

Add tests that exercise the matching predicate used by both scan paths. The tests must assert that the three supplied names match one another, that their returned path/name remains original, and that a different retained-letter/digit key does not match. Include the configured extension in the comparison so `.mkv` remains distinct from a different extension.

- [ ] **Step 2: Run focused worker tests and verify failure**

Run:

```text
go test ./worker/ -run "Test.*Duplicate|Test.*Name" -v
```

Expected: FAIL for at least the separator-normalization case before the scan predicates are changed.

- [ ] **Step 3: Implement one shared matching predicate**

Use a local helper in `worker` (or a clearly named package helper if existing conventions require it) that compares:

```go
media.DuplicateNameKey(filepath.Base(candidateWithoutExtension)) ==
media.DuplicateNameKey(filepath.Base(existingWithoutExtension))
```

and compares the configured extension separately. Apply that exact predicate in `traverseMemCache` and `traverseDir`; do not duplicate subtly different normalization code. Keep `matches = append(matches, media.File{Path: path})` unchanged so downstream code receives the original path.

- [ ] **Step 4: Run focused and package tests**

Run:

```text
go test ./worker/ -run "Test.*Duplicate|Test.*Name" -v
go test ./media/ ./worker/
```

Expected: PASS.

- [ ] **Step 5: Commit**

```text
git add worker/worker.go worker/worker_test.go
git commit -m "feat: normalize names during duplicate scans"
```

---

### Task 3: Verify the complete duplicate behavior

**Files:**
- Modify: none unless a test needs a small convention-aligned correction
- Test: existing `media` and `worker` tests

- [ ] **Step 1: Run the full repository test suite**

Run:

```text
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Build the service**

Run:

```text
go build ./...
```

Expected: successful build.

- [ ] **Step 3: Inspect the final behavior**

Confirm that tests demonstrate all of the following:

- all three supplied variants match;
- internal hyphens and punctuation are removed from keys;
- Unicode letters and digits remain;
- unrelated retained letter/digit sequences remain different;
- original file names and paths are returned unchanged;
- extension differences do not match.

- [ ] **Step 4: Commit any test-only correction**

If Task 3 exposes a test-only issue, correct it and commit with:

```text
git add media worker
git commit -m "test: verify normalized duplicate matching"
```
