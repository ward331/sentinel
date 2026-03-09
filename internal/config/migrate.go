package config

import (
	"os"
	"path/filepath"
	"time"
)

// MigrateFromV1 migrates from V1 config system to V2
func MigrateFromV1(oldConfig *ConfigV1) *Config {
	v2Config := DefaultConfig()
	
	// Migrate basic settings
	v2Config.DataDir = GetDefaultDataDir()
	
	// Migrate server settings
	v2Config.Server.Port = 8080 // Default V1 port
	if oldConfig.HTTPPort != "" {
		// Try to parse port from old config
		port := 8080
		// Simple conversion - in real implementation, parse string to int
		// For now, use default
		v2Config.Server.Port = port
	}
	v2Config.Server.Host = oldConfig.HTTPHost
	
	// Migrate database path
	if oldConfig.DBPath != "" && oldConfig.DBPath != "/tmp/sentinel.db" {
		// If custom DB path was set, use its directory as DataDir
		v2Config.DataDir = filepath.Dir(oldConfig.DBPath)
	}
	
	// Migrate poller interval
	if oldConfig.PollerInterval > 0 {
		v2Config.Providers.USGS.IntervalSeconds = int(oldConfig.PollerInterval.Seconds())
		v2Config.Providers.GDACS.IntervalSeconds = int(oldConfig.PollerInterval.Seconds())
		v2Config.Providers.OpenSky.IntervalSeconds = int(oldConfig.PollerInterval.Seconds())
	}
	
	// Migrate backup settings
	if oldConfig.BackupDir != "" && oldConfig.BackupDir != "/tmp/sentinel-backups" {
		// If custom backup dir was set, adjust DataDir
		backupDir := oldConfig.BackupDir
		// Try to extract base data directory from backup path
		if filepath.Base(backupDir) == "backups" {
			v2Config.DataDir = filepath.Dir(backupDir)
		}
	}
	
	// Migrate event log path
	if oldConfig.EventLogPath != "" && oldConfig.EventLogPath != "/tmp/sentinel-events.ndjson" {
		// If custom event log path was set, adjust DataDir
		eventLogPath := oldConfig.EventLogPath
		if filepath.Base(eventLogPath) == "events.ndjson" {
			v2Config.DataDir = filepath.Dir(eventLogPath)
		}
	}
	
	// Set setup as complete since V1 was already running
	v2Config.SetupComplete = true
	
	return v2Config
}

// ConfigV1 represents the old V1 configuration structure
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
	RateLimitEnabled bool
	RateLimitRPS     int
	RateLimitBurst   int
	
	// Logging
	LoggingEnabled bool
	LoggingFormat  string
	LoggingLevel   string
	
	// Alerting
	AlertWebhookTimeout time.Duration
	
	// Backup
	BackupEnabled   bool
	BackupDir       string
	BackupRetention time.Duration
	BackupMaxCount  int
	BackupSchedule  time.Duration
	
	// Data Infrastructure
	EventLogPath string
}

// LoadV1Config loads the old V1 configuration
func LoadV1Config() *ConfigV1 {
	// This would load from environment variables as V1 did
	// For now, return a default V1 config
	return &ConfigV1{
		DBPath:          "/tmp/sentinel.db",
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
		BackupDir:       "/tmp/sentinel-backups",
		BackupRetention: 7 * 24 * time.Hour,
		BackupMaxCount:  10,
		BackupSchedule:  24 * time.Hour,
		EventLogPath:    "/tmp/sentinel-events.ndjson",
	}
}

// AutoMigrate automatically migrates from V1 to V2 if needed
func AutoMigrate() (*Config, error) {
	// Check if V2 config already exists
	v2ConfigPath := GetDefaultConfigPath()
	if _, err := os.Stat(v2ConfigPath); err == nil {
		// V2 config exists, load it
		return LoadConfig(v2ConfigPath)
	}
	
	// V2 config doesn't exist, check if we're running from V1
	// For now, migrate from V1 defaults
	v1Config := LoadV1Config()
	v2Config := MigrateFromV1(v1Config)
	
	// Save the migrated config
	if err := SaveConfig(v2Config, v2ConfigPath); err != nil {
		return nil, err
	}
	
	return v2Config, nil
}