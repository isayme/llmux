package main

import (
	"fmt"
	"llmux/internal"
	"llmux/internal/config"
	"log/slog"

	"github.com/gin-gonic/gin"
)

func main() {
	err := config.LoadConfig()
	if err != nil {
		slog.Error("load config failed", "err", err)
		return
	}

	r := gin.Default()
	internal.SetupRouter(r)

	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	slog.Info("start listening", "addr", addr)
	r.Run(addr)
}
