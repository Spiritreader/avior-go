package comparator

import (
	"fmt"
	"sort"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)

type DuplicateLengthCheckModule struct {
	moduleConfig *config.ModuleConfig
}

func (s *DuplicateLengthCheckModule) Init(mcfg config.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *DuplicateLengthCheckModule) Run(files ...media.File) (string, string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return s.Name(), NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return s.Name(), NOCH, "disabled"
	}
	settings := &config.DuplicateLengthCheckSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return s.Name(), NOCH, "err"
	}
	// The new recording is judged against its own planned runtime. The existing
	// file's runtime is never a reference: sender EPG durations include
	// commercial breaks and old recordings may have overrun, so a shorter but
	// complete recording is allowed to replace them.
	file := files[0]
	if file.RecordedLength <= 0 || file.Length <= 0 || len(settings.Tolerances) == 0 {
		return s.Name(), NOCH, "no planned runtime for the new recording, skipping module"
	}

	tolerance := toleranceFor(settings.Tolerances, file.Length)
	shortfall := file.Length - file.RecordedLength
	if shortfall > tolerance {
		// Keep the historic prefix: log evaluations match on
		// "no encode - new file too short for replacement".
		return s.Name(), DISC, fmt.Sprintf("new file too short for replacement: %dm recorded vs %dm planned (shortfall %dm > tolerance %dm)",
			file.RecordedLength, file.Length, shortfall, tolerance)
	}
	return s.Name(), NOCH, fmt.Sprintf("ok: %dm recorded vs %dm planned (shortfall %dm <= tolerance %dm)",
		file.RecordedLength, file.Length, shortfall, tolerance)
}

// toleranceFor returns the allowed shortfall in minutes for a recording of the
// given planned runtime. Steps are sorted ascending by BelowMinutes; a step
// without BelowMinutes is the catch-all for anything longer.
func toleranceFor(steps []config.ToleranceStep, runtimeMinutes int) int {
	sorted := make([]config.ToleranceStep, len(steps))
	copy(sorted, steps)
	sort.Slice(sorted, func(i, j int) bool {
		bi, bj := sorted[i].BelowMinutes, sorted[j].BelowMinutes
		if bi == 0 {
			return false // catch-all last
		}
		if bj == 0 {
			return true
		}
		return bi < bj
	})
	for _, step := range sorted {
		if step.BelowMinutes == 0 || runtimeMinutes < step.BelowMinutes {
			return step.Minutes
		}
	}
	last := 0
	for _, step := range sorted {
		if step.Minutes > last {
			last = step.Minutes
		}
	}
	return last
}

func (s *DuplicateLengthCheckModule) Priority() int {
	if s.moduleConfig == nil {
		return -1
	}
	return s.moduleConfig.Priority
}

func (s *DuplicateLengthCheckModule) Name() string {
	return consts.MODULE_NAME_DUPLICATELENGTHCHECK
}
