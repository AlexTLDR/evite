package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AlexTLDR/evite/internal/config"
	"github.com/AlexTLDR/evite/internal/database"
	"github.com/AlexTLDR/evite/internal/i18n"
	"github.com/AlexTLDR/evite/internal/utils"
	"github.com/AlexTLDR/evite/templates"
)

// Public handlers
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	lang := i18n.GetLanguageFromRequest(r)
	themes := config.GetThemes()

	// Check if there's a token in the query parameter
	token := r.URL.Query().Get("token")

	var invitation *database.Invitation
	if token != "" {
		// Try to get invitation by token
		inv, err := s.db.GetInvitationByToken(token)
		if err == nil {
			invitation = inv
			// Mark as opened if not already
			if !invitation.OpenedAt.Valid {
				if err := s.db.MarkAsOpened(invitation.ID); err != nil {
					// Log but don't fail - this is just tracking
					fmt.Printf("Warning: failed to mark invitation as opened: %v\n", err)
				}
			}
		}
	}

	// Check if the RSVP deadline has passed
	now := time.Now()
	deadlinePassed := now.After(s.config.RSVPDeadline)

	// Debug logging
	fmt.Printf("DEBUG: Current time: %v\n", now)
	fmt.Printf("DEBUG: RSVP Deadline: %v\n", s.config.RSVPDeadline)
	fmt.Printf("DEBUG: Deadline passed: %v\n", deadlinePassed)

	if err := templates.Home(string(lang), themes.Light, themes.Dark, invitation, deadlinePassed).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleRSVP(w http.ResponseWriter, r *http.Request) {
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

// rsvpFormData holds the parsed and validated form data
type rsvpFormData struct {
	token                   string
	attending               bool
	guestName               string
	phone                   string
	hasPartner              bool
	kidsCount               int
	menuPreference          string
	companionMenuPreference string
	comment                 string
}

// checkRSVPDeadline validates if the RSVP deadline has passed
func (s *Server) checkRSVPDeadline(w http.ResponseWriter, lang i18n.Language) bool {
	if time.Now().After(s.config.RSVPDeadline) {
		errorMsg := "RSVP deadline has passed"
		if lang == "ro" {
			errorMsg = "Termenul limită pentru confirmare a trecut"
		}
		http.Error(w, errorMsg, http.StatusForbidden)
		return false
	}
	return true
}

// normalizeAndValidatePhone normalizes a phone number to E.164 format
func (s *Server) normalizeAndValidatePhone(phone string, w http.ResponseWriter, lang i18n.Language) (string, bool) {
	normalizedPhone, err := utils.NormalizePhoneNumber(phone)
	if err != nil {
		errorMsg := "Invalid phone number format"
		if lang == "ro" {
			errorMsg = "Număr de telefon invalid"
		}
		http.Error(w, errorMsg, http.StatusBadRequest)
		return "", false
	}
	return normalizedPhone, true
}

// parseRSVPForm parses and validates the RSVP form data
func (s *Server) parseRSVPForm(r *http.Request, w http.ResponseWriter, lang i18n.Language) (*rsvpFormData, bool) {
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return nil, false
	}

	// Get form values
	guestName := strings.TrimSpace(r.FormValue("guest_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))

	// Validate required fields
	if guestName == "" || phone == "" {
		http.Error(w, "Name and phone are required", http.StatusBadRequest)
		return nil, false
	}

	// Normalize phone number
	normalizedPhone, ok := s.normalizeAndValidatePhone(phone, w, lang)
	if !ok {
		return nil, false
	}

	// Parse kids count
	kidsCount := 0
	kidsCountStr := r.FormValue("kids_count")
	if kidsCountStr != "" {
		fmt.Sscanf(kidsCountStr, "%d", &kidsCount)
	}

	return &rsvpFormData{
		token:                   r.FormValue("token"),
		attending:               r.FormValue("attending") == "yes",
		guestName:               guestName,
		phone:                   normalizedPhone,
		hasPartner:              r.FormValue("has_partner") == "true",
		kidsCount:               kidsCount,
		menuPreference:          strings.TrimSpace(r.FormValue("menu_preference")),
		companionMenuPreference: strings.TrimSpace(r.FormValue("companion_menu_preference")),
		comment:                 strings.TrimSpace(r.FormValue("comment")),
	}, true
}

// getOrCreateInvitation retrieves an existing invitation by token or creates a new one
func (s *Server) getOrCreateInvitation(formData *rsvpFormData, w http.ResponseWriter) (int64, bool) {
	if formData.token != "" {
		invitation, err := s.db.GetInvitationByToken(formData.token)
		if err != nil {
			http.Error(w, "Invalid invitation token", http.StatusBadRequest)
			return 0, false
		}
		return invitation.ID, true
	}

	// If no token, create a new invitation
	invitation, err := s.db.CreateInvitation(formData.guestName, formData.phone, "")
	if err != nil {
		http.Error(w, "Failed to create invitation", http.StatusInternalServerError)
		return 0, false
	}
	return invitation.ID, true
}

func (s *Server) handleRSVPSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	lang := i18n.GetLanguageFromRequest(r)

	// Check if the RSVP deadline has passed
	if !s.checkRSVPDeadline(w, lang) {
		return
	}

	// Parse and validate form data
	formData, ok := s.parseRSVPForm(r, w, lang)
	if !ok {
		return
	}

	// Get or create invitation
	invitationID, ok := s.getOrCreateInvitation(formData, w)
	if !ok {
		return
	}

	// Create response
	plusOneName := ""
	if formData.hasPartner {
		plusOneName = "Partner" // We don't collect partner name in this form
	}

	_, err := s.db.CreateResponse(
		invitationID,
		formData.attending,
		formData.hasPartner,
		plusOneName,
		formData.guestName,
		formData.kidsCount,
		formData.menuPreference,
		formData.companionMenuPreference,
		formData.comment,
	)
	if err != nil {
		http.Error(w, "Failed to save response", http.StatusInternalServerError)
		return
	}

	// Redirect to thank you page with language
	redirectURL := "/?submitted=true&lang=" + string(lang)
	if formData.token != "" {
		redirectURL += "&token=" + formData.token
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// Admin handlers
func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement admin dashboard
	email, name := s.getCurrentUser(r)
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

func (s *Server) handleAdminInvitations(w http.ResponseWriter, r *http.Request) {
	_, userName := s.getCurrentUser(r)

	invitations, err := s.db.GetAllInvitationsWithResponses()
	if err != nil {
		http.Error(w, "Failed to load invitations", http.StatusInternalServerError)
		return
	}

	themes := config.GetThemes()
	if err := templates.AdminInvitationsList(userName, invitations, themes.Light, themes.Dark).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminNewInvitation(w http.ResponseWriter, r *http.Request) {
	_, userName := s.getCurrentUser(r)
	themes := config.GetThemes()
	if err := templates.AdminNewInvitation(userName, "", themes.Light, themes.Dark).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminCreateInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/invitations/new", http.StatusSeeOther)
		return
	}

	_, userName := s.getCurrentUser(r)
	themes := config.GetThemes()

	// Parse form
	if err := r.ParseForm(); err != nil {
		_ = templates.AdminNewInvitation(userName, "Eroare la procesarea formularului", themes.Light, themes.Dark).Render(r.Context(), w)
		return
	}

	guestName := strings.TrimSpace(r.FormValue("guest_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))

	// Validate
	if guestName == "" || phone == "" {
		_ = templates.AdminNewInvitation(userName, "Toate câmpurile sunt obligatorii", themes.Light, themes.Dark).Render(r.Context(), w)
		return
	}

	// Normalize phone number to E.164 format
	normalizedPhone, err := utils.NormalizePhoneNumber(phone)
	if err != nil {
		_ = templates.AdminNewInvitation(userName, "Număr de telefon invalid", themes.Light, themes.Dark).Render(r.Context(), w)
		return
	}
	phone = normalizedPhone

	// Generate invite message with placeholder
	lang := i18n.GetLanguageFromRequest(r)
	inviteMessageTemplate := s.generateInviteMessageTemplate(guestName, lang)

	// Create invitation (this will generate the token)
	inv, err := s.db.CreateInvitation(guestName, phone, inviteMessageTemplate)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			_ = templates.AdminNewInvitation(userName, "Acest număr de telefon există deja", themes.Light, themes.Dark).Render(r.Context(), w)
			return
		}
		_ = templates.AdminNewInvitation(userName, "Eroare la crearea invitației", themes.Light, themes.Dark).Render(r.Context(), w)
		return
	}

	// Now update the message with the actual token
	rsvpLink := fmt.Sprintf("%s/rsvp/%s", s.config.BaseURL, inv.Token)
	finalMessage := strings.Replace(inviteMessageTemplate, "{{TOKEN}}", inv.Token, 1)
	finalMessage = strings.Replace(finalMessage, "{{RSVP_LINK}}", rsvpLink, 1)

	// Update the invitation with the final message
	_, err = s.db.Exec("UPDATE invitations SET invite_message = ? WHERE id = ?", finalMessage, inv.ID)
	if err != nil {
		// Log error but don't fail - the invitation is created
		fmt.Printf("Warning: failed to update invite message: %v\n", err)
	}

	// Redirect to list
	http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
}

func (s *Server) handleAdminMarkSent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	var id int64
	fmt.Sscanf(idStr, "%d", &id)

	if err := s.db.MarkAsSent(id); err != nil {
		http.Error(w, "Failed to mark as sent", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
}

func (s *Server) handleAdminEditInvitation(w http.ResponseWriter, r *http.Request) {
	_, userName := s.getCurrentUser(r)
	themes := config.GetThemes()

	// Extract ID from URL path
	idStr := r.URL.Path[len("/admin/invitations/edit/"):]
	var id int64
	fmt.Sscanf(idStr, "%d", &id)

	invitation, err := s.db.GetInvitationByID(id)
	if err != nil {
		http.Error(w, "Invitation not found", http.StatusNotFound)
		return
	}

	if err := templates.AdminEditInvitation(userName, invitation, "", themes.Light, themes.Dark).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminUpdateInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
		return
	}

	_, userName := s.getCurrentUser(r)
	themes := config.GetThemes()

	// Extract ID from URL path
	idStr := r.URL.Path[len("/admin/invitations/update/"):]
	var id int64
	fmt.Sscanf(idStr, "%d", &id)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	guestName := strings.TrimSpace(r.FormValue("guest_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))

	if guestName == "" || phone == "" {
		invitation, _ := s.db.GetInvitationByID(id)
		_ = templates.AdminEditInvitation(userName, invitation, "Toate câmpurile sunt obligatorii", themes.Light, themes.Dark).Render(r.Context(), w)
		return
	}

	// Normalize phone number to E.164 format
	normalizedPhone, err := utils.NormalizePhoneNumber(phone)
	if err != nil {
		invitation, _ := s.db.GetInvitationByID(id)
		_ = templates.AdminEditInvitation(userName, invitation, "Număr de telefon invalid", themes.Light, themes.Dark).Render(r.Context(), w)
		return
	}
	phone = normalizedPhone

	if err := s.db.UpdateInvitation(id, guestName, phone); err != nil {
		invitation, _ := s.db.GetInvitationByID(id)
		_ = templates.AdminEditInvitation(userName, invitation, "Eroare la actualizare. Verifică dacă numărul de telefon nu este deja folosit.", themes.Light, themes.Dark).Render(r.Context(), w)
		return
	}

	http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
}

// csvRowData holds formatted data for a single CSV row
type csvRowData struct {
	name                    string
	phone                   string
	sent                    string
	opened                  string
	responded               string
	attending               string
	plusOne                 string
	kidsCount               string
	menuPreference          string
	companionMenuPreference string
	comment                 string
}

// escapeCSVField escapes a string for CSV format
func escapeCSVField(field string) string {
	// Escape double quotes by doubling them
	escaped := strings.ReplaceAll(field, "\"", "\"\"")
	// Replace newlines with spaces for comment fields
	escaped = strings.ReplaceAll(escaped, "\n", " ")
	return escaped
}

// formatInvitationForCSV converts an invitation to CSV row data
func formatInvitationForCSV(inv *database.InvitationWithResponse) csvRowData {
	row := csvRowData{
		name:                    escapeCSVField(inv.GuestName),
		phone:                   escapeCSVField(inv.Phone),
		sent:                    "Nu",
		opened:                  "Nu",
		responded:               "Nu",
		attending:               "-",
		plusOne:                 "-",
		kidsCount:               "-",
		menuPreference:          "-",
		companionMenuPreference: "-",
		comment:                 "-",
	}

	// Format sent status
	if inv.SentAt.Valid {
		row.sent = "Da"
	}

	// Format opened status
	if inv.OpenedAt.Valid {
		row.opened = "Da"
	}

	// Format responded status
	if inv.RespondedAt.Valid {
		row.responded = "Da"
	}

	// Format response data if available
	if inv.Response != nil {
		if inv.Response.Attending {
			row.attending = "Da"
		} else {
			row.attending = "Nu"
		}

		if inv.Response.PlusOne {
			row.plusOne = "Da"
		} else {
			row.plusOne = "Nu"
		}

		if inv.Response.KidsCount > 0 {
			row.kidsCount = fmt.Sprintf("%d", inv.Response.KidsCount)
		} else {
			row.kidsCount = "0"
		}

		if inv.Response.MenuPreference.Valid && inv.Response.MenuPreference.String != "" {
			row.menuPreference = inv.Response.MenuPreference.String
		}

		if inv.Response.CompanionMenuPreference.Valid && inv.Response.CompanionMenuPreference.String != "" {
			row.companionMenuPreference = inv.Response.CompanionMenuPreference.String
		}

		if inv.Response.Comment.Valid {
			row.comment = escapeCSVField(inv.Response.Comment.String)
		}
	}

	return row
}

// buildCSVRow creates a CSV line from row data
func buildCSVRow(row csvRowData) string {
	return fmt.Sprintf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n",
		row.name, row.phone, row.sent, row.opened, row.responded,
		row.attending, row.plusOne, row.kidsCount,
		row.menuPreference, row.companionMenuPreference, row.comment)
}

// writeCSVHeaders sets HTTP headers and writes CSV header row
func writeCSVHeaders(w http.ResponseWriter) {
	// Set CSV headers
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=rsvp-list.csv")

	// Write UTF-8 BOM for Excel compatibility
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	// Write CSV header
	w.Write([]byte("Nume,Telefon,Trimis,Deschis,Răspuns,Participă,+1,Copii,Meniu,Meniu Însoțitor,Mesaj\n"))
}

func (s *Server) handleAdminDownloadCSV(w http.ResponseWriter, r *http.Request) {
	invitations, err := s.db.GetAllInvitationsWithResponses()
	if err != nil {
		http.Error(w, "Failed to load invitations", http.StatusInternalServerError)
		return
	}

	// Write CSV headers
	writeCSVHeaders(w)

	// Write data rows
	for _, inv := range invitations {
		row := formatInvitationForCSV(inv)
		line := buildCSVRow(row)
		w.Write([]byte(line))
	}
}

func (s *Server) handleAdminDeleteInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	idStr := r.FormValue("id")
	var id int64
	fmt.Sscanf(idStr, "%d", &id)

	if err := s.db.DeleteInvitation(id); err != nil {
		http.Error(w, "Failed to delete invitation", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
}

func (s *Server) generateInviteMessageTemplate(guestName string, lang i18n.Language) string {
	if lang == i18n.Romanian {
		return fmt.Sprintf(`Bună %s,

Fiica noastră se botează 🎉

Evenimentul va avea loc pe 19 Aprilie 2026:
- Biserica: %s
- Restaurant: %s

Te rugăm să confirmi prezența aici:
{{RSVP_LINK}}

Cu drag,
Familia`, guestName, s.config.ChurchName, s.config.RestaurantName)
	}

	return fmt.Sprintf(`Hi %s,

Our daughter is getting baptised 🎉

The event will take place on April 19, 2026:
- Church: %s
- Restaurant: %s

Please confirm your attendance here:
{{RSVP_LINK}}

With love,
The Family`, guestName, s.config.ChurchName, s.config.RestaurantName)
}
