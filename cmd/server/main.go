package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlexTLDR/evite/internal/config"
	"github.com/AlexTLDR/evite/internal/database"
	"github.com/AlexTLDR/evite/internal/middleware"
	"github.com/AlexTLDR/evite/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (ignore error if a file doesn't exist)
	// Use Overload to force to overwrite any existing environment variables
	err := godotenv.Overload()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	} else {
		log.Printf(".env file loaded successfully (with overload)")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}(db)

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create the server
	srv := server.New(cfg, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on :%s", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Stop all rate limiter cleanup goroutines
	middleware.StopAll()

	log.Println("Server exited gracefully")
}
