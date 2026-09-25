package ui

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// DisplayName returns the trimmed first name when it's non-empty (not just
// whitespace), otherwise the username.
func DisplayName(first, username string) string {
	if trimmed := strings.TrimSpace(first); trimmed != "" {
		return trimmed
	}
	return username
}

// Initials returns the uppercased first rune of first and last, joined - "JJ" for
// ("John", "Johnson"). "?" when both are blank. When only one name is given, its
// initial is doubled up to keep the avatar's two-character shape, e.g. Initials("Sam",
// "") is "SS".
func Initials(first, last string) string {
	f := firstRuneUpper(first)
	l := firstRuneUpper(last)
	switch {
	case f == "" && l == "":
		return "?"
	case f == "":
		return l + l
	case l == "":
		return f + f
	default:
		return f + l
	}
}

// Greeting returns "Good morning"/"Good afternoon"/"Good evening" for the given local
// hour (0-23), per DEC-16: morning is before 12, afternoon before 18, otherwise
// evening.
func Greeting(hour int) string {
	switch {
	case hour < 12:
		return "Good morning"
	case hour < 18:
		return "Good afternoon"
	default:
		return "Good evening"
	}
}

// LongDate formats t as "Monday, 25 September" (DEC-16) - weekday and day, no year.
func LongDate(t time.Time) string {
	return t.Format("Monday, 2 January")
}

func firstRuneUpper(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r, _ := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r))
}
