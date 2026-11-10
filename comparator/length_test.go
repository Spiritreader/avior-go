package comparator

import (
	"strings"
	"testing"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/media"
)

func newLengthModule(t *testing.T, settings map[string]interface{}) *LengthModule {
	t.Helper()
	m := &LengthModule{}
	m.Init(config.ModuleConfig{Enabled: true, Priority: 4, Settings: settings})
	return m
}

func lengthToleranceSettings() map[string]interface{} {
	return map[string]interface{}{
		"Tolerances": []interface{}{
			map[string]interface{}{"below_minutes": 10, "minutes": 2},
			map[string]interface{}{"below_minutes": 60, "minutes": 3},
			map[string]interface{}{"below_minutes": 90, "minutes": 4},
			map[string]interface{}{"below_minutes": 120, "minutes": 5},
			map[string]interface{}{"minutes": 6},
		},
	}
}

// Same rule as the duplicate check: allowed shortfall grows with the runtime.
func TestLengthModuleToleranceByRuntime(t *testing.T) {
	m := newLengthModule(t, lengthToleranceSettings())
	tests := []struct {
		name     string
		recorded int
		length   int
		want     string
	}{
		{"Der Schut: 2 min shortfall within 5 min", 113, 115, NOCH},
		{"50 min film, 3 min shortfall (tolerance 3)", 47, 50, NOCH},
		{"50 min film, 5 min shortfall (tolerance 3)", 45, 50, DISC},
		{"130 min film, 6 min shortfall (tolerance 6)", 124, 130, NOCH},
		{"130 min film, 8 min shortfall (tolerance 6)", 122, 130, DISC},
		{"9 min short film, 2 min shortfall (tolerance 2)", 7, 9, NOCH},
		{"9 min short film, 4 min shortfall (tolerance 2)", 5, 9, DISC},
	}
	for _, tt := range tests {
		_, got, msg := m.Run(media.File{RecordedLength: tt.recorded, Length: tt.length})
		if got != tt.want {
			t.Errorf("%s: got %s (%s), want %s", tt.name, got, msg, tt.want)
		}
	}
}

// Without a planned runtime nothing can be judged.
func TestLengthModuleWithoutPlannedRuntimeSkips(t *testing.T) {
	m := newLengthModule(t, lengthToleranceSettings())
	for _, f := range []media.File{
		{RecordedLength: 113, Length: -1},
		{RecordedLength: 113, Length: 0},
		{RecordedLength: -1, Length: 115},
	} {
		if _, result, msg := m.Run(f); result != NOCH {
			t.Errorf("recorded=%d length=%d: got %s (%s), want %s", f.RecordedLength, f.Length, result, msg, NOCH)
		}
	}
}

// A config without tolerances must not fall back to any percentage magic.
func TestLengthModuleWithoutTolerancesSkips(t *testing.T) {
	m := newLengthModule(t, map[string]interface{}{})
	if _, result, msg := m.Run(media.File{RecordedLength: 60, Length: 115}); result != NOCH {
		t.Errorf("no tolerances: got %s (%s), want %s", result, msg, NOCH)
	}
}

// Log evaluations match "recording too short".
func TestLengthModuleDiscardMessagePrefix(t *testing.T) {
	m := newLengthModule(t, lengthToleranceSettings())
	_, got, msg := m.Run(media.File{RecordedLength: 60, Length: 115})
	if got != DISC {
		t.Fatalf("expected DISC, got %s", got)
	}
	if !strings.HasPrefix(msg, "recording too short") {
		t.Errorf("message = %q, want prefix 'recording too short'", msg)
	}
}
