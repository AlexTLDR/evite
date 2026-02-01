package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
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
	done     chan struct{} // for graceful shutdown
	once     sync.Once     // ensure cleanup goroutine starts only once
}

// bucket represents a token bucket for a specific key
type bucket struct {
	tokens    int
	lastReset time.Time
	mu        sync.Mutex
}

var (
	// Global rate limiter registry to prevent goroutine leaks
	rateLimitersMu sync.RWMutex
	rateLimiters   = make(map[string]*RateLimiter)
)

// rateLimiterKey generates a unique key for a rate limiter configuration
func rateLimiterKey(rate int, window time.Duration) string {
	return fmt.Sprintf("%d-%d", rate, window.Nanoseconds())
}

// NewRateLimiter creates a new rate limiter or returns an existing one
// rate: number of requests allowed
// window: time window for the rate limit (e.g., 15 minutes)
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	key := rateLimiterKey(rate, window)

	// Check if limiter already exists
	rateLimitersMu.RLock()
	if rl, exists := rateLimiters[key]; exists {
		rateLimitersMu.RUnlock()
		return rl
	}
	rateLimitersMu.RUnlock()

	// Create new limiter
	rateLimitersMu.Lock()
	defer rateLimitersMu.Unlock()

	// Double-check after acquiring write lock
	if rl, exists := rateLimiters[key]; exists {
		return rl
	}

	rl := &RateLimiter{
		limiters: make(map[string]*bucket),
		rate:     rate,
		window:   window,
		cleanup:  window * 2, // cleanup old entries periodically
		done:     make(chan struct{}),
	}

	// Start cleanup goroutine only once
	go rl.cleanupLoop()

	rateLimiters[key] = rl
	return rl
}

// Allow checks if a request should be allowed for the given key
// This method is thread-safe and prevents race conditions with the cleanup goroutine
func (rl *RateLimiter) Allow(key string) bool {
	// First, try to get or create the bucket
	rl.mu.Lock()
	b, exists := rl.limiters[key]
	if !exists {
		b = &bucket{
			tokens:    rl.rate - 1, // Consume token immediately
			lastReset: time.Now(),
		}
		rl.limiters[key] = b
		rl.mu.Unlock()
		return true
	}

	// Lock the bucket before releasing the map lock to prevent deletion race
	b.mu.Lock()
	rl.mu.Unlock()
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

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			var keysToDelete []string

			// First pass: identify keys to delete (with read lock)
			rl.mu.RLock()
			for key, b := range rl.limiters {
				b.mu.Lock()
				if now.Sub(b.lastReset) > rl.cleanup {
					keysToDelete = append(keysToDelete, key)
				}
				b.mu.Unlock()
			}
			rl.mu.RUnlock()

			// Second pass: delete identified keys (with write lock)
			if len(keysToDelete) > 0 {
				rl.mu.Lock()
				for _, key := range keysToDelete {
					delete(rl.limiters, key)
				}
				rl.mu.Unlock()
			}
		case <-rl.done:
			return
		}
	}
}

// Stop gracefully stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	rl.once.Do(func() {
		close(rl.done)
	})
}

// StopAll gracefully stops all rate limiter cleanup goroutines
// This should be called during application shutdown
func StopAll() {
	rateLimitersMu.Lock()
	defer rateLimitersMu.Unlock()

	for _, rl := range rateLimiters {
		rl.Stop()
	}
	// Clear the registry
	rateLimiters = make(map[string]*RateLimiter)
}

// getClientIP extracts the real client IP from the request
// trustProxy: whether to trust X-Forwarded-For and X-Real-IP headers
// When behind a reverse proxy (like Traefik/nginx), set trustProxy to true
// When directly exposed to the internet, set trustProxy to false to prevent IP spoofing
func getClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		// Check X-Forwarded-For header (for proxies/load balancers)
		// Take the FIRST IP (leftmost) which is the original client IP
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// X-Forwarded-For can contain multiple IPs: client, proxy1, proxy2
			// We want the leftmost (original client)
			if idx := strings.Index(xff, ","); idx != -1 {
				return strings.TrimSpace(xff[:idx])
			}
			return strings.TrimSpace(xff)
		}

		// Check X-Real-IP header (set by some proxies)
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	// Fall back to RemoteAddr (direct connection or trustProxy=false)
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}

// RateLimitByIP creates a middleware that rate limits by IP address
// trustProxy: whether to trust X-Forwarded-For and X-Real-IP headers
func RateLimitByIP(rate int, window time.Duration, trustProxy bool) func(http.HandlerFunc) http.HandlerFunc {
	limiter := NewRateLimiter(rate, window)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r, trustProxy)

			if !limiter.Allow(ip) {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", window.Seconds()))
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next(w, r)
		}
	}
}

// RateLimitByKey creates a middleware that rate limits by a custom key extracted from the request
// trustProxy: whether to trust X-Forwarded-For and X-Real-IP headers for IP fallback
func RateLimitByKey(rate int, window time.Duration, trustProxy bool, keyFunc func(*http.Request) string) func(http.HandlerFunc) http.HandlerFunc {
	limiter := NewRateLimiter(rate, window)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if key == "" {
				// If no key, fall back to IP
				key = getClientIP(r, trustProxy)
			}

			if !limiter.Allow(key) {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", window.Seconds()))
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
