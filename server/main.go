package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"llmux/internal"
	"llmux/internal/config"
	"llmux/internal/log"
	"llmux/internal/trace"

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

	// Initialize logging
	loggingCfg := config.Get().Logging
	logStore, err := log.NewStore(loggingCfg.DBPath)
	if err != nil {
		slog.Error("Failed to initialize log store", "error", err)
		return
	}
	defer logStore.Close()

	logService := log.NewService(logStore)

	// Mark pending requests as interrupted (from previous server session)
	if count, err := logService.MarkPendingAsInterrupted(); err != nil {
		slog.Error("Failed to mark pending requests as interrupted", "error", err)
	} else if count > 0 {
		slog.Info("Marked pending requests as interrupted", "count", count)
	}

	// Start cleaner if logging is enabled
	if loggingCfg.Enabled {
		cleanInterval, _ := time.ParseDuration(loggingCfg.CleanInterval)
		cleaner := log.NewCleaner(logService, loggingCfg.RetentionDays)
		cleaner.Start(cleanInterval)
		defer cleaner.Stop()
	}

	r := gin.Default()
	internal.SetupRouter(r, logService)

	addr := fmt.Sprintf(":%d", config.Get().Server.Port)

	// compatible with vercel portless
	port := os.Getenv("PORT")
	if port != "" {
		addr = fmt.Sprintf(":%s", port)
	}
	slog.Info("start listening", "addr", addr)
	r.Run(addr)
}
