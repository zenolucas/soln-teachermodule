package handler

import (
	"net/url"
	"testing"
)

func TestValidateFractionForm(t *testing.T) {
	tests := []struct {
		name        string
		vals        url.Values
		requireText bool
		wantErrs    []string // field names expected to have an error
		wantNum1    int
		wantDen1    int
		wantNum2    int
		wantDen2    int
		wantText    string
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
		{
			name: "worded: blank text is an error",
			vals: url.Values{
				"fraction1_numerator":   {"1"},
				"fraction1_denominator": {"4"},
				"fraction2_numerator":   {"1"},
				"fraction2_denominator": {"4"},
				"question_text":         {"   "},
			},
			requireText: true,
			wantErrs:    []string{"question_text"},
			wantNum1:    1, wantDen1: 4, wantNum2: 1, wantDen2: 4,
		},
		{
			name: "worded: missing text is an error",
			vals: url.Values{
				"fraction1_numerator":   {"1"},
				"fraction1_denominator": {"4"},
				"fraction2_numerator":   {"1"},
				"fraction2_denominator": {"4"},
			},
			requireText: true,
			wantErrs:    []string{"question_text"},
			wantNum1:    1, wantDen1: 4, wantNum2: 1, wantDen2: 4,
		},
		{
			name: "worded: valid text is not an error",
			vals: url.Values{
				"fraction1_numerator":   {"1"},
				"fraction1_denominator": {"4"},
				"fraction2_numerator":   {"1"},
				"fraction2_denominator": {"4"},
				"question_text":         {"What is 1/4 + 1/4?"},
			},
			requireText: true,
			wantNum1:    1, wantDen1: 4, wantNum2: 1, wantDen2: 4,
			wantText: "What is 1/4 + 1/4?",
		},
		{
			name: "fraction form ignores blank text",
			vals: url.Values{
				"fraction1_numerator":   {"1"},
				"fraction1_denominator": {"4"},
				"fraction2_numerator":   {"1"},
				"fraction2_denominator": {"4"},
			},
			requireText: false,
			wantNum1:    1, wantDen1: 4, wantNum2: 1, wantDen2: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, errs := validateFractionForm(tt.vals, tt.requireText)

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
			if tt.wantText != "" && q.QuestionText != tt.wantText {
				t.Errorf("got QuestionText %q, want %q", q.QuestionText, tt.wantText)
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

func TestValidateQuizForm(t *testing.T) {
	tests := []struct {
		name        string
		vals        url.Values
		isNew       bool
		wantErrs    []string
		wantCorrect int // index (0-based) of the choice expected to be IsCorrect, -1 if none
	}{
		{
			name:  "new mode: all valid",
			isNew: true,
			vals: url.Values{
				"question_text":  {"What is 1/2 + 1/2?"},
				"option_1":       {"1"},
				"option_2":       {"2"},
				"option_3":       {"1/2"},
				"option_4":       {"3/4"},
				"correct_answer": {"option_1"},
			},
			wantCorrect: 0,
		},
		{
			name:  "new mode: blank option",
			isNew: true,
			vals: url.Values{
				"question_text":  {"What is 1/2 + 1/2?"},
				"option_1":       {"1"},
				"option_2":       {"   "},
				"option_3":       {"1/2"},
				"option_4":       {"3/4"},
				"correct_answer": {"option_1"},
			},
			wantErrs:    []string{"option_2"},
			wantCorrect: 0,
		},
		{
			name:  "new mode: blank question text",
			isNew: true,
			vals: url.Values{
				"question_text":  {""},
				"option_1":       {"1"},
				"option_2":       {"2"},
				"option_3":       {"3"},
				"option_4":       {"4"},
				"correct_answer": {"option_1"},
			},
			wantErrs:    []string{"question_text"},
			wantCorrect: 0,
		},
		{
			name:  "new mode: no correct answer chosen",
			isNew: true,
			vals: url.Values{
				"question_text": {"What is 1/2 + 1/2?"},
				"option_1":      {"1"},
				"option_2":      {"2"},
				"option_3":      {"1/2"},
				"option_4":      {"3/4"},
			},
			wantErrs:    []string{"correct_answer"},
			wantCorrect: -1,
		},
		{
			name:  "edit mode: all valid, choice_id selected",
			isNew: false,
			vals: url.Values{
				"question":         {"What is 1/3 + 1/3?"},
				"option1":          {"1/2"},
				"option1_choiceID": {"5"},
				"option2":          {"1/3"},
				"option2_choiceID": {"6"},
				"option3":          {"1/4"},
				"option3_choiceID": {"7"},
				"option4":          {"2/3"},
				"option4_choiceID": {"8"},
				"correct_answer":   {"8"},
			},
			wantCorrect: 3,
		},
		{
			name:  "edit mode: blank option",
			isNew: false,
			vals: url.Values{
				"question":         {"What is 1/3 + 1/3?"},
				"option1":          {"1/2"},
				"option1_choiceID": {"5"},
				"option2":          {""},
				"option2_choiceID": {"6"},
				"option3":          {"1/4"},
				"option3_choiceID": {"7"},
				"option4":          {"2/3"},
				"option4_choiceID": {"8"},
				"correct_answer":   {"8"},
			},
			wantErrs:    []string{"option2"},
			wantCorrect: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, errs := validateQuizForm(tt.vals, tt.isNew)

			if len(errs) != len(tt.wantErrs) {
				t.Errorf("got %d errors %v, want %d errors %v", len(errs), errs, len(tt.wantErrs), tt.wantErrs)
			}
			for _, field := range tt.wantErrs {
				if _, ok := errs[field]; !ok {
					t.Errorf("expected an error on field %q, got none (errs=%v)", field, errs)
				}
			}

			if len(q.Choices) != 4 {
				t.Fatalf("got %d choices, want 4", len(q.Choices))
			}
			for i, c := range q.Choices {
				want := i == tt.wantCorrect
				if c.IsCorrect != want {
					t.Errorf("choice %d IsCorrect = %v, want %v", i, c.IsCorrect, want)
				}
			}
		})
	}
}
