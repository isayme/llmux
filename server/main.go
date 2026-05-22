package main

import (
	"context"
	"fmt"
	"llmux/internal"
	"llmux/internal/config"
	"llmux/internal/trace"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	err := config.LoadConfig()
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(-1)
	}

	// Initialize tracing
	cfg := config.Get().Trace
	tp, err := trace.Init(cfg.Enabled, cfg.Exporter, cfg.Endpoint, cfg.Sampling.Ratio)
	if err != nil {
		slog.Error("trace init failed", "err", err)
		os.Exit(-1)
	}
	if tp != nil {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			trace.Shutdown(ctx, tp)
		}()
	}

	// Set sampling config for middleware
	forceLatency, _ := time.ParseDuration(cfg.Sampling.ForceOnLatency)
	if forceLatency == 0 {
		forceLatency = 5 * time.Second
	}
	trace.SetSamplingConfig(trace.MiddlewareSamplingConfig{
		Ratio:          cfg.Sampling.Ratio,
		ForceOnError:   cfg.Sampling.ForceOnError,
		ForceOnLatency: forceLatency,
	})

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
