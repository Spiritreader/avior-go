package worker

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
