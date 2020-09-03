package comparator

import "github.com/Spiritreader/avior-go/consts"

type AudioModule struct {
	enabled bool
}

func (s *AudioModule) Run() string {
	if !s.enabled {
		return NOCH
	}
	return "not_implemented"
}

func (s *AudioModule) Priority() int {
	return -1
}

func (s *AudioModule) Name() string {
	return consts.MODULE_NAME_AUDIO
}
