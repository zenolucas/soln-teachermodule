package ui

import (
	"fmt"
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

// Pct formats a percentage computed elsewhere (-1 meaning "no data", per DEC-20's
// convention throughout the insight types) as "—" rather than a misleading "-1%".
func Pct(n int) string {
	if n < 0 {
		return "—"
	}
	return fmt.Sprintf("%d%%", n)
}

// RelTime renders how long ago t was, relative to now, for the Recent activity card
// (01 §1a): "just now", "{n} min ago", "{n} hr ago", "{n}d ago", and a bare date once
// it's a week or more old. now before t (clock skew, or a just-inserted row) reads as
// "just now" rather than a negative duration.
func RelTime(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hr ago", int(d/time.Hour))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	default:
		return t.Format("2 Jan")
	}
}

func firstRuneUpper(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r, _ := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r))
}
