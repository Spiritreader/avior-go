package comparator

import (
	"fmt"
	"os"
	"time"

	"github.com/Spiritreader/avior-go/config"
	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)

type AgeModule struct {
	moduleConfig *config.ModuleConfig
}

func (s *AgeModule) Init(mcfg config.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *AgeModule) Run(files ...media.File) (string, string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return s.Name(), NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return s.Name(), NOCH, "disabled"
	}
	settings := &config.AgeModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return s.Name(), NOCH, "err no convert"
	}

	duplicateFileInfo, err := os.Stat(files[1].Path)
	if err != nil {
		_ = glg.Warnf("could not open file \"%s\" for metadata retrieval: %s", files[1].Path, err)
		// if module fails, disable module
		return s.Name(), NOCH, "err no access"
	}
	thresholdTime := time.Now().Add(time.Duration(-1*settings.MaxAge) * time.Hour)
	_ = glg.Debugf("age module threshold time: %s", thresholdTime)
	_ = glg.Debugf("age module duplicate modtime: %s", duplicateFileInfo.ModTime())
	if duplicateFileInfo.ModTime().After(thresholdTime) {
		//difference := duplicateFileInfo.ModTime().Sub(thresholdTime).Hours() / 24
		difference := thresholdTime.Sub(duplicateFileInfo.ModTime()).Hours() / 24
		return s.Name(), KEEP, fmt.Sprintf("%.1f days old, /%d days minimum", difference, settings.MaxAge)
	}
	return s.Name(), NOCH, "ok"
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
