// pkg/telemetry/config/config.go
// Centralized configuration management for all telemetry settings
package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// TelemetryConfig holds all telemetry configuration
type TelemetryConfig struct {
	// Core settings
	Enabled          bool
	AsyncProcessing  bool
	ProcessingTimeout time.Duration
	
	// Metric settings
	MaxCardinality   int
	CleanupInterval  time.Duration
	MaxEntryAge      time.Duration
	MaxEntries       int
	
	// Event processing
	EventBufferSize  int
	BatchSize        int
	FlushInterval    time.Duration
}

var (
	config     *TelemetryConfig
	configOnce sync.Once
)

// Initialize sets up the telemetry configuration from environment variables
func Initialize() {
	configOnce.Do(func() {
		config = &TelemetryConfig{
			// Defaults
			Enabled:           true,
			AsyncProcessing:   true,
			ProcessingTimeout: 5 * time.Second,
			MaxCardinality:    30, // Red Hat limit
			CleanupInterval:   1 * time.Hour,
			MaxEntryAge:       24 * time.Hour,
			MaxEntries:        10000,
			EventBufferSize:   1000,
			BatchSize:         100,
			FlushInterval:     30 * time.Second,
		}
		
		loadFromEnvironment()
		validateConfig()
		logConfig()
	})
}

// loadFromEnvironment loads configuration from environment variables
func loadFromEnvironment() {
	// Core settings
	if enabled := os.Getenv("TELEMETRY_ENABLED"); enabled != "" {
		config.Enabled = strings.ToLower(enabled) != "false"
	}
	
	if async := os.Getenv("TELEMETRY_ASYNC"); async != "" {
		config.AsyncProcessing = strings.ToLower(async) != "false"
	}
	
	if timeout := os.Getenv("TELEMETRY_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.ProcessingTimeout = d
		}
	}
	
	// Metric settings
	if cardinality := os.Getenv("TELEMETRY_MAX_CARDINALITY"); cardinality != "" {
		if c, err := strconv.Atoi(cardinality); err == nil && c > 0 {
			config.MaxCardinality = c
		}
	}
	
	if cleanup := os.Getenv("TELEMETRY_CLEANUP_INTERVAL"); cleanup != "" {
		if d, err := time.ParseDuration(cleanup); err == nil {
			config.CleanupInterval = d
		}
	}
	
	if maxAge := os.Getenv("TELEMETRY_MAX_ENTRY_AGE"); maxAge != "" {
		if d, err := time.ParseDuration(maxAge); err == nil {
			config.MaxEntryAge = d
		}
	}
	
	if maxEntries := os.Getenv("TELEMETRY_MAX_ENTRIES"); maxEntries != "" {
		if e, err := strconv.Atoi(maxEntries); err == nil && e > 0 {
			config.MaxEntries = e
		}
	}
}

// validateConfig ensures configuration values are reasonable
func validateConfig() {
	// Ensure reasonable timeouts
	if config.ProcessingTimeout < 1*time.Second {
		config.ProcessingTimeout = 1 * time.Second
	}
	if config.ProcessingTimeout > 30*time.Second {
		config.ProcessingTimeout = 30 * time.Second
	}
	
	// Ensure reasonable cardinality limits
	if config.MaxCardinality < 1 {
		config.MaxCardinality = 1
	}
	if config.MaxCardinality > 100 {
		klog.Warningf("High cardinality limit set: %d (Red Hat recommends max 30)", config.MaxCardinality)
	}
	
	// Ensure reasonable entry limits
	if config.MaxEntries < 100 {
		config.MaxEntries = 100
	}
	if config.MaxEntries > 100000 {
		config.MaxEntries = 100000
	}
}

// logConfig logs the current configuration (without sensitive data)
func logConfig() {
	klog.Infof("Telemetry configuration: enabled=%v, async=%v, timeout=%v, max_cardinality=%d", 
		config.Enabled, config.AsyncProcessing, config.ProcessingTimeout, config.MaxCardinality)
}

// Get returns the current telemetry configuration
func Get() *TelemetryConfig {
	if config == nil {
		Initialize()
	}
	return config
}

// IsEnabled returns whether telemetry is enabled
func IsEnabled() bool {
	return Get().Enabled
}

// GetMaxCardinality returns the maximum allowed cardinality
func GetMaxCardinality() int {
	return Get().MaxCardinality
}

// GetProcessingTimeout returns the event processing timeout
func GetProcessingTimeout() time.Duration {
	return Get().ProcessingTimeout
}

// GetCleanupInterval returns the cleanup interval for old entries
func GetCleanupInterval() time.Duration {
	return Get().CleanupInterval
}

// GetMaxEntryAge returns the maximum age for cached entries
func GetMaxEntryAge() time.Duration {
	return Get().MaxEntryAge
}

// GetMaxEntries returns the maximum number of entries to cache
func GetMaxEntries() int {
	return Get().MaxEntries
}