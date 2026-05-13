package handler

import (
	"llmux/internal/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

type listProviderResp struct {
	Providers []config.ProviderConfig `json:"providers"`
}

func ListProviders(c *gin.Context) {
	providers := make([]config.ProviderConfig, 0)
	for _, provider := range config.GlobalConfig.Providers {
		providers = append(providers, provider)
	}
	resp := listProviderResp{
		Providers: providers,
	}
	c.JSON(http.StatusOK, resp)
}

func ListAPIKeys(c *gin.Context) {
	c.JSON(http.StatusOK, config.GlobalConfig.APIKeys)
}

type listAliasResp struct {
	Aliases map[string]config.ModelAlias `json:"aliases"`
}

func ListAliases(c *gin.Context) {
	aliases := make(map[string]config.ModelAlias)
	for model, alias := range config.GlobalConfig.Aliases {
		aliases[model] = alias
	}
	resp := listAliasResp{
		Aliases: aliases,
	}
	c.JSON(http.StatusOK, resp)
}
