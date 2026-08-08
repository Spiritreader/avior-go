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
		{"Die Löwin - Spielfilm Deutschland/Estland/Lettland 2024", "2024"},
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
