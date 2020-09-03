package comparator

import (
	"fmt"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)

type AgeModule struct {
	moduleConfig *structs.ModuleConfig
}

func (s *AgeModule) Init(mcfg structs.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *AgeModule) Run() string {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return NOCH
	}
	if !s.moduleConfig.Enabled {
		return NOCH
	}
	settings := &structs.AgeModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s: %s", s.Name(), err)
	}
	fmt.Println(settings.MaxAge)
	return "not_implemented"
}

func (s *AgeModule) Priority() int {
	if s.moduleConfig == nil {
		return -1
	}
	return s.moduleConfig.Priority
}

func (s *AgeModule) Name() string {
	return consts.MODULE_NAME_AGE
}
