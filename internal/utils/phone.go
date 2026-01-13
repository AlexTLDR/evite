package utils

import (
	"strings"

	"github.com/nyaruka/phonenumbers"
)

// NormalizePhoneNumber normalizes a phone number to E.164 format
// Assumes Romanian phone numbers if no country code is provided
func NormalizePhoneNumber(phone string) (string, error) {
	// Trim whitespace
	phone = strings.TrimSpace(phone)

	// Parse the phone number (default to Romania - RO)
	num, err := phonenumbers.Parse(phone, "RO")
	if err != nil {
		return "", err
	}

	// Validate the phone number
	if !phonenumbers.IsValidNumber(num) {
		return "", phonenumbers.ErrNotANumber
	}

	// Format to E.164 (e.g., +40721234567)
	return phonenumbers.Format(num, phonenumbers.E164), nil
}
