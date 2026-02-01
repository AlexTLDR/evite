package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
)

const (
	csrfTokenLength = 32
	csrfSessionName = "csrf-session"
	csrfTokenKey    = "csrf_token"
)

// CSRFProtection provides CSRF protection for forms
// It generates tokens for safe methods (GET, HEAD, OPTIONS) and validates them for unsafe methods
func CSRFProtection(sessionStore *sessions.CookieStore, excludePaths ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Check if this path should be excluded from CSRF protection
			for _, path := range excludePaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next(w, r)
					return
				}
			}

			session, err := sessionStore.Get(r, csrfSessionName)
			if err != nil {
				// If we can't get the session, create a new one
				session, _ = sessionStore.New(r, csrfSessionName)
			}

			// Safe methods: generate and store token
			if isSafeMethod(r.Method) {
				// Generate token if it doesn't exist
				if _, ok := session.Values[csrfTokenKey].(string); !ok {
					token, err := generateCSRFToken()
					if err != nil {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
					session.Values[csrfTokenKey] = token
					if err := session.Save(r, w); err != nil {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
				}
				next(w, r)
				return
			}

			// Unsafe methods: validate token
			expectedToken, ok := session.Values[csrfTokenKey].(string)
			if !ok || expectedToken == "" {
				http.Error(w, "CSRF token not found. Please refresh the page and try again.", http.StatusForbidden)
				return
			}

			// Get token from form or header
			actualToken := r.FormValue("csrf_token")
			if actualToken == "" {
				actualToken = r.Header.Get("X-CSRF-Token")
			}

			if actualToken == "" || !secureCompare(expectedToken, actualToken) {
				http.Error(w, "Invalid CSRF token. Please refresh the page and try again.", http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}

// GetCSRFToken retrieves the CSRF token from the session
// This should be called in handlers to pass the token to templates
func GetCSRFToken(r *http.Request, sessionStore *sessions.CookieStore) string {
	session, err := sessionStore.Get(r, csrfSessionName)
	if err != nil {
		return ""
	}

	token, ok := session.Values[csrfTokenKey].(string)
	if !ok {
		return ""
	}

	return token
}

// generateCSRFToken generates a random CSRF token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, csrfTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// isSafeMethod returns true if the HTTP method is considered safe (doesn't modify data)
func isSafeMethod(method string) bool {
	return method == "GET" || method == "HEAD" || method == "OPTIONS" || method == "TRACE"
}

// secureCompare performs a constant-time comparison of two strings
// This prevents timing attacks
func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}

	return result == 0
}
