package i18n

import (
	"net/http"
)

type Language string

const (
	Romanian Language = "ro"
	English  Language = "en"
)

// GetLanguageFromRequest extracts language from request (query param or cookie)
func GetLanguageFromRequest(r *http.Request) Language {
	// Check query parameter first
	if lang := r.URL.Query().Get("lang"); lang != "" {
		if lang == "ro" {
			return Romanian
		}
		if lang == "en" {
			return English
		}
	}

	// Check cookie
	if cookie, err := r.Cookie("lang"); err == nil {
		if cookie.Value == "ro" {
			return Romanian
		}
		if cookie.Value == "en" {
			return English
		}
	}

	// Default to Romanian
	return Romanian
}
