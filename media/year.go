package media

import (
	"regexp"
	"strings"

	"github.com/kpango/glg"
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
// Returns "" when no year can be determined. Each source is logged so the
// extraction path is transparent (important for diagnosing missing years).
func (f *File) ExtractYearFromFile() string {
	// 1. Job subtitle (avior-dis provides it): "Spielfilm Deutschland 2024".
	if y := ExtractYear(f.Subtitle); y != "" {
		glg.Infof("year extraction: subtitle -> %s", y)
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
			glg.Infof("year extraction: txt Info= -> %s", y)
			return y
		}
		glg.Infof("year extraction: txt Info= line found but no year in %q", infoLine)
	}
	for _, l := range f.MetadataLog {
		if strings.HasPrefix(l, "Description=") {
			desc := strings.TrimPrefix(l, "Description=")
			if seg := firstMetaLikeSegment(desc); seg != "" {
				if y := yearFromMetaCandidate(seg); y != "" {
					glg.Infof("year extraction: txt Description= -> %s", y)
					return y
				}
			}
		}
	}
	if len(f.MetadataLog) > 0 {
		glg.Infof("year extraction: no year in .txt metadata (%d lines)", len(f.MetadataLog))
	} else {
		glg.Infof("year extraction: no .txt metadata present")
	}
	// 3. .log: Timer Name first (most reliable), then the meta line.
	logYear := extractYearFromLog(f.TunerLog)
	if logYear != "" {
		glg.Infof("year extraction: log -> %s", logYear)
		return logYear
	}
	glg.Infof("year extraction: no year found in subtitle/.txt/.log for %s", f.Path)
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

// nonCountryWords mirrors the library's _NON_COUNTRY_WORDS: narrative words that
// must not be mistaken for a country before a year ("Jahr 2022", "im 2024").
var nonCountryWords = map[string]bool{
	"jahr": true, "year": true, "im": true, "ein": true, "eine": true,
	"der": true, "die": true, "das": true, "und": true, "mit": true,
	"von": true, "für": true, "aus": true, "the": true,
	"in": true, "at": true, "nach": true, "auf": true, "bei": true,
	"this": true, "that": true, "all": true,
}

// narrativeMarkerRe mirrors NARRATIVE_MARKER: a non-country word directly
// followed by a digit → the "year" is actually narrative text, not a year.
var narrativeMarkerRe = regexp.MustCompile(`(?i)^\s*(?:jahr|year|im|ein|eine|der|die|das|und|mit|von|für|fuer|aus|the|in|at|nach|auf|bei|this|that|all)\s+\d`)

// preprocessCandidate mirrors the library's candidate cleaning before matching:
// FSK strip, <> strip, leading/trailing parens, " - subtitle" truncation,
// "Min." suffix strip, comma→space, whitespace collapse, duplicate words.
func preprocessCandidate(s string) string {
	s = strings.TrimSpace(s)
	fsKRe := regexp.MustCompile(`(?i)^\s*FSK\s*\d+\s*`)
	s = fsKRe.ReplaceAllString(s, "")
	s = regexp.MustCompile(`^\s*<\s*`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s*>\s*$`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`^\s*\([^)]*\)\s*`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s+[-–—]\s+.*$`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s*\([^)]*\)\s*$`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)deutsche\s+demokratische\s+republik`).ReplaceAllString(s, "DDR")
	s = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`(?i)\bFSK\s*\d+\b`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i),?\s*[A-Za-zÄÖÜäöü0-9 &]{2,30}\s*\d{1,3}\s*Min\.?$`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	// Collapse duplicate words: "Deutschland Deutschland 2024" -> "Deutschland 2024"
	words := strings.Fields(s)
	out := make([]string, 0, len(words))
	for i, w := range words {
		if i > 0 && strings.EqualFold(w, words[i-1]) {
			continue
		}
		out = append(out, w)
	}
	return strings.Join(out, " ")
}

// yearFromMetaCandidate is the full port of the library's _extract_from_candidate
// cascade (metadata.py:589-733). Group 1 = year in all patterns. Returns "" when
// no year can be determined.
func yearFromMetaCandidate(candidate string) string {
	if candidate == "" {
		return ""
	}
	s := strings.TrimSpace(candidate)
	s = preprocessCandidate(s)

	// Determine the first year position to bound the match window.
	mYearFirst := yearFourRe.FindStringIndex(s)

	// Truncate candidate from the first genre/country word up to the first year
	// (mirrors start_match logic).
	startIdx := -1
	if m := regexp.MustCompile(`(?i)\b(?:` + genreWords + `|` + countryHintPattern + `)\b`).FindStringIndex(s); m != nil {
		startIdx = m[0]
	}
	candidateForMatch := s
	if startIdx >= 0 {
		if mYearFirst != nil {
			candidateForMatch = s[startIdx:mYearFirst[1]]
		} else {
			candidateForMatch = s[startIdx:]
		}
	} else if mYearFirst != nil {
		candidateForMatch = s[:mYearFirst[1]]
	}

	// 1. TXT_META_PATTERN
	if m := txtMetaRe.FindStringSubmatch(candidateForMatch); len(m) == 2 {
		return m[1]
	}
	// 2. TYPE2_GENRE_COUNTRY_YEAR_PATTERN
	if m := type2GenreCountryYearRe.FindStringSubmatch(candidateForMatch); len(m) == 2 {
		return m[1]
	}
	// 3. loose_re: <text> <year> at end, gated by NARRATIVE_MARKER
	candidateNorm := preprocessCandidate(s)
	candidateNorm = regexp.MustCompile(`\s+[-–—]\s+.*$`).ReplaceAllString(candidateNorm, "")
	candidateNorm = regexp.MustCompile(`\s*\([^)]*\)\s*$`).ReplaceAllString(candidateNorm, "")
	if m := looseYearRe.FindStringSubmatch(candidateNorm); len(m) == 2 {
		rawCountries := m[1]
		if narrativeMarkerRe.MatchString(rawCountries) {
			return ""
		}
		// Single country starting with a genre word: "Spielfilm Deutschland"
		parts := splitCountryParts(rawCountries)
		if len(parts) == 1 {
			for _, sg := range []string{"zeichentrick", "spielfilm", "film", "serie", "dokumentation", "komödie"} {
				if strings.HasPrefix(parts[0], sg+" ") {
					return m[2]
				}
			}
		}
		if len(parts) > 0 {
			return m[2]
		}
	}
	// 4. short_genre_re: "Genre Land Jahr"
	if m := shortGenreYearRe.FindStringSubmatch(candidate); len(m) == 4 {
		g := strings.ToLower(strings.TrimSpace(m[1]))
		if shortGenres[g] {
			return m[3]
		}
	}
	// 5. permissive_re: <text> <year> anywhere, gated by NARRATIVE_MARKER
	if m := permissiveYearRe.FindStringSubmatch(s); len(m) == 3 {
		combined := m[1] + " " + m[2]
		if narrativeMarkerRe.MatchString(combined) {
			return ""
		}
		parts := splitCountryParts(m[1])
		if len(parts) > 0 {
			return m[2]
		}
	}
	return ""
}

// looseYearRe mirrors the library's loose_re: <non-digit text> <year> at end.
var looseYearRe = regexp.MustCompile(`(?i)([^\d\n\r]+?)\s+((?:19|20)\d{2})\s*$`)

// shortGenreYearRe mirrors short_genre_re: Genre + Countries + Year.
var shortGenreYearRe = regexp.MustCompile(`(?i)^\s*([A-Za-zÄÖÜäöüß\-]{3,20})\s+(.+?)\s+((?:19|20)\d{2})\s*$`)

// permissiveYearRe mirrors m_permissive: <non-digit text> <year>.
var permissiveYearRe = regexp.MustCompile(`(?i)([^\d\n\r|]+?)\s+((?:19|20)\d{2})\b`)

var shortGenres = map[string]bool{
	"fernsehfilm": true, "spielfilm": true, "film": true, "serie": true,
	"dokumentation": true, "komödie": true, "komoedie": true,
	"tragikomödie": true, "tragikomoedie": true, "drama": true,
	"thriller": true, "krimi": true, "melodram": true, "animationsfilm": true,
	"zeichentrick": true, "zeichentrickfilm": true,
}

// splitCountryParts splits a country string by / , ; und (mirrors the library).
func splitCountryParts(s string) []string {
	re := regexp.MustCompile(`(?i)\s*(?:/|,|;|\band\b|\bund\b|und)\s*`)
	parts := re.Split(s, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// firstMetaLikeSegment picks the best "Land Jahr"-style segment from a
// pipe-separated Description= line (mirrors extract_txt_meta_year_and_countries
// description handling, simplified to the year-bearing segment).
// firstMetaLikeSegment mirrors the library's Description=-segment selection with
// scoring (narrative_penalty, meta_like, short_segment): picks the best
// "Land Jahr"-style segment from a pipe-separated Description= line.
func firstMetaLikeSegment(desc string) string {
	parts := []string{}
	for _, p := range strings.Split(desc, "|") {
		if t := strings.TrimSpace(p); t != "" {
			parts = append(parts, t)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	// Single segment: use it if it carries a year.
	if len(parts) == 1 {
		if ExtractYear(parts[0]) != "" {
			return parts[0]
		}
		return ""
	}
	// Score each year-bearing segment like the library: prefer meta-like
	// (genre+country+year), short segments, low narrative penalty.
	type scored struct {
		penalty int
		meta    int
		tail    int
		words   int
		seg     string
	}
	var candidates []scored
	for _, seg := range parts {
		mYear := yearFourRe.FindStringIndex(seg)
		if mYear == nil {
			continue
		}
		tailLen := len(seg) - mYear[1]
		narrativePenalty := 0
		if regexp.MustCompile(`\d{4}\s*:\s*[A-Za-z]`).MatchString(seg) {
			narrativePenalty = 1
		}
		metaLike := false
		if yearFromMetaCandidate(seg) != "" {
			metaLike = true
		}
		wordCount := len(strings.Fields(seg))
		shortSeg := wordCount <= 10 && tailLen < 40
		if !metaLike && !shortSeg {
			continue
		}
		metaScore := 0
		if !metaLike {
			metaScore = 1
		}
		candidates = append(candidates, scored{narrativePenalty, metaScore, tailLen, wordCount, seg})
	}
	if len(candidates) > 0 {
		// Sort: penalty asc, meta asc, tail asc (mirrors the library's sort key).
		best := candidates[0]
		for _, c := range candidates[1:] {
			if c.penalty < best.penalty ||
				(c.penalty == best.penalty && c.meta < best.meta) ||
				(c.penalty == best.penalty && c.meta == best.meta && c.tail < best.tail) {
				best = c
			}
		}
		return best.seg
	}
	// Fallback: last segment carrying a year and letters (mirrors library's last
	// candidate fallback).
	for i := len(parts) - 1; i >= 0; i-- {
		seg := parts[i]
		if yearFourRe.MatchString(seg) && regexp.MustCompile(`[A-Za-zÄÖÜäöüß]`).MatchString(seg) {
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

// IsProbablyEpisodeFilename mirrors is_probably_episode_filename: True for series
// episode filenames (SxxExx, (N_N), NxNN, (Staffel N, Folge N), (123)-in-parens,
// 4-digit in parens NOT at end, Folge/Episode/Kapitel N). A year-suffixed film
// like "Die Löwin (2024)" is NOT an episode (year at end is excluded).
func IsProbablyEpisodeFilename(name string) bool {
	stem := strings.TrimSuffix(name, filepathExt(name))
	for _, re := range episodePatterns {
		if re.MatchString(stem) {
			return true
		}
	}
	// 4-digit in parens NOT at stem end -> episode (e.g. (1188) mid-name).
	// (2011) at the very end is a year, not an episode — handled by the position
	// check below (RE2 has no lookahead).
	if m := episodeFourDigitInParensRe.FindStringIndex(stem); m != nil {
		if m[1] < len(stem) && !regexp.MustCompile(`^\s*$`).MatchString(stem[m[1]:]) {
			return true
		}
	}
	return false
}

// filepathExt returns the extension including the dot, or "".
func filepathExt(name string) string {
	idx := strings.LastIndexAny(name, "./\\")
	if idx < 0 || name[idx] != '.' {
		return ""
	}
	return name[idx:]
}

var episodePatterns = []*regexp.Regexp{
	// SxxExx / S01_E01
	regexp.MustCompile(`(?i)\bS\d{1,2}E\d{1,2}\b|\bS\d{1,2}_E\d{1,2}\b`),
	// (N_N) season_episode in parens
	regexp.MustCompile(`\(\d{1,2}_\d{1,2}\)`),
	// NxNN bare notation (2x07)
	regexp.MustCompile(`\b\d{1,2}x\d{1,2}\b`),
	// (Staffel N, Folge N) in parens
	regexp.MustCompile(`(?i)[Ss]taffel\s*\d+[^)]*[Ff]olge\s*\d+`),
	// 1-3 digit in parens -> episode, never year
	regexp.MustCompile(`\(\d{1,3}\)`),
	// Folge 5 / Episode 3 / Kapitel 16
	regexp.MustCompile(`(?i)\b(?:Folge|Episode|Ep\.?|Kapitel)\s+\d+`),
}

// episodeFourDigitInParensRe matches "(1188)" NOT at the stem end (episode
// number). The library uses a lookahead (?!\s*$); RE2 has none, so the position
// check is done manually in IsProbablyEpisodeFilename.
var episodeFourDigitInParensRe = regexp.MustCompile(`\(\d{4}\)`)
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
