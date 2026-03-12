package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"github.com/openclaw/sentinel-backend/internal/model"
)

// ConnectionPool manages a pool of database connections
type ConnectionPool struct {
	mu          sync.RWMutex
	connections []*sql.DB
	dbPath      string
	maxSize     int
	inUse       map[*sql.DB]bool
	waitQueue   chan struct{}
	stats       PoolStats
}

// PoolStats tracks connection pool statistics
type PoolStats struct {
	TotalConnections int
	ActiveConnections int
	MaxConnections   int
	WaitCount        int64
	AcquireTime      time.Duration
	ReleaseTime      time.Duration
	Errors           int64
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(dbPath string, maxConnections int) (*ConnectionPool, error) {
	if maxConnections < 1 {
		maxConnections = 5 // Default pool size
	}
	
	pool := &ConnectionPool{
		dbPath:    dbPath,
		maxSize:   maxConnections,
		inUse:     make(map[*sql.DB]bool),
		waitQueue: make(chan struct{}, maxConnections*10), // Buffer for waiters
		stats: PoolStats{
			MaxConnections: maxConnections,
		},
	}
	
	// Create initial connection
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	// Configure connection
	conn.SetMaxOpenConns(1) // SQLite limitation
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(30 * time.Minute)
	
	pool.connections = append(pool.connections, conn)
	pool.stats.TotalConnections = 1
	
	// Initialize database schema
	if err := pool.initializeSchema(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	
	return pool, nil
}

// initializeSchema sets up the database schema on a connection
func (p *ConnectionPool) initializeSchema(conn *sql.DB) error {
	// Execute schema creation
	_, err := conn.Exec(defaultSchema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	
	// Enable WAL mode if supported (commented out due to modernc limitations)
	// _, err = conn.Exec("PRAGMA journal_mode = WAL")
	// if err != nil {
	// 	log.Printf("Warning: Could not enable WAL mode: %v", err)
	// }
	
	// Enable foreign keys
	_, err = conn.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Printf("Warning: Could not enable foreign keys: %v", err)
	}
	
	// Set synchronous mode
	_, err = conn.Exec("PRAGMA synchronous = NORMAL")
	if err != nil {
		log.Printf("Warning: Could not set synchronous mode: %v", err)
	}
	
	return nil
}

// Acquire gets a database connection from the pool
func (p *ConnectionPool) Acquire(ctx context.Context) (*sql.DB, error) {
	startTime := time.Now()
	
	p.mu.Lock()
	
	// Try to find an available connection
	for _, conn := range p.connections {
		if !p.inUse[conn] {
			p.inUse[conn] = true
			p.stats.ActiveConnections++
			p.mu.Unlock()
			
			p.stats.AcquireTime = time.Since(startTime)
			return conn, nil
		}
	}
	
	// If we have capacity, create a new connection
	if len(p.connections) < p.maxSize {
		conn, err := p.createNewConnection(p.dbPath)
		if err != nil {
			p.mu.Unlock()
			p.stats.Errors++
			return nil, err
		}
		
		p.inUse[conn] = true
		p.stats.ActiveConnections++
		p.mu.Unlock()
		
		p.stats.AcquireTime = time.Since(startTime)
		return conn, nil
	}
	
	// No connections available, wait
	p.mu.Unlock()
	p.stats.WaitCount++
	
	select {
	case <-ctx.Done():
		p.stats.Errors++
		return nil, ctx.Err()
	case p.waitQueue <- struct{}{}:
		// Got slot, try again
		return p.Acquire(ctx)
	}
}

// Release returns a connection to the pool
func (p *ConnectionPool) Release(conn *sql.DB) {
	startTime := time.Now()
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.inUse[conn] {
		p.inUse[conn] = false
		p.stats.ActiveConnections--
		
		// Signal waiting goroutines
		select {
		case <-p.waitQueue:
			// Remove one waiter
		default:
			// No waiters
		}
	}
	
	p.stats.ReleaseTime = time.Since(startTime)
}

// createNewConnection creates a new database connection
func (p *ConnectionPool) createNewConnection(dbPath string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	
	// Configure connection
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(30 * time.Minute)
	
	p.connections = append(p.connections, conn)
	p.stats.TotalConnections++
	
	return conn, nil
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.stats
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	var lastErr error
	for _, conn := range p.connections {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
	}
	
	p.connections = nil
	p.inUse = make(map[*sql.DB]bool)
	
	return lastErr
}

// WithConnection executes a function with a database connection
func (p *ConnectionPool) WithConnection(ctx context.Context, fn func(*sql.DB) error) error {
	conn, err := p.Acquire(ctx)
	if err != nil {
		return err
	}
	defer p.Release(conn)
	
	return fn(conn)
}

// OptimizedStorage wraps Storage with connection pooling
type OptimizedStorage struct {
	*Storage
	pool *ConnectionPool
}

// NewOptimizedStorage creates a new optimized storage with connection pooling
func NewOptimizedStorage(dbPath string) (*OptimizedStorage, error) {
	// Create connection pool
	pool, err := NewConnectionPool(dbPath, 5) // 5 connections max
	if err != nil {
		return nil, err
	}
	
	// Create base storage with first connection
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		pool.Close()
		return nil, err
	}
	defer pool.Release(conn)
	
	storage := &Storage{
		db: conn,
	}
	
	return &OptimizedStorage{
		Storage: storage,
		pool:    pool,
	}, nil
}

// StoreEvent stores an event with connection pooling
func (s *OptimizedStorage) StoreEvent(ctx context.Context, event *model.Event) error {
	return s.pool.WithConnection(ctx, func(db *sql.DB) error {
		// Use the provided connection
		tempStorage := &Storage{db: db}
		return tempStorage.StoreEvent(ctx, event)
	})
}

// GetEvent retrieves an event by ID with connection pooling
func (s *OptimizedStorage) GetEvent(ctx context.Context, id string) (*model.Event, error) {
	var result *model.Event
	err := s.pool.WithConnection(ctx, func(db *sql.DB) error {
		tempStorage := &Storage{db: db}
		event, err := tempStorage.GetEvent(ctx, id)
		if err == nil {
			result = event
		}
		return err
	})
	return result, err
}

// GetEventBySourceID retrieves an event by source and source ID with connection pooling
func (s *OptimizedStorage) GetEventBySourceID(ctx context.Context, source, sourceID string) (*model.Event, error) {
	var result *model.Event
	err := s.pool.WithConnection(ctx, func(db *sql.DB) error {
		tempStorage := &Storage{db: db}
		event, err := tempStorage.GetEventBySourceID(ctx, source, sourceID)
		if err == nil {
			result = event
		}
		return err
	})
	return result, err
}

// ListEvents lists events with filtering and connection pooling
func (s *OptimizedStorage) ListEvents(ctx context.Context, filter ListFilter) ([]model.Event, int, error) {
	var events []model.Event
	var total int
	
	err := s.pool.WithConnection(ctx, func(db *sql.DB) error {
		tempStorage := &Storage{db: db}
		evts, tot, err := tempStorage.ListEvents(ctx, filter)
		if err == nil {
			events = evts
			total = tot
		}
		return err
	})
	
	return events, total, err
}

// Close closes the storage and connection pool
func (s *OptimizedStorage) Close() error {
	if s.pool != nil {
		return s.pool.Close()
	}
	return nil
}

// PoolStats returns connection pool statistics
func (s *OptimizedStorage) PoolStats() PoolStats {
	if s.pool != nil {
		return s.pool.Stats()
	}
	return PoolStats{}
}