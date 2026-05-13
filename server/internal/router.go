package internal

import (
	"llmux/internal/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine) {
	r.GET("/version", VersionHandler)

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

	r.Static("/admin/", "./dist")
}
