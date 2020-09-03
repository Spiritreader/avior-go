package comparator

import "github.com/Spiritreader/avior-go/consts"

type MaxSizeModule struct {
	enabled bool
}

func (s *MaxSizeModule) Run() string {
	if !s.enabled {
		return NOCH
	}
	return "not_implemented"
}

func (s *MaxSizeModule) Priority() int {
	return -1
}

func (s *MaxSizeModule) Name() string {
	return consts.MODULE_NAME_MAXSIZE
}
