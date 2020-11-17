package comparator

import (
	"fmt"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)


type ErrorSkipModule struct {
	moduleConfig *config.ModuleConfig
}

func (s *ErrorSkipModule) Init(mcfg config.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *ErrorSkipModule) Run(files ...media.File) (string, string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return s.Name(), NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return s.Name(), NOCH, "disabled"
	}
	settings := &config.ErrorModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return s.Name(), NOCH, "err"
	}

	if files[0].Errors > settings.Threshold {
		return s.Name(), DISC, fmt.Sprintf("file has too many errors (count/allowed): %d/%d", files[0].Errors, settings.Threshold)
	} else {
		return s.Name(), NOCH, "ok"
	}
}

func (s *ErrorSkipModule) Priority() int {
	if s.moduleConfig == nil {
		return -1
	}
	return s.moduleConfig.Priority
}

func (s *ErrorSkipModule) Name() string {
	return consts.MODULE_NAME_ERRORSKIP
}
