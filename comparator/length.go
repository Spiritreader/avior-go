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
	diff := file.LengthDifference()
	if diff > settings.Threshold {
		return s.Name(), DISC, fmt.Sprintf("recording too short: r:%dm / l:%dm (d:%d%% / t:%d%%)",
			file.RecordedLength, file.Length, diff, settings.Threshold)
	}
	return s.Name(), NOCH, fmt.Sprintf("ok: r:%dm / l:%dm (d:%d%% / t:%d%%)",
		file.RecordedLength, file.Length, diff, settings.Threshold)
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
