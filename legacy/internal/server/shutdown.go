package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/openclaw/sentinel-backend/internal/backup"
)

// ShutdownManager manages graceful shutdown of server components
type ShutdownManager struct {
	mu          sync.RWMutex
	components  []Shutdownable
	timeout     time.Duration
	shuttingDown bool
}

// Shutdownable represents a component that can be shut down gracefully
type Shutdownable interface {
	Shutdown(ctx context.Context) error
	Name() string
}

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager(timeout time.Duration) *ShutdownManager {
	return &ShutdownManager{
		components: make([]Shutdownable, 0),
		timeout:    timeout,
	}
}

// Register adds a component to be shut down
func (sm *ShutdownManager) Register(component Shutdownable) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.components = append(sm.components, component)
}

// ShutdownAll shuts down all registered components
func (sm *ShutdownManager) ShutdownAll() error {
	sm.mu.Lock()
	if sm.shuttingDown {
		sm.mu.Unlock()
		return nil // Already shutting down
	}
	sm.shuttingDown = true
	sm.mu.Unlock()
	
	log.Println("Initiating graceful shutdown...")
	
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), sm.timeout)
	defer cancel()
	
	// Shutdown components in reverse order (dependencies first)
	var errs []error
	for i := len(sm.components) - 1; i >= 0; i-- {
		component := sm.components[i]
		log.Printf("Shutting down %s...", component.Name())
		
		if err := component.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down %s: %v", component.Name(), err)
			errs = append(errs, err)
		} else {
			log.Printf("%s shut down successfully", component.Name())
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("shutdown completed with %d error(s)", len(errs))
	}
	
	log.Println("Graceful shutdown completed")
	return nil
}

// WaitForSignal waits for termination signals and triggers shutdown
func (sm *ShutdownManager) WaitForSignal() {
	// Create signal channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Wait for signal
	sig := <-sigChan
	log.Printf("Received signal: %v", sig)
	
	// Trigger shutdown
	if err := sm.ShutdownAll(); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

// HTTPServerWrapper wraps http.Server for shutdown management
type HTTPServerWrapper struct {
	server *http.Server
	name   string
}

// NewHTTPServerWrapper creates a new HTTP server wrapper
func NewHTTPServerWrapper(server *http.Server, name string) *HTTPServerWrapper {
	return &HTTPServerWrapper{
		server: server,
		name:   name,
	}
}

// Shutdown shuts down the HTTP server
func (h *HTTPServerWrapper) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

// Name returns the server name
func (h *HTTPServerWrapper) Name() string {
	return h.name
}

// PollerWrapper wraps a poller for shutdown management
type PollerWrapper struct {
	cancel context.CancelFunc
	name   string
}

// NewPollerWrapper creates a new poller wrapper
func NewPollerWrapper(cancel context.CancelFunc, name string) *PollerWrapper {
	return &PollerWrapper{
		cancel: cancel,
		name:   name,
	}
}

// Shutdown stops the poller
func (p *PollerWrapper) Shutdown(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
		// Wait a moment for poller to stop
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Poller should have stopped
		}
	}
	return nil
}

// Name returns the poller name
func (p *PollerWrapper) Name() string {
	return p.name
}

// StorageWrapper wraps storage for shutdown management
type StorageWrapper struct {
	closer func() error
	name   string
}

// NewStorageWrapper creates a new storage wrapper
func NewStorageWrapper(closer func() error, name string) *StorageWrapper {
	return &StorageWrapper{
		closer: closer,
		name:   name,
	}
}

// Shutdown closes the storage
func (s *StorageWrapper) Shutdown(ctx context.Context) error {
	if s.closer != nil {
		return s.closer()
	}
	return nil
}

// Name returns the storage name
func (s *StorageWrapper) Name() string {
	return s.name
}

// GracefulShutdownConfig holds shutdown configuration
type GracefulShutdownConfig struct {
	Timeout time.Duration
}

// DefaultGracefulShutdownConfig returns default shutdown configuration
func DefaultGracefulShutdownConfig() GracefulShutdownConfig {
	return GracefulShutdownConfig{
		Timeout: 30 * time.Second,
	}
}

// BackupWrapper wraps a backup manager for shutdown management
type BackupWrapper struct {
	manager *backup.BackupManager
	name    string
}

// NewBackupWrapper creates a new backup wrapper
func NewBackupWrapper(manager *backup.BackupManager, name string) *BackupWrapper {
	return &BackupWrapper{
		manager: manager,
		name:    name,
	}
}

// Shutdown shuts down the backup manager
func (b *BackupWrapper) Shutdown(ctx context.Context) error {
	// Backup manager doesn't need explicit shutdown
	// It will stop when context is cancelled
	return nil
}

// Name returns the backup manager name
func (b *BackupWrapper) Name() string {
	return b.name
}

// SetupGracefulShutdown sets up graceful shutdown for a server
func SetupGracefulShutdown(
	httpServer *http.Server,
	pollerCancel context.CancelFunc,
	storageCloser func() error,
	backupManager *backup.BackupManager,
	config GracefulShutdownConfig,
) *ShutdownManager {
	// Create shutdown manager
	shutdownManager := NewShutdownManager(config.Timeout)
	
	// Register components in dependency order
	// (HTTP server first, then poller, then backup, then storage)
	shutdownManager.Register(NewHTTPServerWrapper(httpServer, "HTTP Server"))
	if pollerCancel != nil {
		shutdownManager.Register(NewPollerWrapper(pollerCancel, "USGS Poller"))
	}
	if backupManager != nil {
		shutdownManager.Register(NewBackupWrapper(backupManager, "Backup Manager"))
	}
	if storageCloser != nil {
		shutdownManager.Register(NewStorageWrapper(storageCloser, "Database Storage"))
	}
	
	return shutdownManager
}