# Plan: Year-aware Duplicate Detection for avior-go (config-driven)

## Context

Two files can share the exact same title (e.g. "Die Löwin") but be different films
(release years 2020 vs 2024). Current duplicate detection (`worker/worker.go`
`traverseDir`/`traverseMemCache`) matches only the exact `OutName()+Ext`, so a
same-title/different-year film is treated as a duplicate instead of a new film.
Goal: when a new job's title exists in the library with a **different release year**,
the new film gets the year appended in parentheses ("Die Löwin (2024)") and is then
treated as its own film — if that year-suffixed name already exists encoded, the
existing duplicate modules decide replacement; otherwise it is encoded as new.

## Verified facts (this session)

- `media/media.go:168-172`: `OutName()` = `f.Name` if no Subtitle, else `Name + " - " + Subtitle`.
- `worker/worker.go` `traverseDir` (line ~621) and `traverseMemCache` (~615): match
  `filepath.Base(path) == file.OutName()+config.Instance().Local.Ext` — exact-name only.
- Job subtitle carries the year: "Spielfilm Deutschland 2024", "Fernsehfilm Deutschland 2007"
  (seen in DB jobs this session).
- `media/media.go` `File` struct already holds BOTH sources separately:
  - `MetadataLog []string` — `.txt` content (current EPG format: `Title=`, `Info=`, `Description=`)
  - `TunerLog []string` — `.log` content (timer/recording log)
  - `legacy bool` — true when no `.txt` exists (old format, `MetadataLog` empty)
  - `readLogs()` (media.go:352): reads `stem+'.txt'` into MetadataLog, `stem+'.log'`
    (or legacy `.mkv.log`/`.mpg.log`) into TunerLog; sets `legacy` when `.txt` missing.
- `movie_nfo_lib` (Python) has proven patterns; verified against real subtitles AND
  real log formats this session:
  - Current `.txt`: `Info=Spielfilm Deutschland/Estland/Lettland 2024` → year 2024
    (`extract_txt_meta_year_and_countries`, metadata.py:488).
  - Old `.log`-only: `Melodram Südafrika/2011` (meta line after time range) → year 2011
    (`_extract_log_metadata`, metadata.py:1131; meta regex
    `(\S+(?:\s+\S+)?)\s+(.+?)(?:/|\s)((?:19|20)\d{2})$`).
  - Old `.log` Timer Name: `Timer Name: Die Löwin - Spielfilm Deutschland/Estland/Lettland 2024`
    (`TIMER_NAME_META_PATTERN`, metadata.py:132) — most reliable source when present.
  - `_slice_log_lines_for_metadata` (metadata.py:908): stop parsing at
    `Removed Filler Data`/`Total Size`/`Monitoring Mode:`/first timestamp line.
  - `YEAR_FOUR_RE = (?:19|20)\d{2}(?!\s*er\b|\s*er-)` — 4-digit year, ignores decades.
  - `normalize_title` + `similarity` (text_utils.py) for name comparison.
- Go cannot import the Python lib; the regex patterns are platform-neutral and portable.
- No existing year extraction in avior-go (`grep year|Year` in media/ worker/ — none).

## Year source resolution (ported from extract_txt_metadata, metadata.py:1310)

For a given `media.File`, resolve the release year in this order (mirrors the
Python dispatcher):

1. `f.Subtitle` — job-provided subtitle (e.g. "Spielfilm Deutschland 2024");
   `metaYearRe` first, `yearFourRe` fallback.
2. `f.MetadataLog` (.txt) — `Info=` line (e.g. "Spielfilm Deutschland/Estland/Lettland 2024");
   same patterns.
3. `f.TunerLog` (.log) — `Timer Name:` line first (`TIMER_NAME_META_PATTERN`),
   else the meta line directly after the `HH:MM..HH:MM` time range
   (old-format `Melodram Südafrika/2011`). Parse range bounded by
   `Removed Filler Data`/`Total Size`/`Monitoring Mode:`/first `HH:MM:SS` line.

`f.legacy` already tells us `.txt` is absent — use TunerLog for legacy files.

### 1. New file `media/year.go` — year extraction + name normalization (ported)

```go
package media

import (
	"regexp"
	"strings"
)

var yearFourRe = regexp.MustCompile(`(?:19|20)\d{2}(?!\s*er\b|\s*er-)`)

// metaYearRe matches "Spielfilm Deutschland 2024" style fragments and captures the
// 4-digit year. Genre words optional; bare country+year also matches.
var metaYearRe = regexp.MustCompile(`(?i)(?:^|\s-\s|\s–\s|\s—\s)(?:(?:fernsehfilm|spielfilm|film|serie|dokumentation|komödie|komoedie|tragikomödie|tragikomoedie|drama|thriller|krimi|melodram|melodrama|animationsfilm|zeichentrickfilm)\s+)?(?:(?:[A-Za-zÄÖÜäöüß]{2,})(?:\s*(?:/|,|;|und)\s*(?:[A-Za-zÄÖÜäöüß]{2,}))*)\s+((?:19|20)\d{2})\b`)

// oldLogMetaRe matches the old-format meta line "Melodram Südafrika/2011"
// (genre + country + year at end, after the HH:MM..HH:MM time line).
var oldLogMetaRe = regexp.MustCompile(`(?i)^\s*(?:fernsehfilm|spielfilm|film|serie|dokumentation|komödie|komoedie|tragikomödie|tragikomoedie|drama|thriller|krimi|melodram|melodrama|animationsfilm|zeichentrickfilm)\s+.*?(?:/|\s)((?:19|20)\d{2})\s*$`)

// timerNameMetaRe matches "Timer Name: Die Löwin - Spielfilm Deutschland/Estland/Lettland 2024".
var timerNameMetaRe = regexp.MustCompile(`(?i)^\s*Timer\s+Name\s*:\s*.*?(?:(?:fernsehfilm|spielfilm|film|serie|dokumentation|komödie|komoedie|tragikomödie|tragikomoedie|drama|thriller|krimi|melodram|melodrama|animationsfilm|zeichentrickfilm)\s+)?.*?(?:19|20)\d{2}\b`)

// logSliceEndRe bounds the log parse range (mirrors _slice_log_lines_for_metadata).
var logSliceEndRe = regexp.MustCompile(`(?i)Removed Filler Data|Total Size|Monitoring Mode:|^\s*\d{1,2}:\d{2}:\d{2}`)

// ExtractYearFromFile resolves the release year for f using the .txt→.log
// fallback order (subtitle → MetadataLog → TunerLog). Returns "" if none.
func (f *File) ExtractYearFromFile() string

// ExtractYear returns the first 4-digit year found in s, or "".
func ExtractYear(s string) string

// NormalizeName mirrors movie_nfo_lib normalize_title: lowercase, strip non-word
// chars, collapse whitespace. Used for same-title comparison.
func NormalizeName(s string) string

// HasYearSuffix reports whether name already ends in " (YYYY)" — prevents
// double-suffixing when the year was already appended.
func HasYearSuffix(name string) bool
```

Edge cases: empty input → "" / false. No year → ExtractYear "". "1980er" not matched
(negative lookahead). Year-suffixed input → HasYearSuffix true. Legacy file without
`.txt` → TunerLog path only. `Timer Name:` present → wins (most reliable).

### 2. Config flag `YearAwareDupes` (default true; off = current behavior)

`config/config.go` `Local` struct:
```go
// YearAwareDupes: when true, same-title files with different release years are
// treated as separate films (year appended in parentheses). Default true.
YearAwareDupes bool `json:"YearAwareDupes"`
```
`InitWithDefaults`: `cfg.Local.YearAwareDupes = true`.
(No omitempty — same lesson as CacheLibScan: false must persist. With default true
this matters less, but consistency.)

### 3. Worker: resolve the effective output name before duplicate checks

In `worker.ProcessJob`, after `mediaFile.Update()` and before `checkForDuplicates`:

```go
if cfg.Local.YearAwareDupes && !media.HasYearSuffix(mediaFile.Name) {
    // The first encode derives the name from EPG data (Title= + Info= without
    // country/year): "Die Löwin". When that exact name already exists in the
    // library but the release years differ, rename to "Die Löwin (2024)" so the
    // two films are treated as separate.
    year := mediaFile.ExtractYearFromFile()
    if year != "" {
        if collision, dupeYear := findYearCollision(mediaFile, year); collision && dupeYear != year {
            _ = glg.Infof("year collision: appending (%s) to %s (existing has %s)", year, mediaFile.Name, dupeYear)
            mediaFile.Name = fmt.Sprintf("%s (%s)", mediaFile.Name, year)
        }
    }
}
```

`findYearCollision(file, year)` reuses the EXISTING exact-name duplicate scan
(the `checkForDuplicates` match: `filepath.Base(path) == file.OutName()+Ext`)
and, when it finds the exact same name, extracts the found duplicate's own year
(from its `.txt`/`.log` via `ExtractYearFromFile`, or from a ` (YYYY)` suffix in
its filename). Returns the found name's year.

**NOT a normalized title comparison across the whole library.** The collision is
found through the existing exact-name duplicate match; only the year of that one
found duplicate is compared against the new film's year.

When years differ, `mediaFile.Name = fmt.Sprintf("%s (%s)", mediaFile.Name, year)`
(OutName() then yields "... (2024)"). The existing duplicate flow then re-runs
`checkForDuplicates` with the new name: if `Die Löwin (2024).mkv` already exists,
modules decide replacement; if absent, it is encoded as new. `HasYearSuffix`
prevents re-suffixing.

Implementation: the exact-name scan currently runs once inside
`checkForDuplicates`. For the year flow, run the scan, and when a match exists,
extract the duplicate's year (media.File{Path: match}.ExtractYearFromFile or
parse the ` (YYYY)` suffix) and compare. If no exact-name match at all, no
collision — proceed with the original name.

### 4. Reuse existing duplicate decision path — no changes there

The module decision logic (LogMatch, Resolution, ErrorReplace, DuplicateLengthCheck)
in `worker.go` after `checkForDuplicates` stays untouched: once the name carries the
year suffix, the standard duplicate/replace flow applies.

## Critical files & anchors

- `media/media.go:168-172` — `OutName()`; year suffix is appended to `f.Name`, so
  OutName picks it up automatically.
- `worker/worker.go:596-650` — `checkForDuplicates`/`traverseDir`/`traverseMemCache`;
  the year-collision scan mirrors this traversal.
- `config/config.go` — `Local` struct + `InitWithDefaults` for the flag.
- `movie_nfo_lib/movie_metadata/metadata.py:125-165` — patterns to port (verified).

## Verification

Working dir: repo root `C:/repos/avior-go`.

1. `go build ./... && go vet ./...`.
2. Unit check (no DB): scratch `go run` calling `ExtractYear` on
   "Spielfilm Deutschland 2024" → "2024"; "Ruhe in Frieden" → ""; "1980er-Jahre" → "";
   `NormalizeName("Die Löwin - Spielfilm")` == normalized compare input;
   `HasYearSuffix("Die Löwin (2024)")` → true, `HasYearSuffix("Die Löwin")` → false.
3. Logic check of the suffix flow with two synthetic media.File:
   file A OutName "Die Löwin" (year 2020 in subtitle), file B "Die Löwin" (year 2024) —
   collision detected, B.Name becomes "Die Löwin (2024)"; re-run: no double suffix.
4. End-to-end on Unraid (feat/dual-os, instance 1): add a job whose title exists in
   the library with a different year; expect log "year collision: appending (2024)"
   and output file "Die Löwin (2024).mkv". Re-push the same job: duplicate modules
   decide replacement (existing behavior).
5. Regression: `YearAwareDupes=false` → identical naming as before (exact match only).

## Assumptions & contingencies

- Year source is the Subtitle (verified format). If a subtitle lacks a year but the
  filename contains one, `ExtractYear` still finds it via `yearFourRe` fallback.
- Name comparison basis is the title (mediaFile.Name), NOT the full OutName with
  subtitle — the subtitle itself carries the year and would otherwise defeat the
  collision. If reality shows collisions should compare OutName-with-subtitle-minus-year,
  adjust the NormalizeName comparison input accordingly (single spot).
- The suffix format is fixed " (YYYY)" per the user's example ("Die Löwin (2024)").
