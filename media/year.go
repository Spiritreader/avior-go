package media

import (
	"regexp"
	"strings"
)

// Ported from movie_nfo_lib (movie_metadata/metadata.py) — see
// .cortex/plans/year-aware-dupes-plan.md. Patterns and extraction order mirror
// extract_txt_metadata / _extract_log_metadata / extract_txt_meta_year_and_countries.
// Only the release-year result is surfaced here (avior-go's duplicate detection
// needs the year, not countries/genres). RE2 notes: no lookahead, \w is ASCII
// (use \p{L}/\p{N} for Unicode).

const countryHintPattern = `usa|us|u\.?s\.?|uk|u\.?k\.?|gb|gbr|can|cdn|uae|ksa|prc|rok|d-a-ch|dach|ddr|brd|de|at|ch|fr|f|it|es|d|a|cz|cssr|` +
	`deutschland|österreich|oesterreich|schweiz|frankreich|italien|spanien|` +
	`großbritannien|grossbritannien|belgien|niederlande|luxemburg|` +
	`vereinigtes\s+königreich|vereinigtes\s+königreich(?:\s+von\s+amerika)?|` +
	`vereinigte\s+staaten(?:\s+von\s+amerika)?|amerika|` +
	`portugal|griechenland|graechenland|irland|island|` +
	`dänemark|daenemark|schweden|norwegen|finnland|` +
	`polen|tschechien|tschechoslowakei|ungarn|rumänien|rumaenien|bulgarien|slowakei|slowenien|` +
	`kroatien|serbien|ukraine|ukraina|litauen|lettland|estland|zypern|malta|` +
	`kosovo|bosnien|montenegro|mazedonien|nordmazedonien|` +
	`brasilien|argentinien|kolumbien|peru|chile|venezuela|uruguay|mexiko|` +
	`kuba|cuba|` +
	`japan|china|südkorea|suedkorea|nordkorea|hongkong|taiwan|thailand|` +
	`vietnam|philippinen|indonesien|malaysia|singapur|kambodscha|cambodia|` +
	`indien|pakistan|bangladesch|` +
	`israel|iran|irak|saudi-arabien|saudiarabien|libanon|türkei|tuerkei|` +
	`südafrika|suedafrika|ägypten|aegypten|marokko|algerien|tunesien|` +
	`nigeria|kenia|` +
	`australien|neuseeland|kanada|russland|sowjetunion|udssr|ussr|weißrussland|` +
	`weissrussland|belarus|` +
	`united\s+states(?:\s+of\s+america)?|united\s+kingdom|` +
	`canada|mexico|brazil|argentina|` +
	`germany|austria|switzerland|france|italy|spain|` +
	`netherlands|belgium|portugal|greece|ireland|iceland|` +
	`denmark|sweden|norway|finland|` +
	`poland|czech(?:\s+republic)?|czechoslovakia|hungary|romania|bulgaria|slovakia|slovenia|` +
	`croatia|serbia|ukraine|lithuania|latvia|estonia|cyprus|malta|` +
	`kosovo|bosnia|montenegro|` +
	`japan|china|korea|thailand|vietnam|philippines|indonesia|malaysia|` +
	`singapore|india|pakistan|bangladesh|sri\s+lanka|` +
	`israel|iran|turkey|lebanon|` +
	`south\s+africa|egypt|morocco|algeria|tunisia|nigeria|kenya|` +
	`mauritania|mauretanien|` +
	`cote\s+d(?:[\'\"]?\s?ivoire)?|ivory\s+coast|ci|` +
	`australia|new\s+zealand|` +
	`russia|soviet\s+union|ussr|udssr|belarus|north\s+macedonia|macedonia`

const genreWords = `(?:fernsehfilm|spielfilm|film|serie|dokumentation|komödie|komoedie|tragikomödie|tragikomoedie|drama|thriller|krimi|melodram|melodrama|animationsfilm|zeichentrickfilm)`

// countryListRe matches a country list: one or more COUNTRY_HINT alternatives
// separated by / , ; und. No capture group around the list (RE2 chokes on a
// capturing alternation wrapped in a repeat with a following separator).
var countryListRe = regexp.MustCompile(`(?i)(?:` + countryHintPattern + `)(?:\s*(?:/|,|;|und)\s*(?:` + countryHintPattern + `))*`)

// txtMetaRe mirrors TXT_META_PATTERN: optional genre, then country list, then year.
// Group 1 = year (country list is a non-capturing alternation).
var txtMetaRe = regexp.MustCompile(`(?i)^\s*(?:\([^)]*\)\s*)*(?:[-–—:,]\s*)*(?:` + genreWords + `\s*,?\s*)?(?:` + countryHintPattern + `)(?:\s*(?:/|,|;|und)\s*(?:` + countryHintPattern + `))*\s*(?:,|/|\s)\s*((?:19|20)\d{2})\b`)

// logMetaLineRe mirrors LOG_META_LINE_PATTERN: genre (optional) + countries + year.
// Group 1 = year.
var logMetaLineRe = regexp.MustCompile(`(?i)(?:^|\s-\s|\s–\s|\s—\s)(?:` + genreWords + `\s+)?(?:` + countryHintPattern + `)(?:\s*(?:/|,|;|und)\s*(?:` + countryHintPattern + `))*\s+((?:19|20)\d{2})\b`)

// logMetaLineStrictRe mirrors LOG_META_LINE_STRICT_PATTERN: full-line match.
// Group 1 = year.
var logMetaLineStrictRe = regexp.MustCompile(`(?i)^\s*(?:` + genreWords + `\s+)?(?:` + countryHintPattern + `)(?:\s*(?:/|,|;|und)\s*(?:` + countryHintPattern + `))*\s+((?:19|20)\d{2})\s*$`)

// timerNameMetaRe mirrors TIMER_NAME_META_PATTERN: "Timer Name: ... - Spielfilm Land 2024".
// Group 1 = year.
var timerNameMetaRe = regexp.MustCompile(`(?i)(?:^|\s-\s|\s–\s|\s—\s)(?:` + genreWords + `\s+)?(?:` + countryHintPattern + `)(?:\s*/\s*(?:` + countryHintPattern + `))*\s+((?:19|20)\d{2})\b`)

// type2GenreCountryYearRe mirrors TYPE2_GENRE_COUNTRY_YEAR_PATTERN.
// Group 1 = year.
var type2GenreCountryYearRe = regexp.MustCompile(`(?i)^\s*(?:[A-Za-zÄÖÜäöüß][\wÄÖÜäöüß-]*(?:\s+[A-Za-zÄÖÜäöüß][\wÄÖÜäöüß-]*)?\s+)?(?:` + countryHintPattern + `)(?:\s*(?:/|,|;|und)\s*(?:` + countryHintPattern + `))*\s*(?:,|/)\s*((?:19|20)\d{2})\s*$`)

// type2CountryYearFallbackRe mirrors TYPE2_COUNTRY_YEAR_FALLBACK_PATTERN.
// Group 1 = year.
var type2CountryYearFallbackRe = regexp.MustCompile(`(?i)^\s*(?:` + countryHintPattern + `)(?:\s*/\s*(?:` + countryHintPattern + `))*\s*(?:,|/)\s*((?:19|20)\d{2})\s*$`)

// oldLogMetaRe matches the old-format meta line "Melodram Südafrika/2011"
// (genre + country + year at end). Group 1 = year.
var oldLogMetaRe = regexp.MustCompile(`(?i)^\s*(?:` + genreWords + `)\s+(?:` + countryHintPattern + `)(?:\s*(?:/|,|;|und)\s*(?:` + countryHintPattern + `))*\s*((?:19|20)\d{2})\s*$`)

// yearFourRe mirrors YEAR_FOUR_RE: 4-digit year. Decade forms like "1980er"
// are rejected manually (RE2 has no lookahead). Group 1 = year.
var yearFourRe = regexp.MustCompile(`((?:19|20)\d{2})`)

// logSliceEndRe mirrors _slice_log_lines_for_metadata stop markers.
var logSliceEndRe = regexp.MustCompile(`(?i)Removed Filler Data|Total Size|Monitoring Mode:|^\s*\d{1,2}:\d{2}:\d{2}`)

// yearSuffixRe matches a trailing " (YYYY)" in a name.
var yearSuffixRe = regexp.MustCompile(`\s+\(((?:19|20)\d{2})\)\s*$`)

// timeRangeRe matches the old-format "20:15..21:45" airtime line.
var timeRangeRe = regexp.MustCompile(`^\d{1,2}:\d{2}\.\.\d{1,2}:\d{2}$`)

// ExtractYearFromFile resolves the release year for f following the library's
// extract_txt_metadata order: subtitle → .txt (Info=/Description=) → .log
// (Timer Name → meta line), with .log as supplement when .txt lacks a year.
// Returns "" when no year can be determined.
func (f *File) ExtractYearFromFile() string {
	// 1. Job subtitle (avior-dis provides it): "Spielfilm Deutschland 2024".
	if y := ExtractYear(f.Subtitle); y != "" {
		return y
	}
	// 2. .txt metadata (current format). Info= first, then Description=-derived
	//    meta line. Recording dates (Created=/Date=) are NEVER release years.
	infoLine := ""
	for _, l := range f.MetadataLog {
		if strings.HasPrefix(l, "Info=") {
			infoLine = strings.TrimPrefix(l, "Info=")
			break
		}
	}
	if infoLine != "" {
		if y := yearFromMetaCandidate(infoLine); y != "" {
			return y
		}
	}
	for _, l := range f.MetadataLog {
		if strings.HasPrefix(l, "Description=") {
			desc := strings.TrimPrefix(l, "Description=")
			if seg := firstMetaLikeSegment(desc); seg != "" {
				if y := yearFromMetaCandidate(seg); y != "" {
					return y
				}
			}
		}
	}
	// 3. .log: Timer Name first (most reliable), then the meta line.
	logYear := extractYearFromLog(f.TunerLog)
	if logYear != "" {
		return logYear
	}
	return ""
}

// ExtractYear returns the first 4-digit year found in s, or "".
// Decade forms like "1980er" are ignored.
func ExtractYear(s string) string {
	if s == "" {
		return ""
	}
	if y := yearFromMetaCandidate(s); y != "" {
		return y
	}
	// Fallback: any 4-digit year not followed by "er" (decade form).
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

// yearFromMetaCandidate extracts the release year from a metadata fragment using
// the library's TXT_META_PATTERN / TYPE2 patterns in order. Group 1 = year in all
// patterns (country list is a non-capturing alternation).
func yearFromMetaCandidate(candidate string) string {
	if candidate == "" {
		return ""
	}
	if m := txtMetaRe.FindStringSubmatch(candidate); len(m) == 2 {
		return m[1]
	}
	if m := type2GenreCountryYearRe.FindStringSubmatch(candidate); len(m) == 2 {
		return m[1]
	}
	if m := type2CountryYearFallbackRe.FindStringSubmatch(candidate); len(m) == 2 {
		return m[1]
	}
	if m := logMetaLineRe.FindStringSubmatch(candidate); len(m) == 2 {
		return m[1]
	}
	if m := logMetaLineStrictRe.FindStringSubmatch(candidate); len(m) == 2 {
		return m[1]
	}
	return ""
}

// firstMetaLikeSegment picks the best "Land Jahr"-style segment from a
// pipe-separated Description= line (mirrors extract_txt_meta_year_and_countries
// description handling, simplified to the year-bearing segment).
func firstMetaLikeSegment(desc string) string {
	if !strings.Contains(desc, "|") {
		if y := ExtractYear(desc); y != "" {
			return desc
		}
		return ""
	}
	parts := []string{}
	for _, p := range strings.Split(desc, "|") {
		if t := strings.TrimSpace(p); t != "" {
			parts = append(parts, t)
		}
	}
	for _, seg := range parts {
		if y := ExtractYear(seg); y != "" {
			if yearFromMetaCandidate(seg) != "" {
				return seg
			}
		}
	}
	for _, seg := range parts {
		if y := ExtractYear(seg); y != "" {
			return seg
		}
	}
	return ""
}

// extractYearFromLog mirrors _extract_log_metadata's year sources:
// Timer Name first, then the meta line after the time range.
func extractYearFromLog(tunerLog []string) string {
	metaLines := sliceLogMetadata(tunerLog)
	// Timer Name is the most reliable source.
	for _, line := range metaLines {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "timer name") {
			v := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(line), "Timer Name"), "timer name"))
			v = strings.TrimPrefix(v, ":")
			v = strings.TrimSpace(v)
			if y := yearFromMetaCandidate(v); y != "" {
				return y
			}
			if y := ExtractYear(v); y != "" {
				return y
			}
		}
	}
	// Old format: meta line directly after the HH:MM..HH:MM time line.
	for i, line := range metaLines {
		if timeRangeRe.MatchString(strings.TrimSpace(line)) && i+1 < len(metaLines) {
			meta := strings.TrimSpace(metaLines[i+1])
			if m := oldLogMetaRe.FindStringSubmatch(meta); len(m) == 2 {
				return m[1]
			}
			if y := yearFromMetaCandidate(meta); y != "" {
				return y
			}
		}
	}
	// General meta line anywhere in the metadata section.
	for _, line := range metaLines {
		if y := yearFromMetaCandidate(line); y != "" {
			return y
		}
	}
	return ""
}

// NormalizeName mirrors movie_nfo_lib normalize_title: lowercase, strip non-word
// chars, collapse whitespace. \p{L}/\p{N} for Unicode (Go \w is ASCII-only).
func NormalizeName(s string) string {
	s = strings.ToLower(s)
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
