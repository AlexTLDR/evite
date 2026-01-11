package i18n

import (
	"net/http"
)

type Language string

const (
	Romanian Language = "ro"
	English  Language = "en"
)

var translations = map[Language]map[string]string{
	Romanian: {
		// Common
		"app.title":       "Invitație Botez",
		"language.toggle": "English",
		"yes":             "Da",
		"no":              "Nu",
		"submit":          "Trimite",
		"cancel":          "Anulează",
		"save":            "Salvează",
		"delete":          "Șterge",
		"edit":            "Editează",
		"back":            "Înapoi",
		"loading":         "Se încarcă...",

		// Admin - Navigation
		"admin.title":       "Panou Administrare",
		"admin.invitations": "Invitații",
		"admin.dashboard":   "Tablou de Bord",
		"admin.logout":      "Deconectare",

		// Admin - Invitations List
		"invitations.list.title":     "Lista Invitații",
		"invitations.list.new":       "Invitație Nouă",
		"invitations.list.guest":     "Invitat",
		"invitations.list.phone":     "Telefon",
		"invitations.list.status":    "Status",
		"invitations.list.actions":   "Acțiuni",
		"invitations.list.copy":      "Copiază Mesaj",
		"invitations.list.mark_sent": "Marchează ca Trimis",
		"invitations.list.sent":      "Trimis",
		"invitations.list.opened":    "Deschis",
		"invitations.list.responded": "Răspuns",
		"invitations.list.not_sent":  "Netrimis",
		"invitations.list.copied":    "Mesaj copiat!",

		// Admin - New Invitation
		"invitations.new.title":      "Invitație Nouă",
		"invitations.new.guest_name": "Nume Invitat",
		"invitations.new.phone":      "Telefon",
		"invitations.new.create":     "Creează Invitație",
		"invitations.new.success":    "Invitație creată cu succes!",

		// RSVP - Form
		"rsvp.title":              "Confirmă Prezența",
		"rsvp.greeting":           "Bună {{.Name}},",
		"rsvp.event_details":      "Fiica noastră se botează 🎉",
		"rsvp.event_date":         "Evenimentul va avea loc pe 19 Aprilie 2026",
		"rsvp.church":             "Biserica: {{.ChurchName}}",
		"rsvp.restaurant":         "Restaurant: {{.RestaurantName}}",
		"rsvp.attending":          "Vei participa?",
		"rsvp.name_tag":           "Numele pentru eticheta de masă",
		"rsvp.name_tag_help":      "Cum vrei să apară numele tău pe eticheta de la masă?",
		"rsvp.plus_one":           "Vii cu cineva?",
		"rsvp.plus_one_name":      "Numele persoanei însoțitoare",
		"rsvp.plus_one_name_help": "Pentru eticheta de masă",
		"rsvp.kids_count":         "Număr copii",
		"rsvp.comment":            "Comentarii (opțional)",
		"rsvp.submit":             "Confirmă Prezența",
		"rsvp.deadline_passed":    "Termenul limită pentru confirmări a trecut.",
		"rsvp.invalid_token":      "Link invalid.",
		"rsvp.already_responded":  "Ai răspuns deja. Mulțumim!",
		"rsvp.can_edit":           "Poți modifica răspunsul până la {{.Deadline}}",
		"rsvp.thank_you":          "Mulțumim pentru confirmare!",
		"rsvp.see_you":            "Ne vedem la eveniment!",

		// Dashboard
		"dashboard.title":         "Tablou de Bord",
		"dashboard.total_invites": "Total Invitații",
		"dashboard.sent":          "Trimise",
		"dashboard.opened":        "Deschise",
		"dashboard.responded":     "Răspunsuri",
		"dashboard.attending":     "Participă",
		"dashboard.not_attending": "Nu Participă",
		"dashboard.total_guests":  "Total Invitați",
		"dashboard.adults":        "Adulți",
		"dashboard.kids":          "Copii",

		// Errors
		"error.required":        "Acest câmp este obligatoriu",
		"error.invalid_phone":   "Număr de telefon invalid",
		"error.duplicate_phone": "Acest număr de telefon există deja",
		"error.server":          "A apărut o eroare. Te rugăm să încerci din nou.",

		// Landing Page
		"landing.invitation_text": "Anya vă invită la botezul ei în data de 19 Aprilie 2026",
	},
	English: {
		// Common
		"app.title":       "Baptism Invitation",
		"language.toggle": "Română",
		"yes":             "Yes",
		"no":              "No",
		"submit":          "Submit",
		"cancel":          "Cancel",
		"save":            "Save",
		"delete":          "Delete",
		"edit":            "Edit",
		"back":            "Back",
		"loading":         "Loading...",

		// Admin - Navigation
		"admin.title":       "Admin Panel",
		"admin.invitations": "Invitations",
		"admin.dashboard":   "Dashboard",
		"admin.logout":      "Logout",

		// Admin - Invitations List
		"invitations.list.title":     "Invitations List",
		"invitations.list.new":       "New Invitation",
		"invitations.list.guest":     "Guest",
		"invitations.list.phone":     "Phone",
		"invitations.list.status":    "Status",
		"invitations.list.actions":   "Actions",
		"invitations.list.copy":      "Copy Message",
		"invitations.list.mark_sent": "Mark as Sent",
		"invitations.list.sent":      "Sent",
		"invitations.list.opened":    "Opened",
		"invitations.list.responded": "Responded",
		"invitations.list.not_sent":  "Not Sent",
		"invitations.list.copied":    "Message copied!",

		// Admin - New Invitation
		"invitations.new.title":      "New Invitation",
		"invitations.new.guest_name": "Guest Name",
		"invitations.new.phone":      "Phone",
		"invitations.new.create":     "Create Invitation",
		"invitations.new.success":    "Invitation created successfully!",

		// RSVP - Form
		"rsvp.title":              "RSVP",
		"rsvp.greeting":           "Hi {{.Name}},",
		"rsvp.event_details":      "Our daughter is getting baptised 🎉",
		"rsvp.event_date":         "The event will take place on April 19, 2026",
		"rsvp.church":             "Church: {{.ChurchName}}",
		"rsvp.restaurant":         "Restaurant: {{.RestaurantName}}",
		"rsvp.attending":          "Will you attend?",
		"rsvp.name_tag":           "Name for table tag",
		"rsvp.name_tag_help":      "How would you like your name to appear on the table tag?",
		"rsvp.plus_one":           "Bringing someone?",
		"rsvp.plus_one_name":      "Guest's name",
		"rsvp.plus_one_name_help": "For the table tag",
		"rsvp.kids_count":         "Number of kids",
		"rsvp.comment":            "Comments (optional)",
		"rsvp.submit":             "Confirm Attendance",
		"rsvp.deadline_passed":    "The RSVP deadline has passed.",
		"rsvp.invalid_token":      "Invalid link.",
		"rsvp.already_responded":  "You have already responded. Thank you!",
		"rsvp.can_edit":           "You can edit your response until {{.Deadline}}",
		"rsvp.thank_you":          "Thank you for your response!",
		"rsvp.see_you":            "See you at the event!",

		// Dashboard
		"dashboard.title":         "Dashboard",
		"dashboard.total_invites": "Total Invitations",
		"dashboard.sent":          "Sent",
		"dashboard.opened":        "Opened",
		"dashboard.responded":     "Responses",
		"dashboard.attending":     "Attending",
		"dashboard.not_attending": "Not Attending",
		"dashboard.total_guests":  "Total Guests",
		"dashboard.adults":        "Adults",
		"dashboard.kids":          "Kids",

		// Errors
		"error.required":        "This field is required",
		"error.invalid_phone":   "Invalid phone number",
		"error.duplicate_phone": "This phone number already exists",
		"error.server":          "An error occurred. Please try again.",

		// Landing Page
		"landing.invitation_text": "Anya invites you to her baptism on April 19, 2026",
	},
}

// T translates a key for the given language
func T(lang Language, key string) string {
	if trans, ok := translations[lang][key]; ok {
		return trans
	}
	// Fallback to English
	if trans, ok := translations[English][key]; ok {
		return trans
	}
	return key
}

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
