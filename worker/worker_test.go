package worker

import (
	"os"
	"path/filepath"
	"testing"

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
