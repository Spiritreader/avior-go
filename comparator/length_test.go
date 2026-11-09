package comparator

import (
	"testing"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/media"
)

func newLengthModule(t *testing.T, threshold int) *LengthModule {
	t.Helper()
	m := &LengthModule{}
	m.Init(config.ModuleConfig{Enabled: true, Priority: 4, Settings: map[string]interface{}{"Threshold": threshold}})
	return m
}

// Without a planned runtime the percentage is meaningless: Length = -1 would
// yield an absurd value and discard every recording that lacks a .txt.
func TestLengthModuleWithoutPlannedRuntimeSkips(t *testing.T) {
	m := newLengthModule(t, 20)
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

func TestLengthModuleJudgesWithPlannedRuntime(t *testing.T) {
	m := newLengthModule(t, 20)
	if _, result, _ := m.Run(media.File{RecordedLength: 113, Length: 115}); result != NOCH {
		t.Errorf("2%% shortfall at 20%% threshold: got %s, want %s", result, NOCH)
	}
	if _, result, _ := m.Run(media.File{RecordedLength: 60, Length: 115}); result != DISC {
		t.Errorf("48%% shortfall at 20%% threshold: got %s, want %s", result, DISC)
	}
}
