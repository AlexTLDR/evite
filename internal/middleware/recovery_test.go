package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPanicRecovery_NoPanic(t *testing.T) {
	handler := PanicRecovery(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", w.Body.String())
	}
}

func TestPanicRecovery_WithPanic(t *testing.T) {
	// Capture log output
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	handler := PanicRecovery(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic - should recover
	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Internal Server Error") {
		t.Errorf("Expected 'Internal Server Error' in body, got '%s'", w.Body.String())
	}

	// Check that panic was logged
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "PANIC RECOVERED") {
		t.Error("Expected panic to be logged")
	}
	if !strings.Contains(logOutput, "test panic") {
		t.Error("Expected panic message in log")
	}
}

func TestPanicRecoveryWithLogger(t *testing.T) {
	var loggedMessage string
	customLogger := func(format string, v ...interface{}) {
		// Capture the formatted message
		if len(v) > 0 {
			loggedMessage = format
		}
	}

	handler := PanicRecoveryWithLogger(customLogger)(func(w http.ResponseWriter, r *http.Request) {
		panic("custom panic")
	})

	req := httptest.NewRequest("GET", "/test-path", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	if !strings.Contains(loggedMessage, "PANIC RECOVERED") {
		t.Error("Expected custom logger to be called with panic message")
	}
}

func TestPanicRecoveryWithCustomError(t *testing.T) {
	// Redirect log output to avoid panic in log.Printf
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	customErrorCalled := false
	var capturedErr interface{}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err interface{}) {
		customErrorCalled = true
		capturedErr = err
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Custom error response"))
	}

	handler := PanicRecoveryWithCustomError(errorHandler)(func(w http.ResponseWriter, r *http.Request) {
		panic("test error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if !customErrorCalled {
		t.Error("Expected custom error handler to be called")
	}

	if capturedErr != "test error" {
		t.Errorf("Expected captured error to be 'test error', got '%v'", capturedErr)
	}

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Custom error response") {
		t.Errorf("Expected custom error response, got '%s'", w.Body.String())
	}
}

func TestChain(t *testing.T) {
	// Create middleware that adds headers
	middleware1 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-1", "applied")
			next(w, r)
		}
	}

	middleware2 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-2", "applied")
			next(w, r)
		}
	}

	handler := Chain(middleware1, middleware2)(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Header().Get("X-Middleware-1") != "applied" {
		t.Error("Expected middleware 1 to be applied")
	}

	if w.Header().Get("X-Middleware-2") != "applied" {
		t.Error("Expected middleware 2 to be applied")
	}
}
