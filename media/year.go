package media

import (
	"regexp"
	"strings"
)

// Ported from movie_nfo_lib (movie_metadata/metadata.py) — see
// .cortex/plans/year-aware-dupes-plan.md. Patterns are platform-neutral.

var yearFourRe = regexp.MustCompile(`((?:19|20)\d{2})`)

// metaYearRe matches "Spielfilm Deutschland 2024" style fragments and captures the
// 4-digit year. Genre words are optional; a bare country+year also matches.
var metaYearRe = regexp.MustCompile(`(?i)(?:^|\s-\s|\s–\s|\s—\s)(?:(?:fernsehfilm|spielfilm|film|serie|dokumentation|komödie|komoedie|tragikomödie|tragikomoedie|drama|thriller|krimi|melodram|melodrama|animationsfilm|zeichentrickfilm)\s+)?(?:(?:[A-Za-zÄÖÜäöüß]{2,})(?:\s*(?:/|,|;|und)\s*(?:[A-Za-zÄÖÜäöüß]{2,}))*)\s+((?:19|20)\d{2})\b`)

// oldLogMetaRe matches the old-format meta line "Melodram Südafrika/2011"
// (genre + country + year at end, after the HH:MM..HH:MM time line).
var oldLogMetaRe = regexp.MustCompile(`(?i)^\s*(?:fernsehfilm|spielfilm|film|serie|dokumentation|komödie|komoedie|tragikomödie|tragikomoedie|drama|thriller|krimi|melodram|melodrama|animationsfilm|zeichentrickfilm)\s+.*?(?:/|\s)((?:19|20)\d{2})\s*$`)

// timerNameMetaRe matches "Timer Name: Die Löwin - Spielfilm Deutschland/Estland/Lettland 2024".
var timerNameMetaRe = regexp.MustCompile(`(?i)^\s*Timer\s+Name\s*:\s*.*?\b((?:19|20)\d{2})\b`)

// logSliceEndRe bounds the log parse range (mirrors _slice_log_lines_for_metadata):
// stop at "Removed Filler Data", "Total Size", "Monitoring Mode:" or the first
// timestamp line — everything after is recorder noise, not metadata.
var logSliceEndRe = regexp.MustCompile(`(?i)Removed Filler Data|Total Size|Monitoring Mode:|^\s*\d{1,2}:\d{2}:\d{2}`)

// yearSuffixRe matches a trailing " (YYYY)" in a name.
var yearSuffixRe = regexp.MustCompile(`\s+\(((?:19|20)\d{2})\)\s*$`)

// ExtractYearFromFile resolves the release year for f using the .txt→.log
// fallback order: subtitle → MetadataLog (.txt) → TunerLog (.log). Returns ""
// when no year can be determined.
func (f *File) ExtractYearFromFile() string {
	// 1. Job subtitle, e.g. "Spielfilm Deutschland 2024".
	if y := ExtractYear(f.Subtitle); y != "" {
		return y
	}
	// 2. .txt metadata (current format): Info= / Title= lines carry the year.
	for _, line := range f.MetadataLog {
		if y := ExtractYear(line); y != "" {
			return y
		}
	}
	// 3. .log (legacy format, no .txt): Timer Name first (most reliable), then
	//    the old-format meta line "Melodram Südafrika/2011".
	metaLines := sliceLogMetadata(f.TunerLog)
	for _, line := range metaLines {
		if timerNameMetaRe.MatchString(line) {
			if m := timerNameMetaRe.FindStringSubmatch(line); len(m) == 2 {
				return m[1]
			}
		}
	}
	for _, line := range metaLines {
		if m := oldLogMetaRe.FindStringSubmatch(line); len(m) == 2 {
			return m[1]
		}
	}
	return ""
}

// ExtractYear returns the first 4-digit year found in s, or "".
// "1980er"/"1980er-Jahre" are ignored (decade forms, not years).
func ExtractYear(s string) string {
	if s == "" {
		return ""
	}
	if m := metaYearRe.FindStringSubmatch(s); len(m) == 2 {
		return m[1]
	}
	// Fallback: any 4-digit year, but reject decade forms like "1980er"/"1980er-Jahre"
	// (RE2 has no lookahead, so check the chars after each match manually).
	for _, m := range yearFourRe.FindAllStringSubmatchIndex(s, -1) {
		if len(m) != 4 {
			continue
		}
		rest := s[m[1]:]
		if strings.HasPrefix(rest, "er") {
			continue
		}
		return s[m[0]:m[1]]
	}
	return ""
}

// NormalizeName mirrors movie_nfo_lib normalize_title: lowercase, strip non-word
// chars, collapse whitespace. Used for title comparison.
func NormalizeName(s string) string {
	s = strings.ToLower(s)
	// \p{L}/\p{N} (Unicode letters/digits) instead of \w — Go's \w is ASCII-only
	// and would strip umlauts ("Löwin" -> "lwin").
	s = regexp.MustCompile(`[^\p{L}\p{N}\s]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// HasYearSuffix reports whether name already ends in " (YYYY)" — prevents
// double-suffixing when the year was already appended.
func HasYearSuffix(name string) bool {
	return yearSuffixRe.MatchString(name)
}

// YearFromSuffix returns the year from a trailing " (YYYY)" suffix, or "".
func YearFromSuffix(name string) string {
	m := yearSuffixRe.FindStringSubmatch(name)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

// sliceLogMetadata bounds the log lines to the metadata section, mirroring
// movie_nfo_lib _slice_log_lines_for_metadata.
func sliceLogMetadata(lines []string) []string {
	end := len(lines)
	for i, line := range lines {
		if logSliceEndRe.MatchString(line) {
			end = i
			break
		}
	}
	if end > len(lines) {
		end = len(lines)
	}
	return lines[:end]
}
