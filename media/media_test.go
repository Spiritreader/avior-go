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
