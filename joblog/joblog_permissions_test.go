package joblog

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAppendToCreatesWorldWritableInfoLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions do not exist on windows")
	}

	path := filepath.Join(t.TempDir(), "recording.ts.INFO.log")
	var log Data
	log.Add("test")

	if err := log.AppendTo(path, false, false); err != nil {
		t.Fatalf("AppendTo: %s", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat INFO.log: %s", err)
	}
	if got := info.Mode().Perm(); got != 0666 {
		t.Errorf("INFO.log mode = %o, want 666", got)
	}
}
