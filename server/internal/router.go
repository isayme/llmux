package internal

import (
	"llmux/internal/handler"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine) {
	r.GET("/version", VersionHandler)

	{
		r.POST("/api/login", handler.InternalErrorHandler(), handler.SessionMiddleware(), handler.LoginHandler)
		r.GET("/api/check", handler.InternalErrorHandler(), handler.SessionMiddleware(), handler.CheckSessionHandler)

		g := r.Group("/api")
		g.Use(handler.InternalErrorHandler())
		g.Use(handler.SessionMiddleware())
		g.Use(handler.SessionValidationMiddleware())

		g.POST("/logout", handler.LogoutHandler)

		g.GET("/providers", handler.ListProviders)
		g.GET("/api-keys", handler.ListAPIKeys)
		g.POST("/aliases", handler.ListAliases)
	}

	// openai
	{
		g := r.Group("/v1")

		g.Use(handler.OpenaiErrorHandler())
		g.Use(handler.APIKeyValidationMiddleware())

		g.POST("/chat/completions", handler.ChatCompletionsHandler)
		g.GET("/models", handler.ListModelsHandler)

		compatibleGroup := g.Group("/v1")
		compatibleGroup.POST("/chat/completions", handler.ChatCompletionsHandler)
		compatibleGroup.GET("/models", handler.ListModelsHandler)
	}

	// anthropic
	{
		g := r.Group("/anthropic")
		g.Use(handler.AnthropicErrorHandler())
		g.Use(handler.APIKeyValidationMiddleware())
		g.POST("/v1/messages", handler.AnthropicMessagesHandler)
	}

	// r.Static("/admin/", "./dist")
	r.Use(static.ServeRoot("/admin/", "./dist"))
	r.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})
}
