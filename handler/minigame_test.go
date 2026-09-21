package handler

import "testing"

func TestGetCorrectAnswer(t *testing.T) {
	tests := []struct {
		name      string
		isCorrect bool
		want      string
	}{
		{"correct choice is marked selected", true, "selected"},
		{"incorrect choice gets no attribute", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCorrectAnswer(tt.isCorrect); got != tt.want {
				t.Errorf("getCorrectAnswer(%v) = %q, want %q", tt.isCorrect, got, tt.want)
			}
		})
	}
}
