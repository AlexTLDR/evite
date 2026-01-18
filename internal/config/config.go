package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	// Database
	DatabaseURL string

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	AdminEmails        []string

	// Session
	SessionSecret string

	// Event Details
	EventDate         time.Time
	RSVPDeadline      time.Time
	ChurchName        string
	ChurchAddress     string
	RestaurantName    string
	RestaurantAddress string

	// App
	BaseURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://evite:evite@localhost:5432/evite?sslmode=disable"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		SessionSecret:      getEnv("SESSION_SECRET", "change-me-in-production"),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
		ChurchName:         getEnv("CHURCH_NAME", ""),
		ChurchAddress:      getEnv("CHURCH_ADDRESS", ""),
		RestaurantName:     getEnv("RESTAURANT_NAME", ""),
		RestaurantAddress:  getEnv("RESTAURANT_ADDRESS", ""),
	}

	// Parse admin emails
	adminEmailsStr := getEnv("ADMIN_EMAILS", "")
	if adminEmailsStr != "" {
		cfg.AdminEmails = strings.Split(adminEmailsStr, ",")
		for i := range cfg.AdminEmails {
			cfg.AdminEmails[i] = strings.TrimSpace(cfg.AdminEmails[i])
		}
	}

	// Parse event date
	eventDateStr := getEnv("EVENT_DATE", "2026-04-19T14:00:00")
	eventDate, err := time.Parse(time.RFC3339, eventDateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid EVENT_DATE format: %w", err)
	}
	loc, _ := time.LoadLocation("Europe/Bucharest")
	cfg.EventDate = eventDate.In(loc)

	// Parse RSVP deadline
	deadlineStr := getEnv("RSVP_DEADLINE", "2026-04-12T23:59:59")
	deadline, err := time.Parse(time.RFC3339, deadlineStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RSVP_DEADLINE format: %w", err)
	}
	cfg.RSVPDeadline = deadline.In(loc)

	// Debug logging
	fmt.Printf("CONFIG DEBUG: RSVP_DEADLINE string: %s\n", deadlineStr)
	fmt.Printf("CONFIG DEBUG: RSVP_DEADLINE parsed: %v\n", cfg.RSVPDeadline)
	fmt.Printf("CONFIG DEBUG: Current time: %v\n", time.Now())

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
