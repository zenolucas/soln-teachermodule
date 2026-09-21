package handler

import (
	"log/slog"
	"net/http"

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

func hxRedirect(w http.ResponseWriter, r *http.Request, to string) error {
	if len(r.Header.Get("HX-Request")) > 0 {
		w.Header().Set("HX-Redirect", to)
		w.WriteHeader(http.StatusSeeOther) 
		return nil
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
	return nil
}
