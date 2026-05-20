package strategy

import (
	"llmux/internal/config"
	"sync"
)

type fallbackSelector struct {
	mu     sync.Mutex
	models []*config.ModelAliasItemConfig
	index  int
}

func newFallbackSelector(models []*config.ModelAliasItemConfig) *fallbackSelector {
	return &fallbackSelector{
		models: models,
		index:  0,
	}
}

func (s *fallbackSelector) Next() (string, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.models[s.index]
	s.index++
	return m.Provider, m.Model, s.index < len(s.models)
}
