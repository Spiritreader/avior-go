package comparator

import (
	"fmt"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)

type LogMatchModule struct {
	moduleConfig *structs.ModuleConfig
}

func (s *LogMatchModule) Init(mcfg structs.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *LogMatchModule) Run(files ...media.File) (string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return NOCH, "disabled"
	}
	settings := &structs.LogMatchModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return NOCH, "err"
	}
	cfg := config.Instance()
	duplicate := files[1]
	excludeMatches, excludeTerm := duplicate.LogsContain(cfg.Shared.LogExclude)
	includeMatches, includeTerm := duplicate.LogsContain(cfg.Shared.LogInclude)

	switch settings.Mode {
	case consts.LOGMATCH_MODE_INCLUDE:
		if includeMatches {
			return REPL, fmt.Sprintf("include match (mode %s): %s", consts.LOGMATCH_MODE_INCLUDE, includeTerm)
		} else if excludeMatches {
			return KEEP, fmt.Sprintf("exclude match (mode %s): %s", consts.LOGMATCH_MODE_INCLUDE, includeTerm)
		}
	case consts.LOGMATCH_MODE_NEUTRAL:
		if includeMatches && excludeMatches {
			return KEEP, fmt.Sprintf("include and exclude match (mode %s): %s | %s",
				consts.LOGMATCH_MODE_NEUTRAL, includeTerm, excludeTerm)
		} else if includeMatches {
			return REPL, fmt.Sprintf("include match (mode %s): %s", consts.LOGMATCH_MODE_NEUTRAL, includeTerm)
		} else if excludeMatches {
			return KEEP, fmt.Sprintf("exclude match (mode %s): %s", consts.LOGMATCH_MODE_NEUTRAL, includeTerm)
		}
	case consts.LOGMATCH_MODE_EXCLUDE:
		if excludeMatches {
			return KEEP, fmt.Sprintf("exclude match (mode %s): %s", consts.LOGMATCH_MODE_EXCLUDE, includeTerm)
		} else if includeMatches {
			return REPL, fmt.Sprintf("include match (mode %s): %s", consts.LOGMATCH_MODE_EXCLUDE, includeTerm)
		}
	}
	return NOCH, "no matches"
}

func (s *LogMatchModule) Priority() int {
	return -1
}

func (s *LogMatchModule) Name() string {
	return consts.MODULE_NAME_LOGMATCH
}
