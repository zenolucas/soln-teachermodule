package util

import "testing"

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		wantValid bool
	}{
		{"letters only", "johndoe", true},
		{"letters, digits, underscore", "john_doe123", true},
		{"empty string", "", false},
		{"contains space", "john doe", false},
		{"contains hyphen", "john-doe", false},
		{"contains at-sign", "john@doe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, valid := ValidateUsername(tt.username)
			if valid != tt.wantValid {
				t.Errorf("ValidateUsername(%q) valid = %v, want %v", tt.username, valid, tt.wantValid)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantValid bool
	}{
		{"too short", "Ab1!", false},
		{"missing uppercase", "abcdefg1!", false},
		{"missing lowercase", "ABCDEFG1!", false},
		{"missing digit", "Abcdefgh!", false},
		{"missing special character", "Abcdefg1", false},
		{"valid password", "Abcdefg1!", true},
		// unicode.IsPunct('_') is true (Unicode classifies '_' as Pc, "Connector
		// Punctuation"), so it satisfies the special-character case even though it's
		// not one of the characters listed in specialRunes ("!@#$%^&*"). This isn't a
		// contradiction with the error message shown to the user, which already lists
		// "_" as an example special character - just documenting that the check is
		// broader than specialRunes alone would suggest.
		{"underscore counts as special via unicode.IsPunct", "Passw0rd_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, valid := ValidatePassword(tt.password)
			if valid != tt.wantValid {
				t.Errorf("ValidatePassword(%q) valid = %v, want %v", tt.password, valid, tt.wantValid)
			}
		})
	}
}
