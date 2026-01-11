package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/AlexTLDR/evite/internal/i18n"
	"github.com/AlexTLDR/evite/templates"
)

// Public handlers
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<title>Invitație Eveniment</title>
		</head>
		<body>
			<h1>Bine ai venit!</h1>
			<p>Pentru a accesa invitația, folosește link-ul primit pe WhatsApp.</p>
		</body>
		</html>
	`)
}

func (s *Server) handleRSVP(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement RSVP form
	w.Write([]byte("RSVP Form - Coming soon"))
}

func (s *Server) handleRSVPSubmit(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement RSVP submission
	w.Write([]byte("RSVP Submit - Coming soon"))
}

// Admin handlers
func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement admin dashboard
	email, name := s.getCurrentUser(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
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

	templates.AdminInvitationsList(userName, invitations).Render(r.Context(), w)
}

func (s *Server) handleAdminNewInvitation(w http.ResponseWriter, r *http.Request) {
	_, userName := s.getCurrentUser(r)
	templates.AdminNewInvitation(userName, "").Render(r.Context(), w)
}

func (s *Server) handleAdminCreateInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/invitations/new", http.StatusSeeOther)
		return
	}

	_, userName := s.getCurrentUser(r)

	// Parse form
	if err := r.ParseForm(); err != nil {
		templates.AdminNewInvitation(userName, "Eroare la procesarea formularului").Render(r.Context(), w)
		return
	}

	guestName := strings.TrimSpace(r.FormValue("guest_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))

	// Validate
	if guestName == "" || phone == "" {
		templates.AdminNewInvitation(userName, "Toate câmpurile sunt obligatorii").Render(r.Context(), w)
		return
	}

	// Generate invite message with placeholder
	lang := i18n.GetLanguageFromRequest(r)
	inviteMessageTemplate := s.generateInviteMessageTemplate(guestName, lang)

	// Create invitation (this will generate the token)
	inv, err := s.db.CreateInvitation(guestName, phone, inviteMessageTemplate)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			templates.AdminNewInvitation(userName, "Acest număr de telefon există deja").Render(r.Context(), w)
			return
		}
		templates.AdminNewInvitation(userName, "Eroare la crearea invitației").Render(r.Context(), w)
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

	// Extract ID from URL path
	idStr := r.URL.Path[len("/admin/invitations/edit/"):]
	var id int64
	fmt.Sscanf(idStr, "%d", &id)

	invitation, err := s.db.GetInvitationByID(id)
	if err != nil {
		http.Error(w, "Invitation not found", http.StatusNotFound)
		return
	}

	templates.AdminEditInvitation(userName, invitation, "").Render(r.Context(), w)
}

func (s *Server) handleAdminUpdateInvitation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
		return
	}

	_, userName := s.getCurrentUser(r)

	// Extract ID from URL path
	idStr := r.URL.Path[len("/admin/invitations/update/"):]
	var id int64
	fmt.Sscanf(idStr, "%d", &id)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	guestName := r.FormValue("guest_name")
	phone := r.FormValue("phone")

	if guestName == "" || phone == "" {
		invitation, _ := s.db.GetInvitationByID(id)
		templates.AdminEditInvitation(userName, invitation, "Toate câmpurile sunt obligatorii").Render(r.Context(), w)
		return
	}

	if err := s.db.UpdateInvitation(id, guestName, phone); err != nil {
		invitation, _ := s.db.GetInvitationByID(id)
		templates.AdminEditInvitation(userName, invitation, "Eroare la actualizare. Verifică dacă numărul de telefon nu este deja folosit.").Render(r.Context(), w)
		return
	}

	http.Redirect(w, r, "/admin/invitations", http.StatusSeeOther)
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
