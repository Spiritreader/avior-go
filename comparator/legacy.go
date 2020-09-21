package comparator

import (
	"fmt"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
)


type LegacyModule struct {
	moduleConfig *config.ModuleConfig
}

func (s *LegacyModule) Init(mcfg config.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *LegacyModule) Run(files ...media.File) (string, string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return s.Name(), NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return s.Name(), NOCH, "disabled"
	}
	if files[1].Legacy() {
		return s.Name(), REPL, fmt.Sprintf("\"%s\" is legacy", files[0].Name)
	} else {
		return s.Name(), NOCH, "ok"
	}
}

func (s *LegacyModule) Priority() int {
	if s.moduleConfig == nil {
		return -1
	}
	return s.moduleConfig.Priority
}

func (s *LegacyModule) Name() string {
	return consts.MODULE_NAME_LEGACY
}
