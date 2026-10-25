package media

import (
	"fmt"
	"testing"

	"github.com/Spiritreader/avior-go/consts"
)

func TestSanitize(t *testing.T) {
	//testFile := &File{Path: "D:\\Recording\\Monaco 110 - Madonna di Napoli.mkv"}
	//testFile := &File{Path: "D:\\Recording\\Neva Give üp - Der einzig wahre Japaner.mkv"}
	testFile := &File{Path: "D:\\Temp\\test.log.log"}
	testFile.Update()
	testFile.SanitizeLog()
	contains, _ := testFile.LogsContain([]string{"-rc vbr_hq -qmin 16 -qmax 23"}, []string{consts.MODULE_NAME_LOGMATCH})
	fmt.Printf("contains line: %t\n", contains)
	contains, _ = testFile.LogsContain([]string{"-c:v hevc_nvenc -preset p7 -tune hq"}, []string{consts.MODULE_NAME_LOGMATCH})
	fmt.Printf("contains line: %t\n", contains)
}

func TestAudioParsing(t *testing.T) {
	testFile := &File{Path: `\\UMS\recording_pool\Manual\Thomas Hengelbrock dirigiert Ravel und Franck.mkv`}
	testFile.Update()
}

func TestDuplicateNameKey(t *testing.T) {
	want := "aktivundgesundfaszientherapiepoolkeimestanduppaddling"
	for _, name := range []string{
		"aktiv und gesund - Faszientherapie - Poolkeime - Stand-up-Paddling",
		"aktiv und gesund   Faszientherapie   Poolkeime   Stand-up-Paddling",
		"aktiv und gesund _ Faszientherapie _ Poolkeime _ Stand-up-Paddling",
	} {
		if got := DuplicateNameKey(name); got != want {
			t.Errorf("DuplicateNameKey(%q) = %q, want %q", name, got, want)
		}
	}
	if got := DuplicateNameKey("Ä Ö Ü ß 2024"); got != "äöüß2024" {
		t.Errorf("DuplicateNameKey Unicode = %q", got)
	}
	if got := DuplicateNameKey("A/B"); got != DuplicateNameKey("AB") {
		t.Errorf("punctuation should not affect key: %q != %q", got, DuplicateNameKey("AB"))
	}
}

// Framing prefixes (documentation series) must not prevent duplicate detection.
// avior-dedup's semantic_prefixes serve the same purpose.
func TestDuplicateKeyForStripsPrefixes(t *testing.T) {
	patterns := []string{
		`terra\s*x\s*-\s*`,
		`42\s*-\s*die\s+antwort\s+auf\s+fast\s+alles`,
		`^erlebnis\s+erde\s*-?\s*`,
		`^abenteuer\s+wildnis\s*-?\s*`,
		`^heiter\s+bis\s+t[oö]dlich`,
	}
	same := [][2]string{
		{"Terra X - Die Reise zum Mars.mkv", "Die Reise zum Mars.mkv"},
		{"terra x - Der Mars.mkv", "Der Mars.mkv"},
		{"42 - Die Antwort auf fast alles - Schlaf.mkv", "Schlaf.mkv"},
		{"Erlebnis Erde - Wölfe.mkv", "Wölfe.mkv"},
		{"Erlebnis Erde Wölfe.mkv", "Wölfe.mkv"},
		{"Abenteuer Wildnis - Amazonas.mkv", "Amazonas.mkv"},
		{"Heiter bis tödlich - Nordisch herb.mkv", "Nordisch herb.mkv"},
	}
	for _, pair := range same {
		a := DuplicateKeyFor(pair[0], patterns)
		b := DuplicateKeyFor(pair[1], patterns)
		if a != b {
			t.Errorf("DuplicateKeyFor(%q)=%q != DuplicateKeyFor(%q)=%q", pair[0], a, pair[1], b)
		}
	}
	// A prefix in the middle of a name must stay: it is part of the title.
	if DuplicateKeyFor("Die Terra X Story.mkv", patterns) == DuplicateKeyFor("Story.mkv", patterns) {
		t.Error("mid-title occurrence must not be stripped")
	}
	// Unrelated titles stay distinct.
	if DuplicateKeyFor("Terra X - Mars.mkv", patterns) == DuplicateKeyFor("Erlebnis Erde - Venus.mkv", patterns) {
		t.Error("different titles must not collide")
	}
}

// Module suffixes appended when a replacement is parked must not hide the
// duplicate (avior-go: "Name LogMatchModule 2026-09-16 2106.mkv").
func TestDuplicateKeyForStripsModuleSuffix(t *testing.T) {
	patterns := []string{
		`\s+logmatchmodule.*$`,
		`\s+errorreplacemodule.*$`,
	}
	cases := [][2]string{
		{"Basic Instinct LogMatchModule 2026-09-16 2106.mkv", "Basic Instinct.mkv"},
		{"Basic Instinct ErrorReplaceModule 2024-01-02 1504.mkv", "Basic Instinct.mkv"},
	}
	for _, c := range cases {
		if a, b := DuplicateKeyFor(c[0], patterns), DuplicateKeyFor(c[1], patterns); a != b {
			t.Errorf("DuplicateKeyFor(%q)=%q != %q", c[0], a, b)
		}
	}
	// Titles that merely contain the word are untouched when no timestamp follows.
	if DuplicateKeyFor("LogMatchModule Story.mkv", patterns) == DuplicateKeyFor("Story.mkv", patterns) {
		t.Error("leading occurrence must not be stripped")
	}
}

// A broken pattern must not disable duplicate detection.
func TestDuplicateKeyForInvalidPatternIgnored(t *testing.T) {
	patterns := []string{`terra\s*x\s*-\s*`, `([unclosed`}
	if got := DuplicateKeyFor("Terra X - Mars.mkv", patterns); got != DuplicateKeyFor("Mars.mkv", patterns) {
		t.Errorf("valid pattern should still apply with an invalid one present: %q", got)
	}
}
