package comparator

import (
	"fmt"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)

type LengthModule struct {
	moduleConfig *config.ModuleConfig
}

func (s *LengthModule) Init(mcfg config.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *LengthModule) Run(files ...media.File) (string, string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return s.Name(), NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return s.Name(), NOCH, "disabled"
	}
	settings := &config.LengthModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return s.Name(), NOCH, "err"
	}
	file := files[0]
	// Same rule as DuplicateLengthCheckModule: the recording is judged against its
	// own planned runtime. Without one the module cannot tell a truncated
	// recording from a short film, so it must not discard anything.
	if file.RecordedLength <= 0 || file.Length <= 0 || len(settings.Tolerances) == 0 {
		return s.Name(), NOCH, "no planned runtime for the recording, skipping module"
	}

	tolerance := toleranceFor(settings.Tolerances, file.Length)
	shortfall := file.Length - file.RecordedLength
	if shortfall > tolerance {
		return s.Name(), DISC, fmt.Sprintf("recording too short: %dm recorded vs %dm planned (shortfall %dm > tolerance %dm)",
			file.RecordedLength, file.Length, shortfall, tolerance)
	}
	return s.Name(), NOCH, fmt.Sprintf("ok: %dm recorded vs %dm planned (shortfall %dm <= tolerance %dm)",
		file.RecordedLength, file.Length, shortfall, tolerance)
}

func (s *LengthModule) Priority() int {
	if s.moduleConfig == nil {
		return -1
	}
	return s.moduleConfig.Priority
}

func (s *LengthModule) Name() string {
	return consts.MODULE_NAME_LENGTH
}
