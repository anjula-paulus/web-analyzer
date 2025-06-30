package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"web-analyzer/internal/config"
	"web-analyzer/internal/handlers"
	"web-analyzer/internal/middleware"
)

// New creates a new server instance
func New(cfg *config.Config, analyzerHandler *handlers.Analyzer, healthHandler *handlers.Health, logger *slog.Logger) *Server {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/", analyzerHandler.ServeIndex)
	mux.HandleFunc("/api/v1/analyze", analyzerHandler.ServeAnalyze)
	mux.HandleFunc("/api/v1/health", healthHandler.ServeHealth)
	mux.HandleFunc("/api/v1/health/readiness", healthHandler.ServeReadiness)
	mux.HandleFunc("/api/v1/health/liveness", healthHandler.ServeLiveness)

	// Serve static files if they exist
	if _, err := http.Dir("web/static").Open("/"); err == nil {
		fs := http.FileServer(http.Dir("web/static/"))
		mux.Handle("/static/", http.StripPrefix("/static/", fs))
		logger.Info("Static file serving enabled", "path", "web/static")
	}

	// Apply middleware
	var handler http.Handler = mux
	handler = middleware.Recovery(logger)(handler)
	handler = middleware.CORS(logger)(handler)
	handler = middleware.Logger(logger)(handler)

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
