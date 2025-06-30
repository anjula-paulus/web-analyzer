package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"time"
)

type Health struct {
	startTime time.Time
	logger    *slog.Logger
}

// NewHealth func creates a new health singleton handler
func NewHealth(logger *slog.Logger) *Health {
	return &Health{
		startTime: time.Now(),
		logger:    logger,
	}
}

// ServeHealth returns application health status
func (h *Health) ServeHealth(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Health check requested", "remote_addr", r.RemoteAddr)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(h.startTime)
	goroutines := runtime.NumGoroutine()

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    uptime.String(),
		"version":   "1.0.0",
		"memory": map[string]interface{}{
			"alloc_mb":       bToMb(m.Alloc),
			"total_alloc_mb": bToMb(m.TotalAlloc),
			"sys_mb":         bToMb(m.Sys),
			"num_gc":         m.NumGC,
		},
		"goroutines": goroutines,
	}

	h.logger.Info("Health check completed",
		"uptime", uptime.String(),
		"memory_alloc_mb", bToMb(m.Alloc),
		"goroutines", goroutines,
		"remote_addr", r.RemoteAddr,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
