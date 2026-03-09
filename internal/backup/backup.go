package backup

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BackupManager manages database backups
type BackupManager struct {
	dbPath      string
	backupDir   string
	retention   time.Duration
	maxBackups  int
	enabled     bool
	mu          sync.RWMutex
	lastBackup  time.Time
	backupCount int
}

// BackupConfig holds backup configuration
type BackupConfig struct {
	Enabled    bool
	BackupDir  string
	Retention  time.Duration
	MaxBackups int
	Schedule   time.Duration
}

// DefaultBackupConfig returns default backup configuration
func DefaultBackupConfig() BackupConfig {
	// Use platform-specific default
	dataDir := "/tmp/sentinel-data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, 0755)
	}
	
	return BackupConfig{
		Enabled:    true,
		BackupDir:  filepath.Join(dataDir, "backups"),
		Retention:  7 * 24 * time.Hour, // 7 days
		MaxBackups: 10,
		Schedule:   24 * time.Hour, // Daily
	}
}

// NewBackupManager creates a new backup manager
func NewBackupManager(dbPath string, config BackupConfig) (*BackupManager, error) {
	// Create backup directory if it doesn't exist
	if config.Enabled {
		if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create backup directory: %w", err)
		}
	}
	
	return &BackupManager{
		dbPath:     dbPath,
		backupDir:  config.BackupDir,
		retention:  config.Retention,
		maxBackups: config.MaxBackups,
		enabled:    config.Enabled,
	}, nil
}

// CreateBackup creates a backup of the database
func (bm *BackupManager) CreateBackup(ctx context.Context) error {
	if !bm.enabled {
		return nil // Backups disabled
	}
	
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	// Generate backup filename with timestamp
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05")
	backupPath := filepath.Join(bm.backupDir, fmt.Sprintf("sentinel-%s.db", timestamp))
	
	log.Printf("Creating database backup: %s", backupPath)
	
	// Open source database file
	src, err := os.Open(bm.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database file: %w", err)
	}
	defer src.Close()
	
	// Create backup file
	dst, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()
	
	// Copy database file
	if _, err := io.Copy(dst, src); err != nil {
		// Clean up failed backup
		os.Remove(backupPath)
		return fmt.Errorf("failed to copy database: %w", err)
	}
	
	// Update stats
	bm.lastBackup = time.Now()
	bm.backupCount++
	
	log.Printf("Backup created successfully: %s", backupPath)
	
	// Clean up old backups
	go bm.cleanupOldBackups()
	
	return nil
}

// cleanupOldBackups removes backups older than retention period
func (bm *BackupManager) cleanupOldBackups() {
	if !bm.enabled {
		return
	}
	
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		log.Printf("Failed to read backup directory: %v", err)
		return
	}
	
	var backups []backupInfo
	now := time.Now()
	
	// Collect backup information
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		// Parse timestamp from filename
		// Format: sentinel-2006-01-02T15-04-05.db
		name := info.Name()
		if len(name) < 24 || name[:8] != "sentinel-" || name[len(name)-3:] != ".db" {
			continue // Not a backup file
		}
		
		timestampStr := name[8 : len(name)-3]
		timestamp, err := time.Parse("2006-01-02T15-04-05", timestampStr)
		if err != nil {
			continue // Invalid timestamp
		}
		
		backups = append(backups, backupInfo{
			path:      filepath.Join(bm.backupDir, name),
			timestamp: timestamp,
			size:      info.Size(),
		})
	}
	
	// Sort by timestamp (oldest first)
	// Simple bubble sort for small number of backups
	for i := 0; i < len(backups); i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].timestamp.After(backups[j].timestamp) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}
	
	// Remove backups older than retention period
	removedCount := 0
	for _, backup := range backups {
		if now.Sub(backup.timestamp) > bm.retention {
			if err := os.Remove(backup.path); err != nil {
				log.Printf("Failed to remove old backup %s: %v", backup.path, err)
			} else {
				removedCount++
				log.Printf("Removed old backup: %s", backup.path)
			}
		}
	}
	
	// If still too many backups, remove oldest ones
	if len(backups)-removedCount > bm.maxBackups {
		remaining := len(backups) - removedCount
		toRemove := remaining - bm.maxBackups
		
		for i := 0; i < toRemove && i < len(backups); i++ {
			backup := backups[i]
			if err := os.Remove(backup.path); err != nil {
				log.Printf("Failed to remove excess backup %s: %v", backup.path, err)
			} else {
				log.Printf("Removed excess backup: %s", backup.path)
			}
		}
	}
}

// backupInfo holds information about a backup file
type backupInfo struct {
	path      string
	timestamp time.Time
	size      int64
}

// StartScheduledBackups starts periodic backup scheduling
func (bm *BackupManager) StartScheduledBackups(ctx context.Context, interval time.Duration) {
	if !bm.enabled || interval <= 0 {
		return
	}
	
	log.Printf("Starting scheduled backups every %v", interval)
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// Run initial backup
	if err := bm.CreateBackup(ctx); err != nil {
		log.Printf("Initial backup failed: %v", err)
	}
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping backup scheduler")
			return
		case <-ticker.C:
			if err := bm.CreateBackup(ctx); err != nil {
				log.Printf("Scheduled backup failed: %v", err)
			}
		}
	}
}

// GetBackupStats returns backup statistics
func (bm *BackupManager) GetBackupStats() BackupStats {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	
	return BackupStats{
		Enabled:     bm.enabled,
		LastBackup:  bm.lastBackup,
		BackupCount: bm.backupCount,
		BackupDir:   bm.backupDir,
		Retention:   bm.retention,
		MaxBackups:  bm.maxBackups,
	}
}

// BackupStats holds backup statistics
type BackupStats struct {
	Enabled     bool
	LastBackup  time.Time
	BackupCount int
	BackupDir   string
	Retention   time.Duration
	MaxBackups  int
}

// ListBackups returns a list of available backups
func (bm *BackupManager) ListBackups() ([]BackupInfo, error) {
	if !bm.enabled {
		return nil, nil
	}
	
	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	var backups []BackupInfo
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		// Parse timestamp from filename
		name := info.Name()
		if len(name) < 24 || name[:8] != "sentinel-" || name[len(name)-3:] != ".db" {
			continue
		}
		
		timestampStr := name[8 : len(name)-3]
		timestamp, err := time.Parse("2006-01-02T15-04-05", timestampStr)
		if err != nil {
			continue
		}
		
		backups = append(backups, BackupInfo{
			Filename:  name,
			Path:      filepath.Join(bm.backupDir, name),
			Timestamp: timestamp,
			Size:      info.Size(),
		})
	}
	
	return backups, nil
}

// BackupInfo holds information about a backup
type BackupInfo struct {
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size_bytes"`
}

// RestoreBackup restores a backup to the database
func (bm *BackupManager) RestoreBackup(ctx context.Context, backupPath string) error {
	if !bm.enabled {
		return fmt.Errorf("backups are disabled")
	}
	
	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}
	
	log.Printf("Restoring database from backup: %s", backupPath)
	
	// Create a backup of current database before restore
	tempBackup := fmt.Sprintf("%s.pre-restore-%d.db", bm.dbPath, time.Now().Unix())
	if err := bm.copyFile(bm.dbPath, tempBackup); err != nil {
		log.Printf("Warning: Failed to create pre-restore backup: %v", err)
	}
	
	// Restore from backup
	if err := bm.copyFile(backupPath, bm.dbPath); err != nil {
		// Try to restore from temp backup
		if restoreErr := bm.copyFile(tempBackup, bm.dbPath); restoreErr != nil {
			log.Printf("Critical: Failed to restore original database: %v", restoreErr)
		}
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	
	// Clean up temp backup
	os.Remove(tempBackup)
	
	log.Printf("Database restored successfully from: %s", backupPath)
	return nil
}

// copyFile copies a file from src to dst
func (bm *BackupManager) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	
	_, err = io.Copy(destination, source)
	return err
}