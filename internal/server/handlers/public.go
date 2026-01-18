package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/AlexTLDR/evite/internal/config"
	"github.com/AlexTLDR/evite/internal/database"
	"github.com/AlexTLDR/evite/internal/i18n"
	"github.com/AlexTLDR/evite/templates"
)

// Server interface defines the methods needed by handlers
type Server interface {
	GetDB() *database.DB
	GetConfig() *config.Config
}

// HandleHome renders the home page
func HandleHome(s Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.GetLanguageFromRequest(r)
		themes := config.GetThemes()

		// Check if there's a token in the query parameter
		token := r.URL.Query().Get("token")

		var invitation *database.Invitation
		if token != "" {
			// Try to get invitation by token
			inv, err := s.GetDB().GetInvitationByToken(token)
			if err == nil {
				invitation = inv
				// Mark as opened if not already
				if !invitation.OpenedAt.Valid {
					if err := s.GetDB().MarkAsOpened(invitation.ID); err != nil {
						// Log but don't fail - this is just tracking
						fmt.Printf("Warning: failed to mark invitation as opened: %v\n", err)
					}
				}
			}
		}

		// Check if the RSVP deadline has passed
		now := time.Now()
		deadlinePassed := now.After(s.GetConfig().RSVPDeadline)

		// Debug logging
		fmt.Printf("DEBUG: Current time: %v\n", now)
		fmt.Printf("DEBUG: RSVP Deadline: %v\n", s.GetConfig().RSVPDeadline)
		fmt.Printf("DEBUG: Deadline passed: %v\n", deadlinePassed)

		if err := templates.Home(string(lang), themes.Light, themes.Dark, invitation, deadlinePassed).Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
	}
}

// HandleRSVP redirects RSVP links to home page with token
func HandleRSVP(s Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from URL path
		token := r.URL.Path[len("/rsvp/"):]
		if token == "" {
			// Redirect to home page without the token
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Redirect to home page with token as query parameter
		http.Redirect(w, r, "/?token="+token, http.StatusSeeOther)
	}
}
