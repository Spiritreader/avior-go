package comparator

import (
	"testing"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
)

// steps as configured in the GUI: runtime bracket -> allowed shortfall in minutes
func toleranceSteps() []config.ToleranceStep {
	return []config.ToleranceStep{
		{BelowMinutes: 10, Minutes: 2},
		{BelowMinutes: 60, Minutes: 3},
		{BelowMinutes: 90, Minutes: 4},
		{BelowMinutes: 120, Minutes: 5},
		{Minutes: 6},
	}
}

func runModule(t *testing.T, newFile, dupe media.File, threshold int) (string, string) {
	t.Helper()
	m := &DuplicateLengthCheckModule{}
	m.Init(config.ModuleConfig{
		Enabled:  true,
		Priority: 4,
		Settings: map[string]interface{}{
			"Threshold":  threshold,
			"Tolerances": toleranceSteps(),
		},
	})
	_, result, msg := m.Run(newFile, dupe)
	return result, msg
}

func TestDuplicateLengthToleranceByRuntime(t *testing.T) {
	tests := []struct {
		name     string
		recorded int
		length   int
		dupeLen  int
		want     string
	}{
		// Production case: 113 min recorded, 115 min planned -> 2 min shortfall
		{"Der Schut: 2 min shortfall within 5 min tolerance", 113, 115, 138, NOCH},
		{"3 min shortfall on a 50 min film, tolerance 3", 47, 50, 60, NOCH},
		{"5 min shortfall on a 50 min film, tolerance 3", 45, 50, 60, DISC},
		{"8 min shortfall on a 130 min film, tolerance 6", 122, 130, 140, DISC},
		{"2 min shortfall on a 130 min film, tolerance 6", 128, 130, 140, NOCH},
		{"short film: 2 min shortfall on 9 min, tolerance 2", 7, 9, 12, NOCH},
		{"short film: 4 min shortfall on 9 min, tolerance 2", 5, 9, 12, DISC},
	}
	for _, tt := range tests {
		newFile := media.File{RecordedLength: tt.recorded, Length: tt.length}
		dupe := media.File{RecordedLength: tt.dupeLen, Length: -1}
		got, msg := runModule(t, newFile, dupe, 3)
		if got != tt.want {
			t.Errorf("%s: got %s (%s), want %s", tt.name, got, msg, tt.want)
		}
	}
}

// Without EPG data for the new recording, fall back to the previous
// percent comparison against the existing file.
func TestDuplicateLengthFallbackWithoutEPG(t *testing.T) {
	newFile := media.File{RecordedLength: 113, Length: -1}
	dupe := media.File{RecordedLength: 138, Length: -1}
	if got, msg := runModule(t, newFile, dupe, 3); got != DISC {
		t.Errorf("no EPG + 18%% shorter: got %s (%s), want %s", got, msg, DISC)
	}
	newFile2 := media.File{RecordedLength: 136, Length: -1}
	if got, msg := runModule(t, newFile2, dupe, 3); got != NOCH {
		t.Errorf("no EPG + 1.4%% shorter: got %s (%s), want %s", got, msg, NOCH)
	}
}

// Missing length data on either side keeps the previous "skip module" behaviour.
func TestDuplicateLengthMissingData(t *testing.T) {
	newFile := media.File{RecordedLength: -1, Length: -1}
	dupe := media.File{RecordedLength: 138, Length: -1}
	if got, _ := runModule(t, newFile, dupe, 3); got != NOCH {
		t.Errorf("missing recorded length: got %s, want %s", got, NOCH)
	}
}

// The module must be usable without the tolerance list (legacy configs).
func TestDuplicateLengthWithoutTolerances(t *testing.T) {
	m := &DuplicateLengthCheckModule{}
	m.Init(config.ModuleConfig{Enabled: true, Priority: 4, Settings: map[string]interface{}{"Threshold": 3}})
	_, result, _ := m.Run(media.File{RecordedLength: 113, Length: 115}, media.File{RecordedLength: 138, Length: -1})
	if result != DISC {
		t.Errorf("legacy behaviour without tolerances: got %s, want %s", result, DISC)
	}
}

var _ = consts.MODULE_NAME_DUPLICATELENGTHCHECK

// The settings arrive as decoded JSON (map[string]interface{} with nested
// []interface{}), so the mapstructure tags must hold for that shape too.
func TestDuplicateLengthTolerancesFromDecodedJSON(t *testing.T) {
	m := &DuplicateLengthCheckModule{}
	m.Init(config.ModuleConfig{
		Enabled:  true,
		Priority: 4,
		Settings: map[string]interface{}{
			"Threshold": 3,
			"Tolerances": []interface{}{
				map[string]interface{}{"below_minutes": 10, "minutes": 2},
				map[string]interface{}{"below_minutes": 60, "minutes": 3},
				map[string]interface{}{"below_minutes": 90, "minutes": 4},
				map[string]interface{}{"below_minutes": 120, "minutes": 5},
				map[string]interface{}{"minutes": 6},
			},
		},
	})
	// Der Schut: 113 recorded vs 115 planned -> 2 min shortfall, bracket <120 -> 5 min
	_, result, msg := m.Run(media.File{RecordedLength: 113, Length: 115}, media.File{RecordedLength: 138, Length: -1})
	if result != NOCH {
		t.Errorf("decoded JSON settings: got %s (%s), want %s", result, msg, NOCH)
	}
	// clearly incomplete: 100 vs 115 -> 15 min shortfall > 5
	_, result2, msg2 := m.Run(media.File{RecordedLength: 100, Length: 115}, media.File{RecordedLength: 138, Length: -1})
	if result2 != DISC {
		t.Errorf("decoded JSON settings: got %s (%s), want %s", result2, msg2, DISC)
	}
}
