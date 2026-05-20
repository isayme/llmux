package strategy

import (
	"llmux/internal/config"
	"math/rand"
	"time"
)

type randomSelector struct {
	models []*config.ModelAliasItemConfig
	rand   *rand.Rand
}

func newRandomSelector(models []*config.ModelAliasItemConfig) *randomSelector {
	randSource := rand.NewSource(time.Now().UnixNano())

	return &randomSelector{
		models: models,
		rand:   rand.New(randSource),
	}
}

func (s *randomSelector) Next() (string, string, bool) {
	item := s.models[s.rand.Intn(len(s.models))]
	return item.Provider, item.Model, false
}
