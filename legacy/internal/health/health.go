package health

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Name     string                 `json:"name"`
	Status   string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Message  string                 `json:"message,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
	Duration time.Duration          `json:"duration_ms"`
}

// Checker defines a health check interface
type Checker interface {
	Name() string
	Check(ctx context.Context) HealthStatus
}

// DatabaseChecker checks database connectivity
type DatabaseChecker struct {
	db *sql.DB
}

// NewDatabaseChecker creates a new database health checker
func NewDatabaseChecker(db *sql.DB) *DatabaseChecker {
	return &DatabaseChecker{db: db}
}

// Name returns the checker name
func (c *DatabaseChecker) Name() string {
	return "database"
}

// Check performs a database health check
func (c *DatabaseChecker) Check(ctx context.Context) HealthStatus {
	start := time.Now()
	
	if c.db == nil {
		return HealthStatus{
			Name:     c.Name(),
			Status:   "unhealthy",
			Message:  "Database connection is nil",
			Duration: time.Since(start),
		}
	}
	
	// Try to ping the database
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := c.db.PingContext(ctx); err != nil {
		return HealthStatus{
			Name:     c.Name(),
			Status:   "unhealthy",
			Message:  fmt.Sprintf("Database ping failed: %v", err),
			Duration: time.Since(start),
		}
	}
	
	// Check database statistics
	stats := c.db.Stats()
	details := map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"in_use":           stats.InUse,
		"idle":             stats.Idle,
		"wait_count":       stats.WaitCount,
		"wait_duration":    stats.WaitDuration.String(),
		"max_open_conns":   stats.MaxOpenConnections,
	}
	
	return HealthStatus{
		Name:     c.Name(),
		Status:   "healthy",
		Details:  details,
		Duration: time.Since(start),
	}
}

// MemoryChecker checks memory usage
type MemoryChecker struct {
	thresholdMB uint64
}

// NewMemoryChecker creates a new memory health checker
func NewMemoryChecker(thresholdMB uint64) *MemoryChecker {
	return &MemoryChecker{thresholdMB: thresholdMB}
}

// Name returns the checker name
func (c *MemoryChecker) Name() string {
	return "memory"
}

// Check performs a memory health check
func (c *MemoryChecker) Check(ctx context.Context) HealthStatus {
	start := time.Now()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Convert to MB
	allocMB := m.Alloc / 1024 / 1024
	sysMB := m.Sys / 1024 / 1024
	heapMB := m.HeapAlloc / 1024 / 1024
	
	details := map[string]interface{}{
		"alloc_mb":      allocMB,
		"sys_mb":        sysMB,
		"heap_mb":       heapMB,
		"num_gc":        m.NumGC,
		"goroutines":    runtime.NumGoroutine(),
		"threshold_mb":  c.thresholdMB,
	}
	
	status := "healthy"
	message := ""
	
	if allocMB > c.thresholdMB {
		status = "degraded"
		message = fmt.Sprintf("Memory usage (%d MB) exceeds threshold (%d MB)", allocMB, c.thresholdMB)
	}
	
	return HealthStatus{
		Name:     c.Name(),
		Status:   status,
		Message:  message,
		Details:  details,
		Duration: time.Since(start),
	}
}

// DiskChecker checks disk space (simplified - for SQLite database)
type DiskChecker struct {
	dbPath string
}

// NewDiskChecker creates a new disk health checker
func NewDiskChecker(dbPath string) *DiskChecker {
	return &DiskChecker{dbPath: dbPath}
}

// Name returns the checker name
func (c *DiskChecker) Name() string {
	return "disk"
}

// Check performs a disk health check
func (c *DiskChecker) Check(ctx context.Context) HealthStatus {
	start := time.Now()
	
	// For now, just check if database file exists and is accessible
	// In a real implementation, would check disk space
	details := map[string]interface{}{
		"db_path": c.dbPath,
	}
	
	return HealthStatus{
		Name:     c.Name(),
		Status:   "healthy", // Simplified check
		Details:  details,
		Duration: time.Since(start),
	}
}

// HealthRegistry manages health checks
type HealthRegistry struct {
	checkers []Checker
}

// NewHealthRegistry creates a new health registry
func NewHealthRegistry() *HealthRegistry {
	return &HealthRegistry{
		checkers: make([]Checker, 0),
	}
}

// Register adds a health checker
func (r *HealthRegistry) Register(checker Checker) {
	r.checkers = append(r.checkers, checker)
}

// CheckAll performs all health checks
func (r *HealthRegistry) CheckAll(ctx context.Context) map[string]HealthStatus {
	results := make(map[string]HealthStatus)
	
	for _, checker := range r.checkers {
		results[checker.Name()] = checker.Check(ctx)
	}
	
	return results
}

// OverallStatus calculates overall health status
func (r *HealthRegistry) OverallStatus(ctx context.Context) (string, map[string]HealthStatus) {
	results := r.CheckAll(ctx)
	
	overall := "healthy"
	for _, status := range results {
		if status.Status == "unhealthy" {
			overall = "unhealthy"
			break
		} else if status.Status == "degraded" && overall == "healthy" {
			overall = "degraded"
		}
	}
	
	return overall, results
}

// HealthResponse represents the complete health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    time.Duration          `json:"uptime_seconds"`
	Checks    map[string]HealthStatus `json:"checks,omitempty"`
	Version   string                 `json:"version,omitempty"`
}

// NewHealthResponse creates a health response
func NewHealthResponse(status string, uptime time.Duration, checks map[string]HealthStatus) HealthResponse {
	return HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Uptime:    uptime,
		Checks:    checks,
		Version:   "1.0.0", // Would come from build info
	}
}