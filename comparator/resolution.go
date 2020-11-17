package comparator

import (
	"fmt"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)

type ResolutionModule struct {
	moduleConfig *config.ModuleConfig
}

func (s *ResolutionModule) Init(mcfg config.ModuleConfig) {
	s.moduleConfig = &mcfg
}


func (s *ResolutionModule) Run(files ...media.File) (string, string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return s.Name(), NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return s.Name(), NOCH, "disabled"
	}
	settings := &config.ResolutionModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return s.Name(), NOCH, "err"
	}
	new := files[0]
	old := files[1]
	newResolution, err := new.Resolution.GetPixels()
	if err != nil {
		_ = glg.Errorf("could not convert resolution %s to pixel value: %s", new.Resolution.Value, err)
		return s.Name(), NOCH, "err"
	}
	oldResolution, err := old.Resolution.GetPixels()
	if err != nil {
		_ = glg.Errorf("could not convert resolution %s to pixel value: %s", old.Resolution.Value, err)
		return s.Name(), NOCH, "err"
	}
	ratio := int(float64(newResolution) / float64(oldResolution) * 100)
	if ratio == 100 {
		return s.Name(), NOCH, "resolution is the same"
	} else if ratio > 100 {
		if ratio - 100 >= settings.MinResolution {
			return s.Name(), REPL, fmt.Sprintf("new file better: %s vs %s", new.Resolution.Value, old.Resolution.Value)
		} else {
			return s.Name(), NOCH, fmt.Sprintf("minimum resolution improvement of %d%% not met: %d%%", settings.MinResolution, ratio - 100)
		}
	} else {
		return s.Name(), DISC, fmt.Sprintf("old file better: %s vs %s", new.Resolution.Value, old.Resolution.Value)
	}
}

func (s *ResolutionModule) Priority() int {
	if s.moduleConfig == nil {
		return -1
	}
	return s.moduleConfig.Priority
}

func (s *ResolutionModule) Name() string {
	return consts.MODULE_NAME_RESOLUTION
}
