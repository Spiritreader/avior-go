package worker

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spiritreader/avior-go/comparator"
	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/joblog"
	"github.com/Spiritreader/avior-go/media"
)

func TestTraverse(t *testing.T) {
	file := media.File{
		Path: "\\\\UMS\\media\\transcoded\\2075 - Verbrannte Erde.mkv",
		Name: "2075 - Verbrannte Erde",
	}
	traverseDir(&file, "\\\\UMS\\media\\transcoded", false)
}

// A fully readable directory must traverse without error. godirwalk v1.17.0
// regressed here on Windows: its scanner stores the io.EOF that ends a normal
// Readdir loop as the scan error, and Walk returns it without consulting
// ErrorCallback, so every successful walk looked like a failure.
func TestTraverseDirCleanDirHasNoError(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("setup: %s", err)
	}
	for _, name := range []string{"a.mkv", filepath.Join("sub", "b.mkv")} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("setup: %s", err)
		}
	}

	file := media.File{Path: filepath.Join(root, "a.mkv"), Name: "a"}
	if _, err := traverseDir(&file, root, false); err != nil {
		t.Errorf("traverseDir() over a clean directory returned error = %v, want nil", err)
	}
}

func TestTraverseDirMatchesPunctuationVariants(t *testing.T) {
	root := t.TempDir()
	existing := "aktiv und gesund _ Faszientherapie _ Poolkeime _ Stand-up-Paddling.mkv"
	path := filepath.Join(root, existing)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write existing file: %s", err)
	}
	file := media.File{
		Name: "aktiv und gesund - Faszientherapie - Poolkeime - Stand-up-Paddling",
	}
	matches, err := traverseDir(&file, root, false)
	if err != nil {
		t.Fatalf("traverseDir() returned error: %s", err)
	}
	if len(matches) != 1 || matches[0].Path != path {
		t.Fatalf("traverseDir() matches = %#v, want original path %q", matches, path)
	}
}

// The configured strip patterns must also apply during a real directory scan,
// not just in the pure key helper: a "Terra X - " source must match the plain
// library file.
func TestTraverseDirMatchesStrippedPrefix(t *testing.T) {
	cfg := config.Instance()
	prev := cfg.Shared.DuplicateStripPatterns
	cfg.Shared.DuplicateStripPatterns = []string{`terra\s*x\s*-\s*`}
	t.Cleanup(func() { cfg.Shared.DuplicateStripPatterns = prev })

	root := t.TempDir()
	existing := "Die Reise zum Mars.mkv"
	path := filepath.Join(root, existing)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write existing file: %s", err)
	}
	file := media.File{Name: "Terra X - Die Reise zum Mars"}
	matches, err := traverseDir(&file, root, false)
	if err != nil {
		t.Fatalf("traverseDir() returned error: %s", err)
	}
	if len(matches) != 1 || matches[0].Path != path {
		t.Fatalf("traverseDir() matches = %#v, want %q", matches, path)
	}
}

// The strip patterns must be applied to BOTH sides of the comparison: the newly
// derived output name (Name + Subtitle after excludes/char replacement) and the
// existing library file. The output naming itself stays unchanged.
func TestDuplicateNameMatchStripsPrefixFromOutputName(t *testing.T) {
	cfg := config.Instance()
	prev := cfg.Shared.DuplicateStripPatterns
	cfg.Shared.DuplicateStripPatterns = []string{`terra\s*x\s*-\s*`, `^erlebnis\s+erde\s*-?\s*`}
	t.Cleanup(func() { cfg.Shared.DuplicateStripPatterns = prev })

	tests := []struct {
		name      string
		file      media.File
		existing  string
		wantMatch bool
	}{
		{
			name: "prefix only in output name",
			file: media.File{
				Name:     "Terra X - Die Reise zum Mars",
				Subtitle: "Dokumentation Deutschland 2024",
			},
			existing:  "Die Reise zum Mars - Dokumentation Deutschland 2024.mkv",
			wantMatch: true,
		},
		{
			name: "prefix in both names",
			file: media.File{
				Name:     "Terra X - Die Reise zum Mars",
				Subtitle: "Dokumentation Deutschland 2024",
			},
			existing:  "Terra X - Die Reise zum Mars - Dokumentation Deutschland 2024.mkv",
			wantMatch: true,
		},
		{
			name:      "erlebnis erde prefix",
			file:      media.File{Name: "Erlebnis Erde - Wölfe", Subtitle: "Dokumentation Deutschland 2022"},
			existing:  "Wölfe - Dokumentation Deutschland 2022.mkv",
			wantMatch: true,
		},
		{
			name:      "different episode does not match",
			file:      media.File{Name: "Terra X - Die Reise zum Mars", Subtitle: "Dokumentation Deutschland 2024"},
			existing:  "Die Reise zur Venus - Dokumentation Deutschland 2024.mkv",
			wantMatch: false,
		},
		{
			name:      "output name stays unchanged",
			file:      media.File{Name: "Terra X - Die Reise zum Mars", Subtitle: "Dokumentation Deutschland 2024"},
			existing:  "Unrelated.mkv",
			wantMatch: false,
		},
	}
	for _, tt := range tests {
		candidate := tt.file.OutName() + ".mkv"
		got := duplicateNameMatch(candidate, tt.existing, ".mkv")
		if got != tt.wantMatch {
			t.Errorf("%s: duplicateNameMatch(%q, %q) = %v, want %v", tt.name, candidate, tt.existing, got, tt.wantMatch)
		}
	}

	// The output name used for encoding must remain the original one.
	f := media.File{Name: "Terra X - Die Reise zum Mars", Subtitle: "Dokumentation Deutschland 2024"}
	if got := f.OutName(); got != "Terra X - Die Reise zum Mars - Dokumentation Deutschland 2024" {
		t.Errorf("OutName() changed: %q", got)
	}
}

// The .INFO.log files must be world-readable/writable on Unraid so every
// user (nobody:users) can read/write/delete them. lumberjack creates files
// with 0600, so writeSkippedLog must fix the mode explicitly.
func TestWriteSkippedLogInfoPerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions do not exist on windows")
	}
	path := filepath.Join(t.TempDir(), "test.ts")
	f := media.File{Path: path}
	writeSkippedLog(&f, new(joblog.Data), false)
	fi, err := os.Stat(path + ".INFO.log")
	if err != nil {
		t.Fatalf("stat INFO.log: %s", err)
	}
	if got := fi.Mode().Perm(); got != 0o666 {
		t.Errorf("INFO.log mode = %o, want 666", got)
	}
}

// The quality actions decide whether both versions are kept. Defaults keep the
// previous behaviour (discard on lower quality, replace on higher quality).
func TestKeepBothFor(t *testing.T) {
	tests := []struct {
		name         string
		res          string
		module       string
		lower        string
		higher       string
		wantKeepBoth bool
	}{
		{"lower quality, discard (default)", comparator.DISC, consts.MODULE_NAME_RESOLUTION, consts.QUALITY_ACTION_DISCARD, consts.QUALITY_ACTION_REPLACE, false},
		{"lower quality, keep", comparator.DISC, consts.MODULE_NAME_RESOLUTION, consts.QUALITY_ACTION_KEEP, consts.QUALITY_ACTION_REPLACE, true},
		{"lower audio, keep", comparator.DISC, consts.MODULE_NAME_AUDIO, consts.QUALITY_ACTION_KEEP, consts.QUALITY_ACTION_REPLACE, true},
		{"higher quality, replace (default)", comparator.REPL, consts.MODULE_NAME_RESOLUTION, consts.QUALITY_ACTION_DISCARD, consts.QUALITY_ACTION_REPLACE, false},
		{"higher quality, keep", comparator.REPL, consts.MODULE_NAME_RESOLUTION, consts.QUALITY_ACTION_DISCARD, consts.QUALITY_ACTION_KEEP, true},
		{"higher audio, keep", comparator.REPL, consts.MODULE_NAME_AUDIO, consts.QUALITY_ACTION_DISCARD, consts.QUALITY_ACTION_KEEP, true},
		{"other module is never affected (discard)", comparator.DISC, consts.MODULE_NAME_LENGTH, consts.QUALITY_ACTION_KEEP, consts.QUALITY_ACTION_KEEP, false},
		{"other module is never affected (replace)", comparator.REPL, consts.MODULE_NAME_ERRORREPLACE, consts.QUALITY_ACTION_KEEP, consts.QUALITY_ACTION_KEEP, false},
		{"no decision", comparator.NOCH, consts.MODULE_NAME_RESOLUTION, consts.QUALITY_ACTION_KEEP, consts.QUALITY_ACTION_KEEP, false},
	}
	for _, tt := range tests {
		got := keepBothFor(tt.res, tt.module, tt.lower, tt.higher)
		if got != tt.wantKeepBoth {
			t.Errorf("%s: keepBothFor(%q,%q,%q,%q) = %v, want %v",
				tt.name, tt.res, tt.module, tt.lower, tt.higher, got, tt.wantKeepBoth)
		}
	}
}

func TestKeptBothAlreadyPresent(t *testing.T) {
	tests := []struct {
		exitCode int
		suffix   string
		want     bool
	}{
		{107, "720p, STEREO", true},
		{107, "", false}, // ordinary "file already exists" stays an error
		{108, "720p, STEREO", false},
		{0, "720p, STEREO", false},
	}
	for _, tt := range tests {
		if got := keptBothAlreadyPresent(tt.exitCode, tt.suffix); got != tt.want {
			t.Errorf("keptBothAlreadyPresent(%d, %q) = %v, want %v", tt.exitCode, tt.suffix, got, tt.want)
		}
	}
}
