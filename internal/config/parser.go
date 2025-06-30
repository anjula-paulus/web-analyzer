package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Load loads configuration from YAML file and environment variables
func Load() (*Config, error) {
	// Default configuration
	config := &Config{
		Port:         ":8080",
		PprofEnabled: true,
		PprofPort:    "localhost:6060",
		LogLevel:     "info",
		LogFormat:    "json",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Analyzer: AnalyzerConfig{
			MaxWorkers:     10,
			RequestTimeout: 30 * time.Second,
			LinkTimeout:    10 * time.Second,
			MaxRedirects:   5,
		},
	}

	// Try to load from YAML file
	if err := loadFromYAML(config); err != nil {
		// Continue with defaults if YAML loading fails
	}

	// Override with environment variables
	overrideWithEnv(config)

	return config, nil
}

// loadFromYAML loads configuration from YAML file
func loadFromYAML(config *Config) error {
	configPaths := []string{
		"config.yaml",
		"configs/config.yaml",
	}

	if customPath := os.Getenv("CONFIG_PATH"); customPath != "" {
		configPaths = append([]string{customPath}, configPaths...)
	}

	var configData []byte
	var err error

	for _, path := range configPaths {
		if configData, err = os.ReadFile(path); err == nil {
			break
		}
	}

	if err != nil {
		return err
	}

	return yaml.Unmarshal(configData, config)
}

// overrideWithEnv overrides configuration with environment variables
func overrideWithEnv(config *Config) {
	if port := os.Getenv("PORT"); port != "" {
		config.Port = port
	}

	if pprofEnabled := os.Getenv("PPROF_ENABLED"); pprofEnabled != "" {
		config.PprofEnabled = pprofEnabled == "true"
	}

	if pprofPort := os.Getenv("PPROF_PORT"); pprofPort != "" {
		config.PprofPort = pprofPort
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		config.LogFormat = logFormat
	}

	if maxWorkers := os.Getenv("MAX_WORKERS"); maxWorkers != "" {
		if workers, err := strconv.Atoi(maxWorkers); err == nil {
			config.Analyzer.MaxWorkers = workers
		}
	}

	if requestTimeout := os.Getenv("REQUEST_TIMEOUT"); requestTimeout != "" {
		if timeout, err := time.ParseDuration(requestTimeout); err == nil {
			config.Analyzer.RequestTimeout = timeout
		}
	}

	if linkTimeout := os.Getenv("LINK_TIMEOUT"); linkTimeout != "" {
		if timeout, err := time.ParseDuration(linkTimeout); err == nil {
			config.Analyzer.LinkTimeout = timeout
		}
	}

	if maxRedirects := os.Getenv("MAX_REDIRECTS"); maxRedirects != "" {
		if redirects, err := strconv.Atoi(maxRedirects); err == nil {
			config.Analyzer.MaxRedirects = redirects
		}
	}
}
