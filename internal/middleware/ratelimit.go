package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*bucket
	rate     int           // requests allowed
	window   time.Duration // time window
	cleanup  time.Duration // cleanup interval
}

// bucket represents a token bucket for a specific key
type bucket struct {
	tokens    int
	lastReset time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed
// window: time window for the rate limit (e.g., 15 minutes)
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*bucket),
		rate:     rate,
		window:   window,
		cleanup:  window * 2, // cleanup old entries periodically
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request should be allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.RLock()
	b, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if !exists {
		b = &bucket{
			tokens:    rl.rate,
			lastReset: time.Now(),
		}
		rl.mu.Lock()
		rl.limiters[key] = b
		rl.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Reset bucket if window has passed
	if time.Since(b.lastReset) > rl.window {
		b.tokens = rl.rate
		b.lastReset = time.Now()
	}

	// Check if tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanupLoop periodically removes old entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.limiters {
			b.mu.Lock()
			if now.Sub(b.lastReset) > rl.cleanup {
				delete(rl.limiters, key)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if ip, _, err := net.SplitHostPort(xff); err == nil {
			return ip
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}

// RateLimitByIP creates a middleware that rate limits by IP address
func RateLimitByIP(rate int, window time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	limiter := NewRateLimiter(rate, window)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if !limiter.Allow(ip) {
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next(w, r)
		}
	}
}

// RateLimitByKey creates a middleware that rate limits by a custom key extracted from the request
func RateLimitByKey(rate int, window time.Duration, keyFunc func(*http.Request) string) func(http.HandlerFunc) http.HandlerFunc {
	limiter := NewRateLimiter(rate, window)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if key == "" {
				// If no key, fall back to IP
				key = getClientIP(r)
			}

			if !limiter.Allow(key) {
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next(w, r)
		}
	}
}

// CombinedRateLimit applies multiple rate limiters (all must pass)
func CombinedRateLimit(limiters ...func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		handler := next
		// Apply limiters in reverse order so they execute in the correct order
		for i := len(limiters) - 1; i >= 0; i-- {
			handler = limiters[i](handler)
		}
		return handler
	}
}
