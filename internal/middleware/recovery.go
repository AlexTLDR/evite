package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// PanicRecovery is a middleware that recovers from panics and returns a 500 error
// instead of crashing the server. It logs the panic and stack trace for debugging.
func PanicRecovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				log.Printf("PANIC RECOVERED: %v\n%s", err, debug.Stack())

				// Return 500 Internal Server Error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next(w, r)
	}
}

// PanicRecoveryWithLogger is a middleware that recovers from panics with a custom logger
func PanicRecoveryWithLogger(logger func(format string, v ...interface{})) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic with stack trace using custom logger
					logger("PANIC RECOVERED: %v\nPath: %s %s\nStack trace:\n%s",
						err, r.Method, r.URL.Path, debug.Stack())

					// Return 500 Internal Server Error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next(w, r)
		}
	}
}

// PanicRecoveryWithCustomError allows customizing the error response
func PanicRecoveryWithCustomError(errorHandler func(w http.ResponseWriter, r *http.Request, err interface{})) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic with stack trace
					log.Printf("PANIC RECOVERED: %v\n%s", err, debug.Stack())

					// Call custom error handler
					errorHandler(w, r, err)
				}
			}()

			next(w, r)
		}
	}
}

// Chain combines multiple middleware functions into one
// Middleware are applied in the order they are provided
func Chain(middlewares ...func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		handler := next
		// Apply middleware in reverse order so they execute in the correct order
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}
