package handler

import (
	"llmux/internal/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

type listProviderResp struct {
	Providers []*config.ProviderConfig `json:"providers"`
}

func ListProviders(c *gin.Context) {
	providers := make([]*config.ProviderConfig, 0)
	for _, provider := range config.Get().Providers {
		providers = append(providers, provider)
	}
	resp := listProviderResp{
		Providers: providers,
	}
	c.JSON(http.StatusOK, resp)
}

type listAPIKeyResp struct {
	APIKeys []*config.ApiKeyConfig `json:"api_keys"`
}

func ListAPIKeys(c *gin.Context) {
	resp := listAPIKeyResp{
		APIKeys: config.Get().APIKeys,
	}
	c.JSON(http.StatusOK, resp)
}

type listAliasResp struct {
	Aliases map[string]*config.ModelAlias `json:"aliases"`
}

func ListAliases(c *gin.Context) {
	aliases := make(map[string]*config.ModelAlias)
	for model, alias := range config.Get().Aliases {
		aliases[model] = alias
	}
	resp := listAliasResp{
		Aliases: aliases,
	}
	c.JSON(http.StatusOK, resp)
}
