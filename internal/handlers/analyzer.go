package handlers

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"web-analyzer/pkg/analyzer"
)

// Analyzer handles analyzer-related HTTP requests
type Analyzer struct {
	analyzer *analyzer.Analyzer
	template *template.Template
	logger   *slog.Logger
}

// NewAnalyzer creates a new analyzer handler
func NewAnalyzer(analyzer *analyzer.Analyzer, logger *slog.Logger) *Analyzer {
	// Load template from file
	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))

	return &Analyzer{
		analyzer: analyzer,
		template: tmpl,
		logger:   logger,
	}
}

// ServeIndex renders the main page
func (h *Analyzer) ServeIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.logger.Debug("404 request", "path", r.URL.Path, "method", r.Method)
		http.NotFound(w, r)
		return
	}

	h.logger.Debug("Serving index page", "remote_addr", r.RemoteAddr)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.template.Execute(w, nil); err != nil {
		h.logger.Error("Template execution failed",
			"error", err,
			"remote_addr", r.RemoteAddr,
		)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("Index page served successfully", "remote_addr", r.RemoteAddr)
}

// ServeAnalyze handles URL analysis requests
func (h *Analyzer) ServeAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("Invalid method for analyze endpoint",
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
		)
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req analyzer.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid JSON payload",
			"error", err,
			"remote_addr", r.RemoteAddr,
		)
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.URL == "" {
		h.logger.Warn("Empty URL in request", "remote_addr", r.RemoteAddr)
		writeErrorResponse(w, http.StatusBadRequest, "URL is required")
		return
	}

	h.logger.Info("Starting URL analysis",
		"url", req.URL,
		"remote_addr", r.RemoteAddr,
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	start := time.Now()

	// Perform analysis
	result, err := h.analyzer.AnalyzeURL(ctx, req.URL)
	if err != nil {
		h.logger.Error("Analysis failed",
			"url", req.URL,
			"error", err,
			"duration", time.Since(start),
			"remote_addr", r.RemoteAddr,
		)

		result = &analyzer.Result{
			URL:   req.URL,
			Error: err.Error(),
		}
	} else {
		h.logger.Info("Analysis completed successfully",
			"url", req.URL,
			"duration", time.Since(start),
			"internal_links", result.InternalLinks,
			"external_links", result.ExternalLinks,
			"inaccessible_links", result.InaccessibleLinks,
			"has_login_form", result.HasLoginForm,
			"remote_addr", r.RemoteAddr,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("Failed to encode response",
			"error", err,
			"url", req.URL,
			"remote_addr", r.RemoteAddr,
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// writeErrorResponse writes an error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
