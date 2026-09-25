package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Frac is a fraction, numerator over denominator. It's the pure-Go counterpart to the
// answer auto-compute index.js does client-side (02 §B4) - used server-side for
// misconception hints (T4.11) and anywhere else that needs to evaluate or compare a
// fraction question's answer (see DEC-13).
type Frac struct {
	Num int
	Den int
}

// String renders f as "3/4", or just the numerator ("2") when Den == 1, or "0" when
// Num == 0 (regardless of Den).
func (f Frac) String() string {
	if f.Num == 0 {
		return "0"
	}
	if f.Den == 1 {
		return strconv.Itoa(f.Num)
	}
	return fmt.Sprintf("%d/%d", f.Num, f.Den)
}

// gcd is Euclid's algorithm, always non-negative.
func gcd(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// Simplify reduces f to lowest terms, with any negative sign normalized onto the
// numerator (a positive denominator). Simplify(Frac{0, n}) is Frac{0, 1} for any
// nonzero n - "0" has no meaningful denominator to preserve.
func Simplify(f Frac) Frac {
	if f.Den < 0 {
		f.Num, f.Den = -f.Num, -f.Den
	}
	if f.Num == 0 {
		return Frac{0, 1}
	}
	if d := gcd(f.Num, f.Den); d != 0 {
		f.Num /= d
		f.Den /= d
	}
	return f
}

// Combine adds or subtracts b from a - op is "+", "-", or "−" (the minus sign
// Scene.Op uses for World 2, DEC-21) - and simplifies the result. false if either
// denominator is zero, or op isn't recognized.
func Combine(a, b Frac, op string) (Frac, bool) {
	if a.Den == 0 || b.Den == 0 {
		return Frac{}, false
	}
	num := a.Num * b.Den
	switch op {
	case "+":
		num += b.Num * a.Den
	case "-", "−":
		num -= b.Num * a.Den
	default:
		return Frac{}, false
	}
	return Simplify(Frac{num, a.Den * b.Den}), true
}

// Equal reports whether a and b represent the same value. It cross-multiplies rather
// than dividing, so it works correctly even when either is unsimplified or has a
// negative denominator - equality is preserved under cross-multiplication regardless
// of sign, unlike an ordering comparison would be.
func Equal(a, b Frac) bool {
	return a.Num*b.Den == b.Num*a.Den
}

// fracPattern is 02 §B6's exact regex for finding "n/d"-shaped fractions in question
// or choice text.
var fracPattern = regexp.MustCompile(`(\d+)\s*/\s*(\d+)`)

// ParseFracs finds every "n/d"-shaped fraction in text, in the order they appear.
func ParseFracs(text string) []Frac {
	matches := fracPattern.FindAllStringSubmatch(text, -1)
	fracs := make([]Frac, 0, len(matches))
	for _, m := range matches {
		num, _ := strconv.Atoi(m[1])
		den, _ := strconv.Atoi(m[2])
		fracs = append(fracs, Frac{num, den})
	}
	return fracs
}

// ParseChoice parses a single multiple-choice answer's text as a Frac: "n/d" as
// itself, or a bare integer ("1", "2") as n/1 (02 §B6). false if it's neither.
func ParseChoice(text string) (Frac, bool) {
	if fracs := ParseFracs(text); len(fracs) == 1 {
		return fracs[0], true
	}
	if n, err := strconv.Atoi(strings.TrimSpace(text)); err == nil {
		return Frac{n, 1}, true
	}
	return Frac{}, false
}
