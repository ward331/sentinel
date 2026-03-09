package config

import (
	"os"
	"path/filepath"
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



// LoadV1Config loads the old V1 configuration
func LoadV1Config() *ConfigV1 {
	// Use the actual V1 config loader
	return LoadConfigV1()
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