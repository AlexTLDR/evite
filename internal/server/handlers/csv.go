package handlers

import (
	"database/sql"
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

// formatYesNo converts a boolean to "Da"/"Nu"
func formatYesNo(value bool) string {
	if value {
		return "Da"
	}
	return "Nu"
}

// formatNullableString returns the string value or a default if null/empty
func formatNullableString(ns sql.NullString, defaultValue string) string {
	if ns.Valid && ns.String != "" {
		return ns.String
	}
	return defaultValue
}

// formatResponseData extracts and formats all response fields
func formatResponseData(response *database.Response) (attending, plusOne, kidsCount, menuPref, companionMenuPref, comment string) {
	// Default values when no response
	if response == nil {
		return "-", "-", "-", "-", "-", "-"
	}

	// Format boolean fields
	attending = formatYesNo(response.Attending)
	plusOne = formatYesNo(response.PlusOne)

	// Format kids count
	if response.KidsCount > 0 {
		kidsCount = fmt.Sprintf("%d", response.KidsCount)
	} else {
		kidsCount = "0"
	}

	// Format menu preferences
	menuPref = formatNullableString(response.MenuPreference, "-")
	companionMenuPref = formatNullableString(response.CompanionMenuPreference, "-")

	// Format comment with escaping
	if response.Comment.Valid {
		comment = escapeCSVField(response.Comment.String)
	} else {
		comment = "-"
	}

	return
}

// formatInvitationForCSV converts an invitation to CSV row data
func formatInvitationForCSV(inv *database.InvitationWithResponse) csvRowData {
	row := csvRowData{
		name:      escapeCSVField(inv.GuestName),
		phone:     escapeCSVField(inv.Phone),
		sent:      formatYesNo(inv.SentAt.Valid),
		opened:    formatYesNo(inv.OpenedAt.Valid),
		responded: formatYesNo(inv.RespondedAt.Valid),
	}

	// Format response data
	row.attending, row.plusOne, row.kidsCount,
		row.menuPreference, row.companionMenuPreference,
		row.comment = formatResponseData(inv.Response)

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
