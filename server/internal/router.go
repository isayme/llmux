package internal

import (
	"llmux/internal/handler"
	"llmux/internal/log"
	"llmux/internal/trace"
	"net/http"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine, logService *log.Service) {
	proxyHandler := handler.NewProxyHandler(logService)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/admin")
	})

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

	// logs
	{
		g := r.Group("/api/logs")
		g.Use(handler.InternalErrorHandler())
		g.Use(handler.SessionMiddleware())
		g.Use(handler.SessionValidationMiddleware())

		logHandler := log.NewHandler(logService)

		g.GET("/requests", logHandler.ListRequestLogs)
		g.GET("/requests/:id", logHandler.GetRequestLog)
		g.GET("/requests/:id/calls", logHandler.GetProviderCalls)
		g.DELETE("/requests", logHandler.DeleteLogs)
	}

	// openai
	{
		g := r.Group("/v1")

		g.Use(handler.OpenaiErrorHandler())
		g.Use(handler.APIKeyValidationMiddleware())
		g.Use(trace.Middleware())

		g.POST("/chat/completions", proxyHandler.ChatCompletionsHandler)
		g.GET("/models", proxyHandler.ListModelsHandler)
	}

	// openai responses
	{
		g := r.Group("/v1")
		g.Use(handler.ResponsesErrorHandler())
		g.Use(handler.APIKeyValidationMiddleware())
		g.Use(trace.Middleware())

		g.POST("/responses", proxyHandler.ResponsesHandler)
	}

	// anthropic
	{
		g := r.Group("/anthropic")
		g.Use(handler.AnthropicErrorHandler())
		g.Use(handler.APIKeyValidationMiddleware())
		g.Use(trace.Middleware())
		g.POST("/v1/messages", proxyHandler.AnthropicMessagesHandler)
	}

	// r.Static("/admin/", "./dist")
	r.Use(static.ServeRoot("/admin/", "./dist"))
	r.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})
}
