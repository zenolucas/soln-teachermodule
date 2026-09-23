package database

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newFormRequest(t *testing.T, form url.Values) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestConstructChoices(t *testing.T) {
	form := url.Values{
		"option1":          {"1/2"},
		"option1_choiceID": {"10"},
		"option2":          {"1/3"},
		"option2_choiceID": {"11"},
		"option3":          {"1/4"},
		"option3_choiceID": {"12"},
		"option4":          {"1/5"},
		"option4_choiceID": {"13"},
	}
	r := newFormRequest(t, form)

	choices, err := ConstructChoices(r, 12)
	if err != nil {
		t.Fatalf("ConstructChoices returned unexpected error: %v", err)
	}
	if len(choices) != 4 {
		t.Fatalf("got %d choices, want 4", len(choices))
	}

	wantText := []string{"1/2", "1/3", "1/4", "1/5"}
	wantID := []int{10, 11, 12, 13}
	for i, choice := range choices {
		if choice.ChoiceText != wantText[i] {
			t.Errorf("choices[%d].ChoiceText = %q, want %q", i, choice.ChoiceText, wantText[i])
		}
		if choice.ChoiceID != wantID[i] {
			t.Errorf("choices[%d].ChoiceID = %d, want %d", i, choice.ChoiceID, wantID[i])
		}
		// correct_answer identifies the choice by choice_id (12 here), not by
		// position or text - only the third option should come back marked correct.
		wantCorrect := choice.ChoiceID == 12
		if choice.IsCorrect != wantCorrect {
			t.Errorf("choices[%d].IsCorrect = %v, want %v", i, choice.IsCorrect, wantCorrect)
		}
	}
}

func TestConstructChoices_MissingChoiceID(t *testing.T) {
	// option2_choiceID is missing entirely - formInt should surface a descriptive
	// error rather than ConstructChoices silently defaulting it to 0 (see BUG-03's
	// root cause: a bare strconv.Atoi with the error discarded did exactly that).
	form := url.Values{
		"option1":          {"1/2"},
		"option1_choiceID": {"10"},
		"option2":          {"1/3"},
		"option3":          {"1/4"},
		"option3_choiceID": {"12"},
		"option4":          {"1/5"},
		"option4_choiceID": {"13"},
	}
	r := newFormRequest(t, form)

	if _, err := ConstructChoices(r, 12); err == nil {
		t.Fatal("ConstructChoices returned nil error for a missing choiceID form field, want an error")
	}
}
