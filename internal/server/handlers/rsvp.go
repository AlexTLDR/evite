package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AlexTLDR/evite/internal/i18n"
	"github.com/AlexTLDR/evite/internal/utils"
)

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
func checkRSVPDeadline(s Server, w http.ResponseWriter, lang i18n.Language) bool {
	if time.Now().After(s.GetConfig().RSVPDeadline) {
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
func normalizeAndValidatePhone(phone string, w http.ResponseWriter, lang i18n.Language) (string, bool) {
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
func parseRSVPForm(r *http.Request, w http.ResponseWriter, lang i18n.Language) (*rsvpFormData, bool) {
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
	normalizedPhone, ok := normalizeAndValidatePhone(phone, w, lang)
	if !ok {
		return nil, false
	}

	// Parse kids count
	kidsCount := 0
	kidsCountStr := r.FormValue("kids_count")
	if kidsCountStr != "" {
		// Ignore error - default to 0 if parsing fails
		fmt.Sscanf(kidsCountStr, "%d", &kidsCount)
		// Ensure non-negative
		if kidsCount < 0 {
			kidsCount = 0
		}
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
func getOrCreateInvitation(s Server, formData *rsvpFormData, w http.ResponseWriter) (int64, bool) {
	if formData.token != "" {
		invitation, err := s.GetDB().GetInvitationByToken(formData.token)
		if err != nil {
			http.Error(w, "Invalid invitation token", http.StatusBadRequest)
			return 0, false
		}
		return invitation.ID, true
	}

	// If no token, create a new invitation
	invitation, err := s.GetDB().CreateInvitation(formData.guestName, formData.phone, "")
	if err != nil {
		http.Error(w, "Failed to create invitation", http.StatusInternalServerError)
		return 0, false
	}
	return invitation.ID, true
}

// HandleRSVPSubmit processes RSVP form submissions
func HandleRSVPSubmit(s Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		lang := i18n.GetLanguageFromRequest(r)

		// Check if the RSVP deadline has passed
		if !checkRSVPDeadline(s, w, lang) {
			return
		}

		// Parse and validate form data
		formData, ok := parseRSVPForm(r, w, lang)
		if !ok {
			return
		}

		// Get or create invitation
		invitationID, ok := getOrCreateInvitation(s, formData, w)
		if !ok {
			return
		}

		// Create response
		plusOneName := ""
		if formData.hasPartner {
			plusOneName = "Partner" // We don't collect partner name in this form
		}

		_, err := s.GetDB().CreateResponse(
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
}
