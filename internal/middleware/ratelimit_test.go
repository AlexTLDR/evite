package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	// Create a rate limiter: 3 requests per 100ms
	rl := NewRateLimiter(3, 100*time.Millisecond)

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !rl.Allow("test-key") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if rl.Allow("test-key") {
		t.Error("4th request should be blocked")
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow("test-key") {
		t.Error("Request after window reset should be allowed")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)

	// Use up quota for key1
	rl.Allow("key1")
	rl.Allow("key1")

	// key1 should be blocked
	if rl.Allow("key1") {
		t.Error("key1 should be blocked")
	}

	// key2 should still be allowed
	if !rl.Allow("key2") {
		t.Error("key2 should be allowed")
	}
}

func TestRateLimitByIP(t *testing.T) {
	// Create middleware: 2 requests per 100ms
	middleware := RateLimitByIP(2, 100*time.Millisecond)

	handler := middleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got status %d", i+1, w.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request should be rate limited, got status %d", w.Code)
	}
}

func TestRateLimitByIP_DifferentIPs(t *testing.T) {
	middleware := RateLimitByIP(1, 100*time.Millisecond)

	handler := middleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Request from IP1
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	handler(w1, req1)

	if w1.Code != http.StatusOK {
		t.Error("First request from IP1 should succeed")
	}

	// Second request from IP1 should be blocked
	w1b := httptest.NewRecorder()
	handler(w1b, req1)

	if w1b.Code != http.StatusTooManyRequests {
		t.Error("Second request from IP1 should be blocked")
	}

	// Request from IP2 should succeed
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	w2 := httptest.NewRecorder()
	handler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Error("First request from IP2 should succeed")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expectedIP    string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For header",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1",
			expectedIP:    "203.0.113.1",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.2",
			expectedIP: "203.0.113.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}
