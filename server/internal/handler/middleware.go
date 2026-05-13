package handler

import (
	"llmux/internal/config"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
)

func APIKeyValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(authorizationHeader)
		if authHeader == "" {
			c.Error(Unauthorized.WithMessage("authorization header is required"))
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.Error(Unauthorized.WithMessage("invalid authorization header format"))
			c.Abort()
			return
		}

		apiKey := strings.TrimPrefix(authHeader, bearerPrefix)
		apiKeyConfig := findApiKeyConfig(apiKey)
		if apiKeyConfig == nil {
			c.Error(Unauthorized.WithMessage("api key not exists"))
			c.Abort()
			return
		}

		if !apiKeyConfig.Enabled {
			c.Error(Unauthorized.WithMessage("api key is disabled"))
			c.Abort()
			return
		}

		c.Next()
	}
}

func findApiKeyConfig(apiKey string) *config.ApiKeyConfig {
	for _, apiKeyConfig := range config.GlobalConfig.ApiKeys {
		if apiKeyConfig.Key == apiKey {
			return &apiKeyConfig
		}
	}

	return nil
}
