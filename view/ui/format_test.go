package ui

import "testing"

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name, first, username, want string
	}{
		{"first name wins", "John", "user3", "John"},
		{"blank first name falls back to username", "", "user3", "user3"},
		{"whitespace-only first name falls back to username", "   ", "user3", "user3"},
		{"first name is trimmed", "  John  ", "user3", "John"},
		{"unicode first name", "José", "user3", "José"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayName(tt.first, tt.username); got != tt.want {
				t.Errorf("DisplayName(%q, %q) = %q, want %q", tt.first, tt.username, got, tt.want)
			}
		})
	}
}

func TestInitials(t *testing.T) {
	tests := []struct {
		name, first, last, want string
	}{
		{"both names", "John", "Johnson", "JJ"},
		{"both blank", "", "", "?"},
		{"lowercase input is uppercased", "john", "johnson", "JJ"},
		{"single name, last blank", "Sam", "", "SS"},
		{"single name, first blank", "", "Reyes", "RR"},
		{"whitespace-only counts as blank", "  ", "  ", "?"},
		{"unicode names", "Ólafur", "Éowyn", "ÓÉ"},
		{"extra whitespace is trimmed", "  John  ", "  Johnson  ", "JJ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Initials(tt.first, tt.last); got != tt.want {
				t.Errorf("Initials(%q, %q) = %q, want %q", tt.first, tt.last, got, tt.want)
			}
		})
	}
}
