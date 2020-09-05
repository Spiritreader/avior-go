package comparator

import (
	"fmt"

	"github.com/Spiritreader/avior-go/consts"
	"github.com/Spiritreader/avior-go/media"
	"github.com/Spiritreader/avior-go/structs"
	"github.com/kpango/glg"
	"github.com/mitchellh/mapstructure"
)

type AudioModule struct {
	moduleConfig *structs.ModuleConfig
}

func (s *AudioModule) Init(mcfg structs.ModuleConfig) {
	s.moduleConfig = &mcfg
}

func (s *AudioModule) Run(files ...media.File) (string, string, string) {
	if s.moduleConfig == nil {
		_ = glg.Warnf("module %s has never been initialized and has thus been disabled", s.Name())
		return s.Name(), NOCH, "err no init"
	}
	if !s.moduleConfig.Enabled {
		return s.Name(), NOCH, "disabled"
	}
	settings := &structs.AudioModuleSettings{}
	if err := mapstructure.Decode(s.moduleConfig.Settings, settings); err != nil {
		_ = glg.Errorf("could not convert settings map to %s, module has been disabled: %s", s.Name(), err)
		return s.Name(), NOCH, "err"
	}
	new := files[0]
	duplicate := files[1]
	if new.AudioFormat == media.AUDIO_UNKNOWN {
		return s.Name(), KEEP, "new file audio unknown"
	} else if duplicate.AudioFormat == media.AUDIO_UNKNOWN {
		return s.Name(), KEEP, "old file audio unknown"
	}

	switch settings.Accuracy {
	case consts.AUDIO_ACC_LOW:
		if new.AudioFormat > media.AUDIO_UNKNOWN && duplicate.AudioFormat < media.AUDIO_UNKNOWN {
			return s.Name(), REPL, fmt.Sprintf("new file better: %s vs %s",
				new.AudioFormat.String(), duplicate.AudioFormat.String())
		} else if new.AudioFormat < media.AUDIO_UNKNOWN {
			return s.Name(), KEEP, fmt.Sprintf("old file better %s vs %s",
				new.AudioFormat.String(), duplicate.AudioFormat.String())
		}
	case consts.AUDIO_ACC_MED:
		if new.AudioFormat > media.MULTI_MAYBE && duplicate.AudioFormat < media.STEREO_MAYBE {
			return s.Name(), REPL, fmt.Sprintf("new file better: %s vs %s",
				new.AudioFormat.String(), duplicate.AudioFormat.String())
		} else if new.AudioFormat < media.AUDIO_UNKNOWN {
			return s.Name(), KEEP, fmt.Sprintf("old file better %s vs %s",
				new.AudioFormat.String(), duplicate.AudioFormat.String())
		}
	case consts.AUDIO_ACC_HIGH:
		if new.AudioFormat == media.MULTI && duplicate.AudioFormat == media.STEREO {
			return s.Name(), REPL, fmt.Sprintf("new file better: %s vs %s",
				new.AudioFormat.String(), duplicate.AudioFormat.String())
		} else if new.AudioFormat < media.AUDIO_UNKNOWN {
			return s.Name(), KEEP, fmt.Sprintf("old file better %s vs %s",
				new.AudioFormat.String(), duplicate.AudioFormat.String())
		}
	}
	return s.Name(), NOCH, fmt.Sprintf("no action: %s vs %s",
		new.AudioFormat.String(), duplicate.AudioFormat.String())
}

func (s *AudioModule) Priority() int {
	if s.moduleConfig == nil {
		return -1
	}
	return s.moduleConfig.Priority
}

func (s *AudioModule) Name() string {
	return consts.MODULE_NAME_AUDIO
}
