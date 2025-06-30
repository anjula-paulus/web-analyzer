package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"web-analyzer/internal/config"
	"web-analyzer/internal/handlers"
	"web-analyzer/internal/middleware"
)

// New func creates a new server singleton instance
func New(cfg *config.Config, analyzerHandler *handlers.Analyzer, healthHandler *handlers.Health, logger *slog.Logger) *Server {
	r := http.NewServeMux()

	// Register routes
	r.HandleFunc("/", analyzerHandler.ServeIndex)
	r.HandleFunc("/api/v1/analyze", analyzerHandler.ServeAnalyze)
	r.HandleFunc("/api/v1/health", healthHandler.ServeHealth)
	r.Handle("/metrics", promhttp.Handler())

	// Serve static files if they exist
	if _, err := http.Dir("web/static").Open("/"); err == nil {
		fs := http.FileServer(http.Dir("web/static/"))
		r.Handle("/static/", http.StripPrefix("/static/", fs))
		logger.Info("Static file serving enabled", "path", "web/static")
	}

	// Apply middleware
	var handler http.Handler = r
	handler = middleware.NewRecoveryMiddleware(logger)(handler)
	handler = middleware.NewCORSMiddleware(logger)(handler)
	handler = middleware.NewLoggerMiddleware(logger)(handler)
	handler = middleware.NewMetricsMiddleware(logger)(handler)

	logger.Info("Server configured",
		"port", cfg.Port,
		"read_timeout", cfg.ReadTimeout,
		"write_timeout", cfg.WriteTimeout,
	)

	return &Server{
		config: cfg,
		logger: logger,
		httpServer: &http.Server{
			Addr:         cfg.Port,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  60 * time.Second,
			ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("HTTP server starting", "addr", s.config.Port)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Starting graceful shutdown")

	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.logger.Error("Server shutdown failed", "error", err)
		return err
	}

	s.logger.Info("Server shutdown completed")
	return nil
}
