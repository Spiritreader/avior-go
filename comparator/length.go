package comparator

import (
	"fmt"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)

type LengthModule struct {
	moduleConfig *structs.ModuleConfig
}

func (s *LengthModule) Init(mcfg structs.ModuleConfig) {
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
	settings := &structs.LengthModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return s.Name(), NOCH, "err"
	}
	file := files[0]
	if file.IgnoreLength {
		return s.Name(), NOCH, "module has been overridden"
	}
	diff := file.LengthDifference()
	if diff > settings.Threshold {
		return s.Name(), KEEP, fmt.Sprintf("recording too short: %dm / %dm (%d%% / %d%%)",
			file.RecordedLength, file.Length, diff, settings.Threshold)
	}
	return s.Name(), NOCH, fmt.Sprintf("difference: %d", diff)
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
