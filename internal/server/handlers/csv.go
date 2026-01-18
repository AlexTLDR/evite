package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/AlexTLDR/evite/internal/database"
)

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

// HandleAdminDownloadCSV exports invitations to CSV
func HandleAdminDownloadCSV(s Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invitations, err := s.GetDB().GetAllInvitationsWithResponses()
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
}
