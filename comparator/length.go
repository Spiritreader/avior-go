package comparator

import "github.com/Spiritreader/avior-go/consts"

type LengthModule struct {
	enabled bool
}

func (s *LengthModule) Run() string {
	if !s.enabled {
		return NOCH
	}
	return "not_implemented"
}

func (s *LengthModule) Priority() int {
	return -1
}

func (s *LengthModule) Name() string {
	return consts.MODULE_NAME_LENGTH
}
