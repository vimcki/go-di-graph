package build

import (
	"example.com/project/internal/configuration"
	"example.com/project/internal/httpapi/common"
	"example.com/project/internal/httpapi/processor"
)

type Set struct {
	commonComponent common.Component
	Processor       processor.Processor
}

func (s *Set) Build(cfg configuration.Configuration) error {
	if s.commonComponent == nil {
		s.fillCommonComponent(cfg)
	}

	s.Processor = processor.New(s.commonComponent)

	return nil
}

func (s *Set) fillCommonComponent(cfg configuration.Configuration) {
	s.commonComponent = common.NewComponent(cfg.Addr)
}
