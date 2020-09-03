package comparator

import "github.com/Spiritreader/avior-go/consts"

type LogMatchModule struct {
	enabled bool
}

func (s *LogMatchModule) Run() string {
	if !s.enabled {
		return NOCH
	}
	return "not_implemented"
}

func (s *LogMatchModule) Priority() int {
	return -1
}

func (s *LogMatchModule) Name() string {
	return consts.MODULE_NAME_LOGMATCH
}
