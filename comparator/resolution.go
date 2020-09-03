package comparator

import "github.com/Spiritreader/avior-go/consts"

type ResolutionModule struct {
	enabled bool
}

func (s *ResolutionModule) Run() string {
	if !s.enabled {
		return NOCH
	}
	return "not_implemented"
}

func (s *ResolutionModule) Priority() int {
	return -1
}

func (s *ResolutionModule) Name() string {
	return consts.MODULE_NAME_RESOLUTION
}
