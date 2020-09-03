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

func (s *LengthModule) Run(files ...media.File) (string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return NOCH, "disabled"
	}
	settings := &structs.LengthModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return NOCH, "err"
	}
	file := files[0]
	diff := file.LengthDifference()
	if diff > settings.Threshold {
		return KEEP, fmt.Sprintf("recording too short: %dm / %dm (%d%% / %d%%)",
			file.RecordedLength, file.Length, diff, settings.Threshold)
	}
	return NOCH, fmt.Sprintf("ok: %dm / %dm (%d%% / %d%%)",
		file.RecordedLength, file.Length, diff, settings.Threshold)
}

func (s *LengthModule) Priority() int {
	return -1
}

func (s *LengthModule) Name() string {
	return consts.MODULE_NAME_LENGTH
}
