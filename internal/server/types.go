package server

import (
	"log/slog"
	"net/http"
	"web-analyzer/internal/config"
)

// Server wraps the HTTP server
type Server struct {
	httpServer *http.Server
	config     *config.Config
	logger     *slog.Logger
}
