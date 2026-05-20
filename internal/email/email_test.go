package email

import (
	"testing"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"invalid-email", false},
		{"", false},
		{"a@b.c", false},
		{"user@domain.co.uk", false},
		{"user@invalid-domain", false},
		{"user@outlook.com", true},
	}

	for _, tt := range tests {
		actual := IsValidEmail(tt.email)
		if actual != tt.expected {
			t.Errorf("IsValidEmail(%q) = %v, want %v", tt.email, actual, tt.expected)
		}
	}
}
