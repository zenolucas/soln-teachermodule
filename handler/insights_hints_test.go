package handler

import "testing"

func TestClassHint(t *testing.T) {
	tests := []struct {
		name     string
		question string
		choices  []ChoiceStat
		want     string
		wantOK   bool
	}{
		{
			name:     "Q1 1/2 + 1/2, 5 picks of 1/2 -> straight_across",
			question: "What is 1/2 + 1/2?",
			choices: []ChoiceStat{
				{Letter: "A", Text: "1", Correct: true, Count: 3},
				{Letter: "B", Text: "1/2", Correct: false, Count: 5},
				{Letter: "C", Text: "2/2", Correct: false, Count: 1},
				{Letter: "D", Text: "0", Correct: false, Count: 1},
			},
			want:   "5 students added the numerators and the denominators straight across.",
			wantOK: true,
		},
		{
			name:     "Q10 1/2 + 1/4, picks of 1/2 -> no_common_denominator",
			question: "What is 1/2 + 1/4?",
			choices: []ChoiceStat{
				{Letter: "A", Text: "3/4", Correct: true, Count: 2},
				{Letter: "B", Text: "1/2", Correct: false, Count: 5},
				{Letter: "C", Text: "1/4", Correct: false, Count: 1},
				{Letter: "D", Text: "1", Correct: false, Count: 1},
			},
			want:   "5 students combined the numerators without finding a common denominator.",
			wantOK: true,
		},
		{
			name:     "a duplicate-value choice -> equivalent_choice",
			question: "What is 1/3 + 1/3?",
			choices: []ChoiceStat{
				{Letter: "A", Text: "2/3", Correct: true, Count: 2},
				{Letter: "B", Text: "4/6", Correct: false, Count: 6},
				{Letter: "C", Text: "1/3", Correct: false, Count: 1},
			},
			want:   "Choice B (4/6) also equals the correct answer — consider replacing it so only one answer is correct.",
			wantOK: true,
		},
		{
			name:     "subtraction straight across with equal denominators doesn't panic",
			question: "What is 2/3 − 1/3?",
			choices: []ChoiceStat{
				{Letter: "A", Text: "1/3", Correct: true, Count: 3},
				{Letter: "B", Text: "5/6", Correct: false, Count: 3},
				{Letter: "C", Text: "0", Correct: false, Count: 2},
			},
			// Whatever the result, it must not panic - straightAcross's candidate
			// denominator (3-3=0) is skipped rather than divided.
			wantOK: false,
		},
		{
			name:     "fewer than 5 responses -> no hint",
			question: "What is 1/2 + 1/2?",
			choices: []ChoiceStat{
				{Letter: "A", Text: "1", Correct: true, Count: 2},
				{Letter: "B", Text: "1/2", Correct: false, Count: 2},
			},
			wantOK: false,
		},
		{
			name:     "not hintable: only one fraction in the question",
			question: "What is 1/2 doubled?",
			choices: []ChoiceStat{
				{Letter: "A", Text: "1", Correct: true, Count: 10},
				{Letter: "B", Text: "1/2", Correct: false, Count: 10},
			},
			wantOK: false,
		},
		{
			name:     "no rule matches and the top wrong choice isn't concentrated -> spread across",
			question: "What is 1/2 + 1/3?",
			choices: []ChoiceStat{
				{Letter: "A", Text: "5/6", Correct: true, Count: 8},
				{Letter: "B", Text: "1/4", Correct: false, Count: 3},
				{Letter: "C", Text: "9/8", Correct: false, Count: 3},
				{Letter: "D", Text: "7/11", Correct: false, Count: 3},
			},
			want:   "Wrong answers were spread across the choices — that suggests guessing rather than one mistake.",
			wantOK: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, gotOK := ClassHint(tc.question, tc.choices)
			if gotOK != tc.wantOK {
				t.Fatalf("ClassHint(%q, ...) ok = %v, want %v (got %q)", tc.question, gotOK, tc.wantOK, got)
			}
			if tc.wantOK && got != tc.want {
				t.Errorf("ClassHint(%q, ...) = %q, want %q", tc.question, got, tc.want)
			}
		})
	}
}

func TestMatchRule(t *testing.T) {
	tests := []struct {
		name     string
		question string
		choice   string
		want     string
	}{
		{"straight across", "1/2 + 1/2", "1/2", "straight_across"},
		{"no common denominator", "1/2 + 1/4", "1/2", "no_common_denominator"},
		{"equivalent choice", "1/3 + 1/3", "4/6", "equivalent_choice"},
		{"matches nothing", "1/2 + 1/4", "1/5", ""},
		{"not hintable - one fraction", "1/2 doubled", "1", ""},
		{"not hintable - no operator", "1/2 1/4", "1/2", ""},
		{"zero-denominator straight-across candidate is skipped, not matched", "2/3 − 1/3", "5/6", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchRule(tc.question, tc.choice); got != tc.want {
				t.Errorf("matchRule(%q, %q) = %q, want %q", tc.question, tc.choice, got, tc.want)
			}
		})
	}
}

func TestStudentHint(t *testing.T) {
	type wrongAnswer = struct{ Question, Chosen string }

	tests := []struct {
		name   string
		wrong  []wrongAnswer
		want   string
		wantOK bool
	}{
		{
			name: "2 straight-across misses -> hint",
			wrong: []wrongAnswer{
				{Question: "1/2 + 1/2", Chosen: "1/2"},
				{Question: "1/4 + 1/4", Chosen: "2/8"},
			},
			want:   "This student tends to add or subtract the numerators and the denominators straight across, without a common denominator.",
			wantOK: true,
		},
		{
			name: "only 1 match of a rule -> no hint",
			wrong: []wrongAnswer{
				{Question: "1/2 + 1/2", Chosen: "1/2"},
				{Question: "1/2 + 1/4", Chosen: "1/5"},
			},
			wantOK: false,
		},
		{
			name:   "no wrong answers -> no hint",
			wrong:  nil,
			wantOK: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, gotOK := StudentHint(tc.wrong)
			if gotOK != tc.wantOK {
				t.Fatalf("StudentHint(%+v) ok = %v, want %v (got %q)", tc.wrong, gotOK, tc.wantOK, got)
			}
			if tc.wantOK && got != tc.want {
				t.Errorf("StudentHint(%+v) = %q, want %q", tc.wrong, got, tc.want)
			}
		})
	}
}
