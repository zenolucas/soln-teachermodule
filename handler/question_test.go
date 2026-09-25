package handler

import (
	"net/url"
	"testing"
)

func TestValidateFractionForm(t *testing.T) {
	tests := []struct {
		name     string
		vals     url.Values
		wantErrs []string // field names expected to have an error
		wantNum1 int
		wantDen1 int
		wantNum2 int
		wantDen2 int
	}{
		{
			name: "all valid",
			vals: url.Values{
				"fraction1_numerator":   {"1"},
				"fraction1_denominator": {"3"},
				"fraction2_numerator":   {"1"},
				"fraction2_denominator": {"3"},
			},
			wantNum1: 1, wantDen1: 3, wantNum2: 1, wantDen2: 3,
		},
		{
			name: "denominator zero",
			vals: url.Values{
				"fraction1_numerator":   {"7"},
				"fraction1_denominator": {"10"},
				"fraction2_numerator":   {"1"},
				"fraction2_denominator": {"0"},
			},
			wantErrs: []string{"fraction2_denominator"},
			wantNum1: 7, wantDen1: 10, wantNum2: 1, wantDen2: 0,
		},
		{
			name: "negative numerator",
			vals: url.Values{
				"fraction1_numerator":   {"-1"},
				"fraction1_denominator": {"2"},
				"fraction2_numerator":   {"1"},
				"fraction2_denominator": {"2"},
			},
			wantErrs: []string{"fraction1_numerator"},
			wantNum1: -1, wantDen1: 2, wantNum2: 1, wantDen2: 2,
		},
		{
			name: "non-numeric field",
			vals: url.Values{
				"fraction1_numerator":   {"abc"},
				"fraction1_denominator": {"2"},
				"fraction2_numerator":   {"1"},
				"fraction2_denominator": {"2"},
			},
			wantErrs: []string{"fraction1_numerator"},
			wantNum1: 0, wantDen1: 2, wantNum2: 1, wantDen2: 2,
		},
		{
			name: "missing fields",
			vals: url.Values{},
			wantErrs: []string{
				"fraction1_numerator", "fraction1_denominator",
				"fraction2_numerator", "fraction2_denominator",
			},
		},
		{
			name: "zero numerator is valid",
			vals: url.Values{
				"fraction1_numerator":   {"0"},
				"fraction1_denominator": {"2"},
				"fraction2_numerator":   {"1"},
				"fraction2_denominator": {"2"},
			},
			wantNum1: 0, wantDen1: 2, wantNum2: 1, wantDen2: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, errs := validateFractionForm(tt.vals)

			if len(errs) != len(tt.wantErrs) {
				t.Errorf("got %d errors %v, want %d errors %v", len(errs), errs, len(tt.wantErrs), tt.wantErrs)
			}
			for _, field := range tt.wantErrs {
				if _, ok := errs[field]; !ok {
					t.Errorf("expected an error on field %q, got none (errs=%v)", field, errs)
				}
			}

			if q.Fraction1_Numerator != tt.wantNum1 || q.Fraction1_Denominator != tt.wantDen1 ||
				q.Fraction2_Numerator != tt.wantNum2 || q.Fraction2_Denominator != tt.wantDen2 {
				t.Errorf("got %+v, want Num1=%d Den1=%d Num2=%d Den2=%d", q, tt.wantNum1, tt.wantDen1, tt.wantNum2, tt.wantDen2)
			}
		})
	}
}

func TestOperationLabel(t *testing.T) {
	tests := []struct {
		op   string
		want string
	}{
		{"+", "Addition"},
		{"−", "Subtraction"},
		{"-", "Subtraction"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := operationLabel(tt.op); got != tt.want {
			t.Errorf("operationLabel(%q) = %q, want %q", tt.op, got, tt.want)
		}
	}
}
