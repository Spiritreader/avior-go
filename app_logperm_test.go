package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Runtime logs must end up world-readable/writable; lumberjack creates them 0600.
func TestFixLogPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions do not exist on windows")
	}
	dir := t.TempDir()
	mainLog := filepath.Join(dir, "main.log")
	errLog := filepath.Join(dir, "err.log")
	for _, p := range []string{mainLog, errLog} {
		if err := os.WriteFile(p, []byte("x"), 0600); err != nil {
			t.Fatalf("setup: %s", err)
		}
	}

	fixLogPermissions(dir, "main.log", "err.log", "skipped.log") // skipped.log absent

	for _, p := range []string{mainLog, errLog} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %s: %s", p, err)
		}
		if perm := info.Mode().Perm(); perm != 0666 {
			t.Errorf("%s mode = %o, want 666", filepath.Base(p), perm)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "skipped.log")); !os.IsNotExist(err) {
		t.Errorf("missing log must not be created, stat err = %v", err)
	}
}
