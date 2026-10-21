package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"soln-teachermodule/view/layout"
	"soln-teachermodule/view/ui"
)

// The landing page is the one public page that changes in the static showcase: with
// DEMO_MODE off it must stay byte-for-byte the normal portal (login link, embedded
// download), and with it on it walks straight into /home and offers a live demo.
func TestLandingDemoMode(t *testing.T) {
	render := func(t *testing.T) string {
		t.Helper()
		rec := httptest.NewRecorder()
		if err := HandleLandingIndex(rec, httptest.NewRequest(http.MethodGet, "/", nil)); err != nil {
			t.Fatal(err)
		}
		return rec.Body.String()
	}
	t.Cleanup(InitDemo)

	t.Setenv("DEMO_MODE", "")
	InitDemo()
	normal := render(t)
	for _, want := range []string{`href="/login"`, `href="/public/downloads/soln.zip"`} {
		if !strings.Contains(normal, want) {
			t.Errorf("normal landing page is missing %s", want)
		}
	}
	for _, unwanted := range []string{"demo.js", "book-demo-modal", `href="/home"`} {
		if strings.Contains(normal, unwanted) {
			t.Errorf("normal landing page contains demo markup %q", unwanted)
		}
	}

	t.Setenv("DEMO_MODE", "true")
	t.Setenv("DEMO_BOOKING_URL", "https://cal.example/soln")
	t.Setenv("DEMO_DOWNLOAD_URL", "https://downloads.example/soln.zip")
	InitDemo()
	demoPage := render(t)
	for _, want := range []string{
		`src="/public/js/demo.js"`, `href="/home"`, `id="book-demo-modal"`,
		`href="https://cal.example/soln"`, `href="https://downloads.example/soln.zip"`,
	} {
		if !strings.Contains(demoPage, want) {
			t.Errorf("demo landing page is missing %s", want)
		}
	}
	if strings.Contains(demoPage, `href="/login"`) {
		t.Error("demo landing page still links to /login")
	}

	// No booking URL configured: demo mode still works, just without the booking CTA.
	t.Setenv("DEMO_BOOKING_URL", "")
	InitDemo()
	if page := render(t); strings.Contains(page, "book-demo-modal") {
		t.Error("demo landing page renders the booking dialog with no DEMO_BOOKING_URL")
	}
}

// Shell pages: the banner, the sidebar's "Book a live demo" button and "Exit demo"
// (instead of the POST /logout form) appear only in demo mode.
func TestShellDemoMode(t *testing.T) {
	render := func(t *testing.T, d ui.Demo) string {
		t.Helper()
		var b strings.Builder
		if err := layout.Base(layout.Page{Title: "x", Shell: true, Demo: d}).Render(context.Background(), &b); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}

	normal := render(t, ui.Demo{})
	if !strings.Contains(normal, `action="/logout"`) {
		t.Error("normal shell is missing the logout form")
	}
	for _, unwanted := range []string{"data-demo-reset", "demo.js", "book-demo-modal", "Exit demo"} {
		if strings.Contains(normal, unwanted) {
			t.Errorf("normal shell contains demo markup %q", unwanted)
		}
	}

	demoShell := render(t, ui.Demo{Enabled: true, BookingURL: "https://cal.example/soln"})
	for _, want := range []string{"data-demo-reset", `src="/public/js/demo.js"`, `id="book-demo-modal"`, `aria-label="Exit demo"`, `data-modal-open="book-demo-modal"`} {
		if !strings.Contains(demoShell, want) {
			t.Errorf("demo shell is missing %s", want)
		}
	}
	if strings.Contains(demoShell, `action="/logout"`) {
		t.Error("demo shell still renders the logout form")
	}
}
