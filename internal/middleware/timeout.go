package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout wraps a handler with a timeout
// If the handler doesn't complete within the specified duration,
// it returns a 408 Request Timeout error
func Timeout(duration time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), duration)
			defer cancel()

			// Channel to signal completion
			done := make(chan struct{})

			// Response writer wrapper to track if headers were written
			tw := &timeoutWriter{ResponseWriter: w, done: done}

			// Run the handler in a goroutine
			go func() {
				defer func() {
					// Recover from panics in the handler
					if err := recover(); err != nil {
						// Let the panic recovery middleware handle this
						panic(err)
					}
				}()

				next(tw, r.WithContext(ctx))
				close(done)
			}()

			// Wait for either completion or timeout
			select {
			case <-done:
				// Handler completed successfully
				return
			case <-ctx.Done():
				// Timeout occurred
				if !tw.headerWritten {
					http.Error(w, "Request timeout", http.StatusRequestTimeout)
				}
				// If headers were already written, we can't send an error
				// The response is already partially sent
			}
		}
	}
}

// timeoutWriter wraps http.ResponseWriter to track if headers were written
type timeoutWriter struct {
	http.ResponseWriter
	headerWritten bool
	done          chan struct{}
}

func (tw *timeoutWriter) WriteHeader(code int) {
	select {
	case <-tw.done:
		// Handler already completed, don't write
		return
	default:
		tw.headerWritten = true
		tw.ResponseWriter.WriteHeader(code)
	}
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	select {
	case <-tw.done:
		// Handler already completed, don't write
		return 0, http.ErrHandlerTimeout
	default:
		tw.headerWritten = true
		return tw.ResponseWriter.Write(b)
	}
}
