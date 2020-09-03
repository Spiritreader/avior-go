package comparator

import "github.com/Spiritreader/avior-go/consts"

type SizeApproxModule struct {
	enabled bool
}

func (s *SizeApproxModule) Run() string {
	if !s.enabled {
		return NOCH
	}
	return "not_implemented"
}

func (s *SizeApproxModule) Priority() int {
	return -1
}

func (s *SizeApproxModule) Name() string {
	return consts.MODULE_NAME_SIZEAPPROX
}
