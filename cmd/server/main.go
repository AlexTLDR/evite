package main

import (
	"log"
	"os"

	"github.com/AlexTLDR/evite/internal/config"
	"github.com/AlexTLDR/evite/internal/database"
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
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("Failed to close database: %v", err)
		}
	}(db)

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create and start the server
	srv := server.New(cfg, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on :%s", port)
	if err := srv.Start(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
