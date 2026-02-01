package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related HTTP headers to responses
// This helps protect against common web vulnerabilities
func SecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking attacks by disallowing the page to be embedded in frames
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing which could lead to XSS attacks
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable browser's XSS protection (legacy, but still useful for older browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Control how much referrer information should be included with requests
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy - allows inline styles/scripts for templ templates
		// All resources are served locally for better security
		// Note: 'unsafe-eval' is required for Alpine.js which uses Function() constructor
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"font-src 'self' data:; "+
				"img-src 'self' data: https:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none';")

		// HSTS - only set if using HTTPS
		// This tells browsers to only connect via HTTPS for the next year
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next(w, r)
	}
}
