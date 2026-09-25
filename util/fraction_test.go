package util

import "testing"

func TestCombine(t *testing.T) {
	tests := []struct {
		name   string
		a, b   Frac
		op     string
		want   Frac
		wantOK bool
	}{
		{"1/2 + 1/2 = 1", Frac{1, 2}, Frac{1, 2}, "+", Frac{1, 1}, true},
		{"7/10 + 1/5 = 9/10", Frac{7, 10}, Frac{1, 5}, "+", Frac{9, 10}, true},
		{"2/3 - 1/3 = 1/3", Frac{2, 3}, Frac{1, 3}, "-", Frac{1, 3}, true},
		{"1/4 - 3/4 = -1/2", Frac{1, 4}, Frac{3, 4}, "-", Frac{-1, 2}, true},
		{"minus sign U+2212 also works", Frac{2, 3}, Frac{1, 3}, "−", Frac{1, 3}, true},
		{"zero denominator on a returns false", Frac{1, 0}, Frac{1, 2}, "+", Frac{}, false},
		{"zero denominator on b returns false", Frac{1, 2}, Frac{1, 0}, "+", Frac{}, false},
		{"unrecognized op returns false", Frac{1, 2}, Frac{1, 2}, "*", Frac{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Combine(tt.a, tt.b, tt.op)
			if ok != tt.wantOK {
				t.Fatalf("Combine(%v, %v, %q) ok = %v, want %v", tt.a, tt.b, tt.op, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("Combine(%v, %v, %q) = %v, want %v", tt.a, tt.b, tt.op, got, tt.want)
			}
		})
	}
}

func TestSimplify(t *testing.T) {
	tests := []struct {
		name string
		in   Frac
		want Frac
	}{
		{"already simplified", Frac{1, 2}, Frac{1, 2}},
		{"reduces", Frac{2, 4}, Frac{1, 2}},
		{"negative denominator normalizes to numerator", Frac{1, -2}, Frac{-1, 2}},
		{"zero numerator", Frac{0, 5}, Frac{0, 1}},
		{"negative over negative", Frac{-2, -4}, Frac{1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Simplify(tt.in); got != tt.want {
				t.Errorf("Simplify(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b Frac
		want bool
	}{
		{"2/4 equals 1/2", Frac{2, 4}, Frac{1, 2}, true},
		{"1/2 does not equal 1/3", Frac{1, 2}, Frac{1, 3}, false},
		{"identical", Frac{3, 5}, Frac{3, 5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Equal(tt.a, tt.b); got != tt.want {
				t.Errorf("Equal(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestParseFracs(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []Frac
	}{
		{"question text", "What is 1/2 + 1/4 ?", []Frac{{1, 2}, {1, 4}}},
		{"spaces around slash", "1 / 2", []Frac{{1, 2}}},
		{"no fractions", "hello", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseFracs(tt.text)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseFracs(%q) = %v, want %v", tt.text, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseFracs(%q)[%d] = %v, want %v", tt.text, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseChoice(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		want   Frac
		wantOK bool
	}{
		{"fraction", "3/4", Frac{3, 4}, true},
		{"bare integer", "1", Frac{1, 1}, true},
		{"bare integer with whitespace", "  2  ", Frac{2, 1}, true},
		{"not parseable", "banana", Frac{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseChoice(tt.text)
			if ok != tt.wantOK {
				t.Fatalf("ParseChoice(%q) ok = %v, want %v", tt.text, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("ParseChoice(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestFracString(t *testing.T) {
	tests := []struct {
		name string
		f    Frac
		want string
	}{
		{"proper fraction", Frac{3, 4}, "3/4"},
		{"whole number", Frac{2, 1}, "2"},
		{"zero", Frac{0, 1}, "0"},
		{"zero with nonzero denominator", Frac{0, 5}, "0"},
		{"negative fraction", Frac{-1, 2}, "-1/2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.String(); got != tt.want {
				t.Errorf("%v.String() = %q, want %q", tt.f, got, tt.want)
			}
		})
	}
}
