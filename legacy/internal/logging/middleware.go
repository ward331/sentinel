package logging

import (
	"log"
	"net/http"
	"time"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time     `json:"timestamp"`
	Method      string        `json:"method"`
	Path        string        `json:"path"`
	StatusCode  int           `json:"status_code"`
	Duration    time.Duration `json:"duration_ms"`
	ClientIP    string        `json:"client_ip"`
	UserAgent   string        `json:"user_agent,omitempty"`
	RequestID   string        `json:"request_id,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// Logger defines the logging interface
type Logger interface {
	Log(entry LogEntry)
}

// StdLogger logs to standard output
type StdLogger struct{}

// Log logs an entry to standard output
func (l *StdLogger) Log(entry LogEntry) {
	log.Printf("[%s] %s %s %d %v %s",
		entry.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		entry.Method,
		entry.Path,
		entry.StatusCode,
		entry.Duration,
		entry.ClientIP,
	)
}

// JSONLogger logs as JSON (stub for future implementation)
type JSONLogger struct{}

// Log logs an entry as JSON
func (l *JSONLogger) Log(entry LogEntry) {
	// In a real implementation, would marshal to JSON
	log.Printf("JSON LOG: %+v", entry)
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Enabled     bool
	Format      string // "text" or "json"
	LogLevel    string // "info", "debug", "warn", "error"
	IncludeBody bool   // Whether to log request/response bodies
}

// DefaultLoggingConfig returns default logging configuration
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Enabled:     true,
		Format:      "text",
		LogLevel:    "info",
		IncludeBody: false,
	}
}

// LoggingMiddleware creates logging middleware
func LoggingMiddleware(config LoggingConfig) func(http.Handler) http.Handler {
	if !config.Enabled {
		// Logging disabled, return pass-through middleware
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	
	// Create logger based on format
	var logger Logger
	switch config.Format {
	case "json":
		logger = &JSONLogger{}
	default:
		logger = &StdLogger{}
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Create response writer wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			
			// Get client IP
			clientIP := getClientIP(r)
			
			// Process request
			next.ServeHTTP(rw, r)
			
			// Calculate duration
			duration := time.Since(start)
			
			// Create log entry
			entry := LogEntry{
				Timestamp:  start,
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: rw.statusCode,
				Duration:   duration,
				ClientIP:   clientIP,
				UserAgent:  r.UserAgent(),
			}
			
			// Add request ID if present
			if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
				entry.RequestID = requestID
			}
			
			// Log the entry
			logger.Log(entry)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	
	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	
	// Fall back to remote address
	return r.RemoteAddr
}

// RequestLogger logs individual requests with additional context
func RequestLogger(next http.Handler, logger Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create response writer wrapper
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Get client IP
		clientIP := getClientIP(r)
		
		// Process request
		next.ServeHTTP(rw, r)
		
		// Calculate duration
		duration := time.Since(start)
		
		// Create log entry
		entry := LogEntry{
			Timestamp:  start,
			Method:     r.Method,
			Path:       r.URL.Path,
			StatusCode: rw.statusCode,
			Duration:   duration,
			ClientIP:   clientIP,
			UserAgent:  r.UserAgent(),
		}
		
		// Add request ID if present
		if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
			entry.RequestID = requestID
		}
		
		// Log the entry
		logger.Log(entry)
	})
}

// ErrorLogger logs errors with context
func ErrorLogger(logger Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For now, just pass through
		// In a real implementation, would catch and log errors
		next.ServeHTTP(w, r)
	})
}

// Logging utilities for application code
var (
	// Default logger instance
	defaultLogger Logger = &StdLogger{}
)

// SetDefaultLogger sets the default logger
func SetDefaultLogger(logger Logger) {
	defaultLogger = logger
}

// Info logs an info message
func Info(msg string, fields ...interface{}) {
	// Simple implementation - would be enhanced with structured logging
	log.Printf("[INFO] %s", msg)
}

// Error logs an error message
func Error(err error, msg string, fields ...interface{}) {
	log.Printf("[ERROR] %s: %v", msg, err)
}

// Warn logs a warning message
func Warn(msg string, fields ...interface{}) {
	log.Printf("[WARN] %s", msg)
}

// Debug logs a debug message
func Debug(msg string, fields ...interface{}) {
	log.Printf("[DEBUG] %s", msg)
}