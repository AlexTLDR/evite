package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/AlexTLDR/evite/internal/config"
	"github.com/AlexTLDR/evite/internal/database"
	"github.com/AlexTLDR/evite/internal/i18n"
	"github.com/AlexTLDR/evite/internal/utils"
	"github.com/AlexTLDR/evite/templates"
)

// AdminServer extends Server with admin-specific methods
type AdminServer interface {
	Server
	GetCurrentUser(r *http.Request) (string, string)
}

// parseID parses an ID string and returns an error if invalid
func parseID(idStr string) (int64, error) {
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return 0, fmt.Errorf("invalid ID format: %w", err)
	}
	if id <= 0 {
		return 0, fmt.Errorf("invalid ID: must be positive")
	}
	return id, nil
}

// parseFormID parses and validates an ID from a POST form
// Returns the ID and true if successful, or writes an error response and returns false
func parseFormID(r *http.Request, w http.ResponseWriter) (int64, bool) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
		return 0, false
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return 0, false
	}

	idStr := r.FormValue("id")
	id, err := parseID(idStr)
	if err != nil {
		http.Error(w, "Invalid invitation ID", http.StatusBadRequest)
		return 0, false
	}

	return id, true
}

// HandleAdminDashboard renders the admin dashboard
func HandleAdminDashboard(s AdminServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email, name := s.GetCurrentUser(r)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>Admin Dashboard</title>
			</head>
			<body>
				<h1>Admin Dashboard</h1>
				<p>Welcome, %s (%s)</p>
				<nav>
					<a href="/admin/invitations">Invitations</a> |
					<a href="/auth/logout">Logout</a>
				</nav>
			</body>
			</html>
		`, name, email)
	}
}

// HandleAdminInvitations lists all invitations
func HandleAdminInvitations(s AdminServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, userName := s.GetCurrentUser(r)

		invitations, err := s.GetDB().GetAllInvitationsWithResponses()
		if err != nil {
			http.Error(w, "Failed to load invitations", http.StatusInternalServerError)
			return
		}

		themes := config.GetThemes()
		if err := templates.AdminInvitationsList(userName, invitations, themes.Light, themes.Dark).Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
	}
}

// HandleAdminNewInvitation shows the new invitation form
func HandleAdminNewInvitation(s AdminServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, userName := s.GetCurrentUser(r)
		themes := config.GetThemes()
		if err := templates.AdminNewInvitation(userName, "", themes.Light, themes.Dark).Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
	}
}

// invitationFormData holds parsed invitation form data
type invitationFormData struct {
	guestName string
	phone     string
}

// parseInvitationForm parses and validates the invitation form
func parseInvitationForm(r *http.Request, w http.ResponseWriter, userName string, themes config.ThemeConfig) (*invitationFormData, bool) {
	if err := r.ParseForm(); err != nil {
		_ = templates.AdminNewInvitation(userName, "Eroare la procesarea formularului", themes.Light, themes.Dark).Render(r.Context(), w)
		return nil, false
	}

	guestName := strings.TrimSpace(r.FormValue("guest_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))

	// Validate required fields
	if guestName == "" || phone == "" {
		_ = templates.AdminNewInvitation(userName, "Toate câmpurile sunt obligatorii", themes.Light, themes.Dark).Render(r.Context(), w)
		return nil, false
	}

	// Normalize phone number to E.164 format
	normalizedPhone, err := utils.NormalizePhoneNumber(phone)
	if err != nil {
		_ = templates.AdminNewInvitation(userName, "Număr de telefon invalid", themes.Light, themes.Dark).Render(r.Context(), w)
		return nil, false
	}

	return &invitationFormData{
		guestName: guestName,
		phone:     normalizedPhone,
	}, true
}

// createInvitationRecord creates a new invitation in the database
func createInvitationRecord(s Server, formData *invitationFormData, messageTemplate string) (*database.Invitation, error) {
	return s.GetDB().CreateInvitation(formData.guestName, formData.phone, messageTemplate)
}

// handleInvitationCreationError renders an error message for invitation creation failures
func handleInvitationCreationError(err error, w http.ResponseWriter, r *http.Request, userName string, themes config.ThemeConfig) {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		_ = templates.AdminNewInvitation(userName, "Acest număr de telefon există deja", themes.Light, themes.Dark).Render(r.Context(), w)
		return
	}
	_ = templates.AdminNewInvitation(userName, "Eroare la crearea invitației", themes.Light, themes.Dark).Render(r.Context(), w)
}

// createInvitationWithMessage creates an invitation and updates its message with the token
func createInvitationWithMessage(s Server, formData *invitationFormData, messageTemplate string, w http.ResponseWriter, r *http.Request, userName string, themes config.ThemeConfig) bool {
	// Create invitation (this will generate the token)
	inv, err := createInvitationRecord(s, formData, messageTemplate)
	if err != nil {
		handleInvitationCreationError(err, w, r, userName, themes)
		return false
	}

	// Update the message with the actual token
	if err := updateInvitationMessage(s, inv, messageTemplate); err != nil {
		// Log error but don't fail - the invitation is created
		fmt.Printf("Warning: failed to update invite message: %v\n", err)
	}

	return true
}

// updateInvitationMessage replaces placeholders in the message template and updates the invitation
func updateInvitationMessage(s Server, inv *database.Invitation, messageTemplate string) error {
	rsvpLink := fmt.Sprintf("%s/rsvp/%s", s.GetConfig().BaseURL, inv.Token)
	finalMessage := strings.Replace(messageTemplate, "{{TOKEN}}", inv.Token, 1)
	finalMessage = strings.Replace(finalMessage, "{{RSVP_LINK}}", rsvpLink, 1)

	_, err := s.GetDB().Exec("UPDATE invitations SET invite_message = $1 WHERE id = $2", finalMessage, inv.ID)
	return err
}

// generateInviteMessageTemplate creates an invitation message template
func generateInviteMessageTemplate(s Server, guestName string, lang i18n.Language) string {
	return fmt.Sprintf(`Bună %s,

Cu multă bucurie și emoție, vă invităm să fiți alături de noi la botezul micuței noastre Anya-Maria. Detaliile evenimentului sunt în link-ul de mai jos:

{{RSVP_LINK}}

Evenimentul va avea loc pe 19 Aprilie 2026

Cu drag,
Familia Mureșeanu`, guestName)
}

// HandleAdminCreateInvitation creates a new invitation
func HandleAdminCreateInvitation(s AdminServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/admin/invitations/new", http.StatusSeeOther)
			return
		}

		_, userName := s.GetCurrentUser(r)
		themes := config.GetThemes()

		// Parse and validate form
		formData, ok := parseInvitationForm(r, w, userName, themes)
		if !ok {
			return
		}

		// Generate invite message template
		lang := i18n.GetLanguageFromRequest(r)
		inviteMessageTemplate := generateInviteMessageTemplate(s, formData.guestName, lang)

		// Create invitation and update message
		if !createInvitationWithMessage(s, formData, inviteMessageTemplate, w, r, userName, themes) {
			return
		}

		// Redirect to list
		http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
	}
}

// HandleAdminMarkSent marks an invitation as sent
func HandleAdminMarkSent(s AdminServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseFormID(r, w)
		if !ok {
			return
		}

		if err := s.GetDB().MarkAsSent(id); err != nil {
			http.Error(w, "Failed to mark as sent", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
	}
}

// HandleAdminEditInvitation shows the edit invitation form
func HandleAdminEditInvitation(s AdminServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, userName := s.GetCurrentUser(r)
		themes := config.GetThemes()

		// Extract ID from URL path
		idStr := r.URL.Path[len("/admin/invitations/edit/"):]
		id, err := parseID(idStr)
		if err != nil {
			http.Error(w, "Invalid invitation ID", http.StatusBadRequest)
			return
		}

		invitation, err := s.GetDB().GetInvitationByID(id)
		if err != nil {
			http.Error(w, "Invitation not found", http.StatusNotFound)
			return
		}

		if err := templates.AdminEditInvitation(userName, invitation, "", themes.Light, themes.Dark).Render(r.Context(), w); err != nil {
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
	}
}

// HandleAdminUpdateInvitation updates an existing invitation
func HandleAdminUpdateInvitation(s AdminServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
			return
		}

		_, userName := s.GetCurrentUser(r)
		themes := config.GetThemes()

		// Extract ID from URL path
		idStr := r.URL.Path[len("/admin/invitations/update/"):]
		id, err := parseID(idStr)
		if err != nil {
			http.Error(w, "Invalid invitation ID", http.StatusBadRequest)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form", http.StatusBadRequest)
			return
		}

		guestName := strings.TrimSpace(r.FormValue("guest_name"))
		phone := strings.TrimSpace(r.FormValue("phone"))

		if guestName == "" || phone == "" {
			invitation, _ := s.GetDB().GetInvitationByID(id)
			_ = templates.AdminEditInvitation(userName, invitation, "Toate câmpurile sunt obligatorii", themes.Light, themes.Dark).Render(r.Context(), w)
			return
		}

		// Normalize phone number to E.164 format
		normalizedPhone, err := utils.NormalizePhoneNumber(phone)
		if err != nil {
			invitation, _ := s.GetDB().GetInvitationByID(id)
			_ = templates.AdminEditInvitation(userName, invitation, "Număr de telefon invalid", themes.Light, themes.Dark).Render(r.Context(), w)
			return
		}
		phone = normalizedPhone

		if err := s.GetDB().UpdateInvitation(id, guestName, phone); err != nil {
			invitation, _ := s.GetDB().GetInvitationByID(id)
			_ = templates.AdminEditInvitation(userName, invitation, "Eroare la actualizare. Verifică dacă numărul de telefon nu este deja folosit.", themes.Light, themes.Dark).Render(r.Context(), w)
			return
		}

		http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
	}
}

// HandleAdminDeleteInvitation deletes an invitation
func HandleAdminDeleteInvitation(s AdminServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseFormID(r, w)
		if !ok {
			return
		}

		if err := s.GetDB().DeleteInvitation(id); err != nil {
			http.Error(w, "Failed to delete invitation", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
	}
}
