package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQueryIntAndFormInt(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/x?classroom_id=7&bad=abc", strings.NewReader("classroomID=9"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if n, err := queryInt(r, "classroom_id"); err != nil || n != 7 {
		t.Errorf("queryInt(classroom_id) = %d, %v; want 7, nil", n, err)
	}
	if n, err := formInt(r, "classroomID"); err != nil || n != 9 {
		t.Errorf("formInt(classroomID) = %d, %v; want 9, nil", n, err)
	}
	for _, key := range []string{"bad", "missing"} {
		_, err := queryInt(r, key)
		var bad badRequestError
		if !errors.As(err, &bad) {
			t.Errorf("queryInt(%q) error = %v; want a badRequestError", key, err)
		}
	}
	// queryInt must not read the body: classroomID is only in the form.
	if _, err := queryInt(r, "classroomID"); err == nil {
		t.Error("queryInt(classroomID) read the request body; want an error")
	}
}

func TestMakeStatusForErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"bad parameter is the client's fault", badRequestError{errors.New("invalid or missing \"userID\"")}, http.StatusBadRequest},
		{"anything else is a server error", errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Make(func(w http.ResponseWriter, r *http.Request) error { return tt.err })
			r := httptest.NewRequest(http.MethodGet, "/student/score?userID=abc", nil)
			r.Header.Set("HX-Request", "true") // plain-text error, no template rendering needed
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
