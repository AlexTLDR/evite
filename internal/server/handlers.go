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

func (s *Server) handleRSVPSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	lang := i18n.GetLanguageFromRequest(r)

	// Check if the RSVP deadline has passed
	if time.Now().After(s.config.RSVPDeadline) {
		errorMsg := "RSVP deadline has passed"
		if lang == "ro" {
			errorMsg = "Termenul limită pentru confirmare a trecut"
		}
		http.Error(w, errorMsg, http.StatusForbidden)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get form values
	token := r.FormValue("token")
	attending := r.FormValue("attending") == "yes"
	guestName := strings.TrimSpace(r.FormValue("guest_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	hasPartner := r.FormValue("has_partner") == "true"
	kidsCountStr := r.FormValue("kids_count")
	menuPreference := strings.TrimSpace(r.FormValue("menu_preference"))
	companionMenuPreference := strings.TrimSpace(r.FormValue("companion_menu_preference"))
	comment := strings.TrimSpace(r.FormValue("comment"))

	// Validate required fields
	if guestName == "" || phone == "" {
		http.Error(w, "Name and phone are required", http.StatusBadRequest)
		return
	}

	// Normalize phone number to E.164 format
	normalizedPhone, err := utils.NormalizePhoneNumber(phone)
	if err != nil {
		errorMsg := "Invalid phone number format"
		if lang == "ro" {
			errorMsg = "Număr de telefon invalid"
		}
		http.Error(w, errorMsg, http.StatusBadRequest)
		return
	}
	phone = normalizedPhone

	// Parse kids count
	kidsCount := 0
	if kidsCountStr != "" {
		fmt.Sscanf(kidsCountStr, "%d", &kidsCount)
	}

	// Get invitation by token
	var invitationID int64
	if token != "" {
		invitation, err := s.db.GetInvitationByToken(token)
		if err != nil {
			http.Error(w, "Invalid invitation token", http.StatusBadRequest)
			return
		}
		invitationID = invitation.ID
	} else {
		// If no token, create a new invitation
		invitation, err := s.db.CreateInvitation(guestName, phone, "")
		if err != nil {
			http.Error(w, "Failed to create invitation", http.StatusInternalServerError)
			return
		}
		invitationID = invitation.ID
	}

	// Create response
	plusOneName := ""
	if hasPartner {
		plusOneName = "Partner" // We don't collect partner name in this form
	}

	_, respErr := s.db.CreateResponse(invitationID, attending, hasPartner, plusOneName, guestName, kidsCount, menuPreference, companionMenuPreference, comment)
	if respErr != nil {
		http.Error(w, "Failed to save response", http.StatusInternalServerError)
		return
	}

	// Redirect to thank you page with language
	redirectURL := "/?submitted=true&lang=" + string(lang)
	if token != "" {
		redirectURL += "&token=" + token
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

func (s *Server) handleAdminDownloadCSV(w http.ResponseWriter, r *http.Request) {
	invitations, err := s.db.GetAllInvitationsWithResponses()
	if err != nil {
		http.Error(w, "Failed to load invitations", http.StatusInternalServerError)
		return
	}

	// Set CSV headers
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=rsvp-list.csv")

	// Write UTF-8 BOM for Excel compatibility
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	// Write CSV header
	w.Write([]byte("Nume,Telefon,Trimis,Deschis,Răspuns,Participă,+1,Copii,Meniu,Meniu Însoțitor,Mesaj\n"))

	// Write data rows
	for _, inv := range invitations {
		// Escape CSV fields
		name := strings.ReplaceAll(inv.GuestName, "\"", "\"\"")
		phone := strings.ReplaceAll(inv.Phone, "\"", "\"\"")

		sent := "Nu"
		if inv.SentAt.Valid {
			sent = "Da"
		}

		opened := "Nu"
		if inv.OpenedAt.Valid {
			opened = "Da"
		}

		responded := "Nu"
		if inv.RespondedAt.Valid {
			responded = "Da"
		}

		attending := "-"
		plusOne := "-"
		kidsCount := "-"
		menuPreference := "-"
		companionMenuPreference := "-"
		comment := "-"

		if inv.Response != nil {
			if inv.Response.Attending {
				attending = "Da"
			} else {
				attending = "Nu"
			}

			if inv.Response.PlusOne {
				plusOne = "Da"
			} else {
				plusOne = "Nu"
			}

			if inv.Response.KidsCount > 0 {
				kidsCount = fmt.Sprintf("%d", inv.Response.KidsCount)
			} else {
				kidsCount = "0"
			}

			if inv.Response.MenuPreference.Valid && inv.Response.MenuPreference.String != "" {
				menuPreference = inv.Response.MenuPreference.String
			}

			if inv.Response.CompanionMenuPreference.Valid && inv.Response.CompanionMenuPreference.String != "" {
				companionMenuPreference = inv.Response.CompanionMenuPreference.String
			}

			if inv.Response.Comment.Valid {
				comment = strings.ReplaceAll(inv.Response.Comment.String, "\"", "\"\"")
				comment = strings.ReplaceAll(comment, "\n", " ")
			}
		}

		line := fmt.Sprintf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n",
			name, phone, sent, opened, responded, attending, plusOne, kidsCount, menuPreference, companionMenuPreference, comment)
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
