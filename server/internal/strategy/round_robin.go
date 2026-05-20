package strategy

import (
	"llmux/internal/config"
	"sync/atomic"
)

type roundRobin struct {
	models []*config.ModelAliasItemConfig
	idx    atomic.Int32
}

func newRoundRobin(models []*config.ModelAliasItemConfig) *roundRobin {
	rb := &roundRobin{
		models: models,
	}
	rb.idx.Store(-1)

	return rb
}

func (rb *roundRobin) Next() (string, string, bool) {
	newIdx := rb.idx.Add(1)
	item := rb.models[newIdx%int32(len(rb.models))]
	return item.Provider, item.Model, false
}
