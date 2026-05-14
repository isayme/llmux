package handler

import (
	"llmux/internal/config"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

const (
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
	sessionCookieName   = "llmux_session"
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
	for _, apiKeyConfig := range config.Get().APIKeys {
		if apiKeyConfig.Key == apiKey {
			return apiKeyConfig
		}
	}

	return nil
}

func SessionMiddleware() gin.HandlerFunc {
	store := cookie.NewStore([]byte(config.Get().Server.Session.SecretKey))
	return sessions.Sessions(config.Get().Server.Session.CookieName, store)
}

func SessionValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		authed := session.Get("authed")
		if authed == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
			return
		}

		c.Next()
	}
}
