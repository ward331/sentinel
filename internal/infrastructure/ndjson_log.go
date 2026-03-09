package infrastructure

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/openclaw/sentinel-backend/internal/model"
)

// NDJSONLog implements an append-only NDJSON event log
type NDJSONLog struct {
	file     *os.File
	writer   *bufio.Writer
	mu       sync.Mutex
	filePath string
}

// NewNDJSONLog creates a new NDJSON log at the specified path
func NewNDJSONLog(logPath string) (*NDJSONLog, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file in append mode, create if it doesn't exist
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &NDJSONLog{
		file:     file,
		writer:   bufio.NewWriter(file),
		filePath: logPath,
	}, nil
}

// AppendEvent appends an event to the NDJSON log
func (l *NDJSONLog) AppendEvent(event *model.Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create log entry with timestamp
	logEntry := struct {
		Timestamp time.Time   `json:"timestamp"`
		Event     *model.Event `json:"event"`
	}{
		Timestamp: time.Now().UTC(),
		Event:     event,
	}

	// Marshal to JSON
	data, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write as NDJSON (JSON line)
	if _, err := l.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write to log: %w", err)
	}
	if err := l.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	// Flush to ensure data is written
	return l.writer.Flush()
}

// Close closes the log file
func (l *NDJSONLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.writer.Flush(); err != nil {
		return err
	}
	return l.file.Close()
}

// Rotate rotates the log file (creates new file with timestamp)
func (l *NDJSONLog) Rotate() (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close current file
	if err := l.writer.Flush(); err != nil {
		return "", err
	}
	if err := l.file.Close(); err != nil {
		return "", err
	}

	// Create new filename with timestamp
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05")
	newPath := fmt.Sprintf("%s.%s", l.filePath, timestamp)

	// Rename current file
	if err := os.Rename(l.filePath, newPath); err != nil {
		return "", fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Open new file
	file, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open new log file: %w", err)
	}

	l.file = file
	l.writer = bufio.NewWriter(file)

	return newPath, nil
}

// GetLogPath returns the path to the log file
func (l *NDJSONLog) GetLogPath() string {
	return l.filePath
}

// Stats returns statistics about the log file
func (l *NDJSONLog) Stats() (os.FileInfo, error) {
	return os.Stat(l.filePath)
}