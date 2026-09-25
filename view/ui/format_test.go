package ui

import (
	"testing"
	"time"
)

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

func TestGreeting(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{0, "Good morning"},
		{9, "Good morning"},
		{11, "Good morning"},
		{12, "Good afternoon"},
		{17, "Good afternoon"},
		{18, "Good evening"},
		{23, "Good evening"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := Greeting(tt.hour); got != tt.want {
				t.Errorf("Greeting(%d) = %q, want %q", tt.hour, got, tt.want)
			}
		})
	}
}

func TestLongDate(t *testing.T) {
	got := LongDate(time.Date(2026, time.September, 25, 10, 0, 0, 0, time.UTC))
	want := "Friday, 25 September"
	if got != want {
		t.Errorf("LongDate(...) = %q, want %q", got, want)
	}

	got = LongDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	want = "Sunday, 1 March"
	if got != want {
		t.Errorf("LongDate(...) = %q, want %q", got, want)
	}
}

func TestPct(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{-1, "—"},
		{0, "0%"},
		{60, "60%"},
		{100, "100%"},
	}
	for _, tt := range tests {
		if got := Pct(tt.n); got != tt.want {
			t.Errorf("Pct(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestRelTime(t *testing.T) {
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"just happened", 0, "just now"},
		{"59 seconds", 59 * time.Second, "just now"},
		{"1 minute", time.Minute, "1 min ago"},
		{"1 minute 59 seconds truncates down", time.Minute + 59*time.Second, "1 min ago"},
		{"8 minutes", 8 * time.Minute, "8 min ago"},
		{"59 minutes", 59 * time.Minute, "59 min ago"},
		{"1 hour", time.Hour, "1 hr ago"},
		{"3 hours", 3 * time.Hour, "3 hr ago"},
		{"23 hours", 23 * time.Hour, "23 hr ago"},
		{"1 day", 24 * time.Hour, "1d ago"},
		{"6 days", 6 * 24 * time.Hour, "6d ago"},
		{"future timestamp (clock skew) reads as just now", -time.Minute, "just now"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RelTime(now, now.Add(-tt.ago)); got != tt.want {
				t.Errorf("RelTime(now, now-%v) = %q, want %q", tt.ago, got, tt.want)
			}
		})
	}

	got := RelTime(now, now.Add(-8*24*time.Hour))
	want := "17 Sep"
	if got != want {
		t.Errorf("RelTime(now, now-8d) = %q, want %q", got, want)
	}
}
