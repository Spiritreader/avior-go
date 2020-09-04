package comparator

import (
	"fmt"
	"os"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/Spiritreader/avior-go/tools"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)


type MaxSizeModule struct {
	moduleConfig *structs.ModuleConfig
}

func (s *MaxSizeModule) Init(mcfg structs.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *MaxSizeModule) Run(files ...media.File) (string, string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return s.Name(), NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return s.Name(), NOCH, "disabled"
	}
	settings := &structs.MaxSizeModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return s.Name(), NOCH, "err"
	}
	if settings.MaxSize == 0 {
		return s.Name(), NOCH, "no limit specified"
	}
	fileInfo, err := os.Stat(files[0].Path)
	if err != nil {
		_ = glg.Warnf("couldn't open file %s for metadata retrieval: %s", files[0].Path, err)
		// if ms.Name(), odule fails, disable module
		return s.Name(), NOCH, "err no access"
	}
	fileSizeGb, _ := tools.ByteCountUpSI(fileInfo.Size(), 3)
	maxSize, _ := tools.ByteCountDownSI(float64(settings.MaxSize), 4, 4)
	if int64(fileSizeGb) > int64(settings.MaxSize) {
		difference := tools.ByteCountSI(fileInfo.Size() - int64(maxSize))
		return s.Name(), KEEP, fmt.Sprintf("file %s larger than %dGB", difference, settings.MaxSize)
	}
	return s.Name(), NOCH, "ok"
}

func (s *MaxSizeModule) Priority() int {
	if s.moduleConfig == nil {
		return -1
	}
	return s.moduleConfig.Priority
}

func (s *MaxSizeModule) Name() string {
	return consts.MODULE_NAME_MAXSIZE
}


