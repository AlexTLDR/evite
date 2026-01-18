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

// homePageData holds all data needed to render the home page
type homePageData struct {
	lang           string
	lightTheme     string
	darkTheme      string
	invitation     *database.Invitation
	deadlinePassed bool
}

// loadInvitationByToken loads an invitation by token and marks it as opened
func loadInvitationByToken(db *database.DB, token string) *database.Invitation {
	if token == "" {
		return nil
	}

	invitation, err := db.GetInvitationByToken(token)
	if err != nil {
		return nil
	}

	// Mark as opened if not already
	if !invitation.OpenedAt.Valid {
		if err := db.MarkAsOpened(invitation.ID); err != nil {
			// Log but don't fail - this is just tracking
			fmt.Printf("Warning: failed to mark invitation as opened: %v\n", err)
		}
	}

	return invitation
}

// checkDeadlinePassed checks if the RSVP deadline has passed with debug logging
func checkDeadlinePassed(cfg *config.Config) bool {
	now := time.Now()
	deadlinePassed := now.After(cfg.RSVPDeadline)

	// Debug logging
	fmt.Printf("DEBUG: Current time: %v\n", now)
	fmt.Printf("DEBUG: RSVP Deadline: %v\n", cfg.RSVPDeadline)
	fmt.Printf("DEBUG: Deadline passed: %v\n", deadlinePassed)

	return deadlinePassed
}

// prepareHomePageData gathers all data needed for the home page
func prepareHomePageData(s Server, r *http.Request) homePageData {
	lang := i18n.GetLanguageFromRequest(r)
	themes := config.GetThemes()
	token := r.URL.Query().Get("token")

	return homePageData{
		lang:           string(lang),
		lightTheme:     themes.Light,
		darkTheme:      themes.Dark,
		invitation:     loadInvitationByToken(s.GetDB(), token),
		deadlinePassed: checkDeadlinePassed(s.GetConfig()),
	}
}

// HandleHome renders the home page
func HandleHome(s Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := prepareHomePageData(s, r)

		if err := templates.Home(data.lang, data.lightTheme, data.darkTheme, data.invitation, data.deadlinePassed).Render(r.Context(), w); err != nil {
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
