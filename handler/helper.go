package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
)

// responseRecorder wraps http.ResponseWriter to track whether a response has
// already been started, so Make can tell a handler error that got nothing on
// the wire (safe to turn into a 500) apart from one where the handler already
// wrote a partial response (nothing more we can safely send).
type responseRecorder struct {
	http.ResponseWriter
	wroteHeader bool
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.wroteHeader = true
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	// http.ResponseWriter.Write implicitly sends a 200 status if WriteHeader
	// hasn't been called yet, so this also counts as "a response started".
	rec.wroteHeader = true
	return rec.ResponseWriter.Write(b)
}

func Make(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w}
		if err := h(rec, r); err != nil {
			slog.Error("internal server error", "err", err, "path", r.URL.Path)
			if !rec.wroteHeader {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}
	}
}

func render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	return component.Render(r.Context(), w)
}

// getTeacherID reads the authenticated teacher's ID out of the session. Every route that
// calls this already sits behind WithAuth, so teacherID is always set in practice - but an
// unchecked type assertion here still panics rather than erroring if that ever isn't true
// (a tampered cookie, a bug elsewhere), so check it explicitly.
func getTeacherID(r *http.Request) (int, error) {
	session, _ := store.Get(r, sessionUserKey)
	teacherID, ok := session.Values["teacherID"].(int)
	if !ok {
		return 0, errors.New("no teacherID in session")
	}
	return teacherID, nil
}

// formInt reads a form value and parses it as an int, returning a descriptive error
// instead of silently defaulting to 0 on missing or non-numeric input. A bare
// strconv.Atoi with the error discarded is what let a bad or missing ID quietly turn
// into a query against ID 0 (see BUG-03) instead of failing loudly.
func formInt(r *http.Request, key string) (int, error) {
	n, err := strconv.Atoi(r.FormValue(key))
	if err != nil {
		return 0, fmt.Errorf("invalid or missing %q: %w", key, err)
	}
	return n, nil
}

func hxRedirect(w http.ResponseWriter, r *http.Request, to string) error {
	if len(r.Header.Get("HX-Request")) > 0 {
		w.Header().Set("HX-Redirect", to)
		w.WriteHeader(http.StatusSeeOther) 
		return nil
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
	return nil
}
