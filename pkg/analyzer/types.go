package analyzer

import (
	"log/slog"
	"net/http"
	"web-analyzer/internal/config"
)

// Analyzer provides web page analysis functionality
type Analyzer struct {
	client *http.Client
	config config.AnalyzerConfig
	logger *slog.Logger
}

// Result represents the analysis result
type Result struct {
	URL               string         `json:"url"`
	HTMLVersion       string         `json:"html_version"`
	Title             string         `json:"title"`
	Headings          map[string]int `json:"headings"`
	InternalLinks     int            `json:"internal_links"`
	ExternalLinks     int            `json:"external_links"`
	InaccessibleLinks int            `json:"inaccessible_links"`
	HasLoginForm      bool           `json:"has_login_form"`
	Error             string         `json:"error,omitempty"`
}

// Request represents the analysis request
type Request struct {
	URL string `json:"url"`
}
