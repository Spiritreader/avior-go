package comparator

import (
	"fmt"

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
	file := files[0]
	duplicate := files[1]
	if files[0].RecordedLength == -1 || files[1].RecordedLength == -1 {
		return s.Name(), NOCH, "duplicate had insufficient length data, skipping module"
	}
	// discard file if new file is shorter than the threshold compared to the duplicate
	diff := float64(1) - float64(file.RecordedLength / duplicate.RecordedLength)
	if (diff * 100) > float64(settings.Threshold) {
		return s.Name(), DISC, fmt.Sprintf("new file too short for replacement, (n:%dm/d:%dm) with diff: %d%%", 
			file.RecordedLength, duplicate.RecordedLength, int64(diff))
	}
	return s.Name(), NOCH, "ok"	
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
