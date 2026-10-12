package media

import "testing"

func TestExtractYear(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Spielfilm Deutschland 2024", "2024"},
		{"Fernsehfilm Deutschland 2007", "2007"},
		{"Spielfilm USA 2020", "2020"},
		{"Melodram Südafrika/2011", "2011"},
		{"Spielfilm Deutschland/Estland/Lettland 2024", "2024"},
		{"Die Löwin - Spielfilm Deutschland/Estland/Lettland 2024", "2024"},
		{"Dokumentarfilm, Deutschland 2023", "2023"},
		{"Spielfilm Deutschland 2017, ZDF", "2017"},
		{"Ruhe in Frieden", ""},
		{"1980er-Jahre", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ExtractYear(tt.in); got != tt.want {
			t.Errorf("ExtractYear(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestHasYearSuffix(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"Die Löwin (2024)", true},
		{"Die Löwin", false},
		{"Die Löwin (2024) Teil 2", false}, // year not at end
		{"", false},
	}
	for _, tt := range tests {
		if got := HasYearSuffix(tt.in); got != tt.want {
			t.Errorf("HasYearSuffix(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestYearFromSuffix(t *testing.T) {
	if got := YearFromSuffix("Die Löwin (2024)"); got != "2024" {
		t.Errorf("YearFromSuffix = %q, want 2024", got)
	}
	if got := YearFromSuffix("Die Löwin"); got != "" {
		t.Errorf("YearFromSuffix(no suffix) = %q, want empty", got)
	}
}

func TestNormalizeName(t *testing.T) {
	a := NormalizeName("Die Löwin - Spielfilm")
	b := NormalizeName("Die Löwin Spielfilm")
	if a != b {
		t.Errorf("NormalizeName mismatch: %q vs %q", a, b)
	}
	if NormalizeName("Die Löwin") != "die löwin" {
		t.Errorf("NormalizeName lower/trim wrong: %q", NormalizeName("Die Löwin"))
	}
}

func TestExtractYearFromFileLegacyLog(t *testing.T) {
	// Old format: no .txt, log carries "Melodram Südafrika/2011" after the time line.
	f := File{
		Name: "Die Löwin",
		TunerLog: []string{
			"ZDF HD 02.01.2012",
			"Die Löwin",
			"20:15..21:45",
			"Melodram Südafrika/2011",
			"20:10:02 Start",
			"Total Size 9078,8 MB",
		},
	}
	if y := f.ExtractYearFromFile(); y != "2011" {
		t.Errorf("ExtractYearFromFile(legacy log) = %q, want 2011", y)
	}
}

func TestExtractYearFromFileCurrentTxt(t *testing.T) {
	// Current format: .txt metadata with Info= line.
	f := File{
		Name:        "Die Löwin",
		MetadataLog: []string{"Info=Spielfilm Deutschland/Estland/Lettland 2024"},
	}
	if y := f.ExtractYearFromFile(); y != "2024" {
		t.Errorf("ExtractYearFromFile(txt) = %q, want 2024", y)
	}
}

func TestExtractYearFromFileTxtIgnoresRecordingDate(t *testing.T) {
	// The .txt carries the RECORDING date (Created=03.08.2026) BEFORE the Info=
	// line. The recording year must NOT be used — only the Info= release year.
	f := File{
		Name: "Die Löwin",
		MetadataLog: []string{
			"[Media]",
			"Created=03.08.2026 23:00:06",
			"Channel=arte HD (deu)",
			"[0]",
			"Date=03.08.2026",
			"Title=Die Löwin",
			"Info=Spielfilm Deutschland/Estland/Lettland 2024",
		},
	}
	if y := f.ExtractYearFromFile(); y != "2024" {
		t.Errorf("ExtractYearFromFile(txt with recording date) = %q, want 2024 (release year), got recording year", y)
	}
}

func TestExtractYearFromFileTimerName(t *testing.T) {
	// Timer Name is the most reliable source in .log.
	f := File{
		Name: "Die Löwin",
		TunerLog: []string{
			"Timer Name: Die Löwin - Spielfilm Deutschland/Estland/Lettland 2024",
			"23:00:08 Start Recording",
			"Total Size 7489,5 MB",
		},
	}
	if y := f.ExtractYearFromFile(); y != "2024" {
		t.Errorf("ExtractYearFromFile(timer name) = %q, want 2024", y)
	}
}

func TestSliceLogMetadataStopsAtNoise(t *testing.T) {
	// Lines after "Removed Filler Data" must not leak into metadata parsing.
	lines := []string{
		"Melodram Südafrika/2011",
		"Removed Filler Data: 117,8 MB",
		"22:58:00 / 00:00:00 (~ 0,00 MB) Start EPG Monitoring",
	}
	sliced := sliceLogMetadata(lines)
	if len(sliced) != 1 {
		t.Errorf("sliceLogMetadata kept %d lines, want 1 (stop at noise)", len(sliced))
	}
}

// Regression: yearFromMetaCandidate panicked with "slice bounds out of range"
// when the first genre/country word appears AFTER the first 4-digit number in
// the candidate (startIdx > mYearFirst[1]). Seen in production on
// "Tele-Gym (5 8)" and "Good bye, Lenin!" jobs crashing the whole service.
func TestYearFromMetaCandidateGenreAfterYear(t *testing.T) {
	candidates := []string{
		"2024 Spielfilm",       // genre after year
		"2023 Deutschland",     // country after year
		"1980er Jahre Spielfilm", // number-ish prefix, then genre
		"PID 5126 AC3 5.1",     // log-noise style: number then token
		"Spielfilm Deutschland 2024", // normal order must still work
		"Aerobic, Bewegung, Tanz",
		"Good bye, Lenin!",
	}
	for _, c := range candidates {
		if got := yearFromMetaCandidate(c); got != "" {
			t.Logf("yearFromMetaCandidate(%q) = %q (no panic, ok)", c, got)
		}
	}
	if y := ExtractYear("Spielfilm Deutschland 2024"); y != "2024" {
		t.Errorf("ExtractYear normal case = %q, want 2024", y)
	}
}

// Regression (production): "Mythos Marbella - Der Traum vom ewigen Sommer"
// captured the year 1954 from the plot text ("Den Grundstein legt 1954 Prinz
// Alfonso ...") instead of no year at all. Per the reference contract
// (add_rating_ui AGENTS.md 2.9 / movie_nfo_lib
// extract_txt_meta_year_and_countries) a narrative year in the description
// body is NEVER a release year: only the Info= line, the metadata segment
// before the first '|', or a trailing country/year segment qualify.
func TestExtractYearFromFileNarrativeYearIgnored(t *testing.T) {
	tests := []struct {
		name string
		file File
	}{
		{
			name: "Mythos Marbella (plot year 1954, no release year)",
			file: File{
				Name: "Mythos Marbella - Der Traum vom ewigen Sommer",
				MetadataLog: []string{
					"Title=Mythos Marbella - Der Traum vom ewigen Sommer",
					"Info=Film von Hannes Schuler",
					`Description=Vom Geheimtipp der High Society zum touristischen Hotspot Europas: Marbella steht wie kaum eine andere Stadt für Glamour, Reichtum und internationale Prominenz.||Den Grundstein legt 1954 Prinz Alfonso zu Hohenlohe mit der Eröffnung des "Marbella Club". Adel, Superreiche und Hollywoodstars entdecken die malerische Kulisse am Mittelmeer und machen Marbella zum Treffpunkt des internationalen Jetsets.|HD-Produktion||[16:9]   [PDC 15.09. 22:25]`,
				},
			},
		},
		{
			name: "England 1554 (plot year directly in description start)",
			file: File{
				Name: "Bergfried",
				MetadataLog: []string{
					"Info=",
					"Description=England 1554. Unter der Herrschaft der katholischen Königin Mary …",
				},
			},
		},
		{
			name: "recording dates in .log are not release years",
			file: File{
				Name: "Mythos Marbella - Der Traum vom ewigen Sommer",
				TunerLog: []string{
					"3sat HD (AC3,deu) 15/09/2026",
					"Timer Name: Mythos Marbella - Der Traum vom ewigen Sommer - Film von Hannes Schuler",
					"Timer Start: 15/09/2026 22:27:00",
					"Timer Duration: 00:48:00 (48 min. incl. 2 min. lead time, 2 min. follow-up time)",
				},
			},
		},
	}
	for _, tt := range tests {
		if y := tt.file.ExtractYearFromFile(); y != "" {
			t.Errorf("%s: got %q, want \"\" (narrative year is not a release year)", tt.name, y)
		}
	}
}

// The reference contract must still accept the real metadata segment before the
// first '|' (and the trailing country/year segment).
func TestExtractYearFromFileDescriptionMetaSegment(t *testing.T) {
	f := File{
		Name: "Bergfried",
		MetadataLog: []string{
			"Info=Historienfilm, Tschechien 2017",
			"Description=Historienfilm, Tschechien 2017|Martin Luther schlägt vor 500 Jahren seine Thesen an die Kirchentür …",
		},
	}
	if y := f.ExtractYearFromFile(); y != "2017" {
		t.Errorf("meta segment: got %q, want 2017", y)
	}

	// Info= empty, year only in the leading description segment.
	f2 := File{
		Name: "Bergfried",
		MetadataLog: []string{
			"Info=",
			"Description=Historienfilm, Tschechien 2017|Martin Luther schlägt vor 500 Jahren seine Thesen an die Kirchentür …",
		},
	}
	if y := f2.ExtractYearFromFile(); y != "2017" {
		t.Errorf("description meta segment: got %q, want 2017", y)
	}
}
