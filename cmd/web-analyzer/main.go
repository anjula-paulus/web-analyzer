package main

import (
	"context"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"web-analyzer/internal/config"
	"web-analyzer/internal/handlers"
	"web-analyzer/internal/server"
	"web-analyzer/pkg/analyzer"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup structured logging
	logger := setupLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	logger.Info("Starting web analyzer",
		"port", cfg.Port,
		"pprof_enabled", cfg.PprofEnabled,
		"log_level", cfg.LogLevel,
		"max_workers", cfg.Analyzer.MaxWorkers,
	)

	// Create analyzer service
	analyzerService := analyzer.New(cfg.Analyzer, logger)

	// Create handlers with logger
	analyzerHandler := handlers.NewAnalyzer(analyzerService, logger)
	healthHandler := handlers.NewHealth(logger)

	// Start pprof server if enabled
	if cfg.PprofEnabled {
		go func() {
			logger.Info("Starting pprof server", "port", cfg.PprofPort)
			if err := http.ListenAndServe(cfg.PprofPort, nil); err != nil {
				logger.Error("pprof server failed", "error", err)
			}
		}()
	}

	// Create and start server
	srv := server.New(cfg, analyzerHandler, healthHandler, logger)

	// Start server in goroutine
	go func() {
		logger.Info("Server starting", "addr", cfg.Port)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Received shutdown signal", "signal", sig.String())

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Server shutdown completed successfully")
}

// setupLogger configures structured logging based on configuration
func setupLogger(level, format string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: logLevel == slog.LevelDebug,
	}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
