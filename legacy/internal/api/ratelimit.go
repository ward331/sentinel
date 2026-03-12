package api

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements IP-based rate limiting
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rps      int
	burst    int
	cleanup  *time.Ticker
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    burst,
		cleanup:  time.NewTicker(time.Minute),
	}
	
	// Start cleanup goroutine
	go rl.cleanupOldLimiters()
	
	return rl
}

// cleanupOldLimiters periodically removes old limiters to prevent memory leak
func (rl *RateLimiter) cleanupOldLimiters() {
	for range rl.cleanup.C {
		rl.mu.Lock()
		// For simplicity, we'll just keep all limiters for now
		// In a production system, we'd track last access time
		rl.mu.Unlock()
	}
}

// getLimiter returns or creates a rate limiter for an IP
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[ip]
	rl.mu.RUnlock()
	
	if exists {
		return limiter
	}
	
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	// Check again in case it was created while we were waiting for lock
	if limiter, exists := rl.limiters[ip]; exists {
		return limiter
	}
	
	limiter = rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
	rl.limiters[ip] = limiter
	return limiter
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	return rl.getLimiter(ip).Allow()
}

// Middleware returns a rate limiting middleware
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		ip := getClientIP(r)
		
		// Check rate limit
		if !rl.Allow(ip) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
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

// Close stops the cleanup goroutine
func (rl *RateLimiter) Close() {
	rl.cleanup.Stop()
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled bool
	RPS     int
	Burst   int
	ExemptPaths []string
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled: false,
		RPS:     100,
		Burst:   200,
		ExemptPaths: []string{
			"/api/health",
			"/api/metrics",
		},
	}
}

// RateLimitMiddleware creates rate limiting middleware based on configuration
func RateLimitMiddleware(config RateLimitConfig) func(http.Handler) http.Handler {
	if !config.Enabled {
		// Rate limiting disabled, return pass-through middleware
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	
	// Create rate limiter
	limiter := NewRateLimiter(config.RPS, config.Burst)
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is exempt
			for _, exemptPath := range config.ExemptPaths {
				if r.URL.Path == exemptPath {
					next.ServeHTTP(w, r)
					return
				}
			}
			
			// Apply rate limiting
			ip := getClientIP(r)
			if !limiter.Allow(ip) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}