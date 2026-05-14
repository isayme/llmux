package main

import (
	"fmt"
	"llmux/internal"
	"llmux/internal/config"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	err := config.LoadConfig()
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(-1)
	}

	r := gin.Default()
	internal.SetupRouter(r)

	addr := fmt.Sprintf(":%d", config.Get().Server.Port)

	// compatible with vercel portless
	port := os.Getenv("PORT")
	if port != "" {
		addr = fmt.Sprintf(":%s", port)
	}
	slog.Info("start listening", "addr", addr)
	r.Run(addr)
}
