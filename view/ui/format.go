package ui

import (
	"strings"
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

func firstRuneUpper(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r, _ := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r))
}
