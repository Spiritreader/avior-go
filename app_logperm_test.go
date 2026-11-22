package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Every file in the log directory must end up world-readable/writable: lumberjack
// writes 0600, rotated files keep their old mode, and logs created before the rule
// existed stay 0600 unless the whole directory is corrected.
func TestFixLogPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions do not exist on windows")
	}
	dir := t.TempDir()
	names := []string{
		"main.log",
		"err.log",
		"processed.log",
		"skipped.log",
		"main-2026-09-11T19-32-44.082.log",
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0600); err != nil {
			t.Fatalf("setup %s: %s", name, err)
		}
	}
	subdir := filepath.Join(dir, "archive")
	if err := os.Mkdir(subdir, 0700); err != nil {
		t.Fatalf("setup dir: %s", err)
	}

	fixLogPermissions(dir)

	for _, name := range names {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("stat %s: %s", name, err)
		}
		if perm := info.Mode().Perm(); perm != 0666 {
			t.Errorf("%s mode = %o, want 666", name, perm)
		}
	}
	if info, err := os.Stat(subdir); err != nil {
		t.Fatalf("stat subdir: %s", err)
	} else if info.Mode().Perm() != 0700 {
		t.Errorf("subdirectory must stay untouched, mode = %o", info.Mode().Perm())
	}
}
