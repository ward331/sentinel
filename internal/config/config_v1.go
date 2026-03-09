package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ConfigV1 holds V1 application configuration
type ConfigV1 struct {
	// Database
	DBPath          string
	MaxConnections  int
	ConnectionPool  bool
	
	// HTTP Server
	HTTPHost        string
	HTTPPort        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	
	// Poller
	PollerInterval  time.Duration
	
	// Rate Limiting
	RateLimitEnabled bool `env:"SENTINEL_RATE_LIMIT_ENABLED" default:"false"`
	RateLimitRPS     int  `env:"SENTINEL_RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst   int  `env:"SENTINEL_RATE_LIMIT_BURST" default:"200"`
	
	// Logging
	LoggingEnabled bool   `env:"SENTINEL_LOGGING_ENABLED" default:"true"`
	LoggingFormat  string `env:"SENTINEL_LOGGING_FORMAT" default:"text"`
	LoggingLevel   string `env:"SENTINEL_LOGGING_LEVEL" default:"info"`
	
	// Alerting
	AlertWebhookTimeout time.Duration `env:"SENTINEL_ALERT_WEBHOOK_TIMEOUT" default:"10s"`
	
	// Backup
	BackupEnabled   bool          `env:"SENTINEL_BACKUP_ENABLED" default:"true"`
	BackupDir       string        `env:"SENTINEL_BACKUP_DIR" default:""` // Will be set in DefaultConfig
	BackupRetention time.Duration `env:"SENTINEL_BACKUP_RETENTION" default:"168h"` // 7 days
	BackupMaxCount  int           `env:"SENTINEL_BACKUP_MAX_COUNT" default:"10"`
	BackupSchedule  time.Duration `env:"SENTINEL_BACKUP_SCHEDULE" default:"24h"` // Daily
	
	// Data Infrastructure
	EventLogPath string `env:"SENTINEL_EVENT_LOG_PATH" default:""` // Will be set in DefaultConfig
}

// getDefaultDataDir returns platform-specific default data directory
func getDefaultDataDir() string {
	// For V1 compatibility, use /tmp on Unix-like systems
	// In production, this would use platform-specific directories
	return "/tmp/sentinel-data"
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	dataDir := getDefaultDataDir()
	
	return &Config{
		DBPath:          filepath.Join(dataDir, "sentinel.db"),
		MaxConnections:  5,
		ConnectionPool:  true,
		HTTPHost:        "0.0.0.0",
		HTTPPort:        "8080",
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		PollerInterval:  60 * time.Second,
		RateLimitEnabled: false,
		RateLimitRPS:    100,
		RateLimitBurst:  200,
		LoggingEnabled:  true,
		LoggingFormat:   "text",
		LoggingLevel:    "info",
		AlertWebhookTimeout: 10 * time.Second,
		BackupEnabled:   true,
		BackupDir:       filepath.Join(dataDir, "backups"),
		BackupRetention: 7 * 24 * time.Hour,
		BackupMaxCount:  10,
		BackupSchedule:  24 * time.Hour,
		EventLogPath:    filepath.Join(dataDir, "events.ndjson"),
	}
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() {
	// Database
	if val := os.Getenv("SENTINEL_DB_PATH"); val != "" {
		c.DBPath = val
	}
	if val := os.Getenv("SENTINEL_MAX_CONNECTIONS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.MaxConnections = n
		}
	}
	if val := os.Getenv("SENTINEL_CONNECTION_POOL"); val != "" {
		c.ConnectionPool = val == "true" || val == "1"
	}
	
	// HTTP Server
	if val := os.Getenv("SENTINEL_HTTP_HOST"); val != "" {
		c.HTTPHost = val
	}
	if val := os.Getenv("SENTINEL_HTTP_PORT"); val != "" {
		c.HTTPPort = val
	}
	if val := os.Getenv("SENTINEL_READ_TIMEOUT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.ReadTimeout = time.Duration(n) * time.Second
		}
	}
	if val := os.Getenv("SENTINEL_WRITE_TIMEOUT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.WriteTimeout = time.Duration(n) * time.Second
		}
	}
	if val := os.Getenv("SENTINEL_IDLE_TIMEOUT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.IdleTimeout = time.Duration(n) * time.Second
		}
	}
	
	// Poller
	if val := os.Getenv("SENTINEL_POLLER_INTERVAL"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.PollerInterval = time.Duration(n) * time.Second
		}
	}
	
	// Rate Limiting
	if val := os.Getenv("SENTINEL_RATE_LIMIT_ENABLED"); val != "" {
		c.RateLimitEnabled = val == "true" || val == "1"
	}
	if val := os.Getenv("SENTINEL_RATE_LIMIT_RPS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.RateLimitRPS = n
		}
	}
	if val := os.Getenv("SENTINEL_RATE_LIMIT_BURST"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.RateLimitBurst = n
		}
	}
	
	// Logging
	if val := os.Getenv("SENTINEL_LOGGING_ENABLED"); val != "" {
		c.LoggingEnabled = val == "true" || val == "1"
	}
	if val := os.Getenv("SENTINEL_LOGGING_LEVEL"); val != "" {
		c.LoggingLevel = val
	}
	if val := os.Getenv("SENTINEL_LOGGING_FORMAT"); val != "" {
		c.LoggingFormat = val
	}
	
	// Alerting
	if val := os.Getenv("SENTINEL_ALERT_WEBHOOK_TIMEOUT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.AlertWebhookTimeout = time.Duration(n) * time.Second
		}
	}
	
	// Backup
	if val := os.Getenv("SENTINEL_BACKUP_ENABLED"); val != "" {
		c.BackupEnabled = val == "true" || val == "1"
	}
	if val := os.Getenv("SENTINEL_BACKUP_DIR"); val != "" {
		c.BackupDir = val
	}
	if val := os.Getenv("SENTINEL_BACKUP_RETENTION"); val != "" {
		if d, err := time.ParseDuration(val); err == nil && d > 0 {
			c.BackupRetention = d
		}
	}
	if val := os.Getenv("SENTINEL_BACKUP_MAX_COUNT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			c.BackupMaxCount = n
		}
	}
	if val := os.Getenv("SENTINEL_BACKUP_SCHEDULE"); val != "" {
		if d, err := time.ParseDuration(val); err == nil && d > 0 {
			c.BackupSchedule = d
		}
	}
	
	// Data Infrastructure
	if val := os.Getenv("SENTINEL_EVENT_LOG_PATH"); val != "" {
		c.EventLogPath = val
	}
}

// LoadConfig loads configuration with defaults and environment overrides
func LoadConfig() *Config {
	config := DefaultConfig()
	config.LoadFromEnv()
	return config
}