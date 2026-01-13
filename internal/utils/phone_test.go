package utils

import (
	"testing"
)

func TestNormalizePhoneNumber(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		shouldError bool
	}{
		{
			name:        "Romanian mobile with country code",
			input:       "+40721234567",
			expected:    "+40721234567",
			shouldError: false,
		},
		{
			name:        "Romanian mobile without country code",
			input:       "0721234567",
			expected:    "+40721234567",
			shouldError: false,
		},
		{
			name:        "Romanian mobile with spaces",
			input:       "0721 234 567",
			expected:    "+40721234567",
			shouldError: false,
		},
		{
			name:        "Romanian mobile with dashes",
			input:       "0721-234-567",
			expected:    "+40721234567",
			shouldError: false,
		},
		{
			name:        "Romanian mobile with leading/trailing spaces",
			input:       "  0721234567  ",
			expected:    "+40721234567",
			shouldError: false,
		},
		{
			name:        "Romanian landline Bucharest",
			input:       "0211234567",
			expected:    "+40211234567",
			shouldError: false,
		},
		{
			name:        "International format with country code",
			input:       "+40 721 234 567",
			expected:    "+40721234567",
			shouldError: false,
		},
		{
			name:        "Invalid phone number - too short",
			input:       "123",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "Invalid phone number - letters",
			input:       "abcdefghij",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    "",
			shouldError: true,
		},
		// German phone numbers
		{
			name:        "German mobile with country code",
			input:       "+491701234567",
			expected:    "+491701234567",
			shouldError: false,
		},
		{
			name:        "German mobile without country code",
			input:       "01701234567",
			expected:    "", // Invalid - will be parsed as Romanian but not a valid RO number
			shouldError: true,
		},
		{
			name:        "German mobile with spaces",
			input:       "+49 170 1234567",
			expected:    "+491701234567",
			shouldError: false,
		},
		{
			name:        "German landline Berlin",
			input:       "+49 30 12345678",
			expected:    "+493012345678",
			shouldError: false,
		},
		{
			name:        "German mobile with dashes",
			input:       "+49-170-1234567",
			expected:    "+491701234567",
			shouldError: false,
		},
		// Irish phone numbers
		{
			name:        "Irish mobile with country code",
			input:       "+353871234567",
			expected:    "+353871234567",
			shouldError: false,
		},
		{
			name:        "Irish mobile without country code",
			input:       "0871234567",
			expected:    "", // Invalid - will be parsed as Romanian but not a valid RO number
			shouldError: true,
		},
		{
			name:        "Irish mobile with spaces",
			input:       "+353 87 123 4567",
			expected:    "+353871234567",
			shouldError: false,
		},
		{
			name:        "Irish landline Dublin",
			input:       "+353 1 1234567",
			expected:    "+35311234567",
			shouldError: false,
		},
		{
			name:        "Irish mobile with parentheses",
			input:       "+353 (87) 123 4567",
			expected:    "+353871234567",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizePhoneNumber(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("For input %q, expected %q but got %q", tt.input, tt.expected, result)
				}
			}
		})
	}
}
