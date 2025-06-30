package config

import (
	"time"
)

// Config holds application configuration
type Config struct {
	Port         string         `yaml:"port"`
	PprofEnabled bool           `yaml:"pprof_enabled"`
	PprofPort    string         `yaml:"pprof_port"`
	LogLevel     string         `yaml:"log_level"`
	LogFormat    string         `yaml:"log_format"`
	ReadTimeout  time.Duration  `yaml:"read_timeout"`
	WriteTimeout time.Duration  `yaml:"write_timeout"`
	Analyzer     AnalyzerConfig `yaml:"analyzer"`
}

// AnalyzerConfig holds analyzer-specific configuration
type AnalyzerConfig struct {
	MaxWorkers     int           `yaml:"max_workers"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
	LinkTimeout    time.Duration `yaml:"link_timeout"`
	MaxRedirects   int           `yaml:"max_redirects"`
}
