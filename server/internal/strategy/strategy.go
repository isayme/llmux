package strategy

import "llmux/internal/config"

type ModelSelector interface {
	Next() (provider, model string, retryable bool)
}

func NewSelector(strategy string, models []*config.ModelAliasItemConfig) ModelSelector {
	switch strategy {
	case "random":
		return newRandomSelector(models)
	case "round_robin":
		return newRoundRobin(models)
	case "fallback":
		return newFallbackSelector(models)
	default:
		return newRoundRobin(models)
	}
}
