package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCrossOriginAllowed(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		secFetchSite string
		origin       string
		want         bool
	}{
		{"GET is always allowed", http.MethodGet, "cross-site", "https://evil.example", true},
		{"same-origin POST (htmx, forms)", http.MethodPost, "same-origin", "", true},
		{"user-initiated navigation", http.MethodPost, "none", "", true},
		{"cross-site POST", http.MethodPost, "cross-site", "https://evil.example", false},
		{"same-site but different origin POST", http.MethodPost, "same-site", "https://other.portal.example", false},
		{"old browser: matching Origin", http.MethodPost, "", "http://portal.example:3000", true},
		{"old browser: foreign Origin", http.MethodPost, "", "https://evil.example", false},
		{"non-browser client (curl, the game): no headers", http.MethodPost, "", "", true},
		{"garbage Origin", http.MethodPost, "", "::not a url", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "http://portal.example:3000/createclassroom", nil)
			if tt.secFetchSite != "" {
				r.Header.Set("Sec-Fetch-Site", tt.secFetchSite)
			}
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if got := crossOriginAllowed(r); got != tt.want {
				t.Errorf("crossOriginAllowed = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithCrossOriginProtectionRejects(t *testing.T) {
	called := false
	h := WithCrossOriginProtection(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	r := httptest.NewRequest(http.MethodPost, "http://portal.example/delete/student", nil)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || called {
		t.Errorf("got status %d, handler called %v; want 403 and not called", w.Code, called)
	}
}
