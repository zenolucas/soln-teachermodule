package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"soln-teachermodule/database"
	"soln-teachermodule/view/errorpage"
	"strconv"
	"strings"

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
				renderErrorPage(w, r, http.StatusInternalServerError, "Something went wrong on our end. Please try again.")
			}
		}
	}
}

// wantsErrorPage reports whether r looks like a real browser navigation that a full
// styled error page would help, as opposed to an htmx fragment request, a JSON API
// call (the statistics/chart endpoints fetched with plain fetch()), or the Godot
// client's /game/* requests, none of which should get an HTML document back (see
// FE-05 and the longer note on Make above).
func wantsErrorPage(r *http.Request) bool {
	if r.Header.Get("HX-Request") == "true" {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/game/") {
		return false
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

// renderErrorPage responds with code, choosing between the full styled HTML page (a
// genuine full-page browser navigation - see wantsErrorPage) and a plain-text
// http.Error otherwise. This makes it safe to call from any handler no matter how
// it's reached - a full-page route, an htmx fragment endpoint, a JSON API call, or a
// hand-crafted request pretending to be one of those - without each call site having
// to reason about which is which (see FE-05; Make's own fallback below and three
// direct callers use this the same way).
func renderErrorPage(w http.ResponseWriter, r *http.Request, code int, message string) error {
	if !wantsErrorPage(r) {
		http.Error(w, message, code)
		return nil
	}
	w.WriteHeader(code)
	return errorpage.Error(code, message).Render(r.Context(), w)
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

// assertOwnsClassroom verifies the authenticated teacher owns the given classroom,
// writing the response and returning a non-nil error if not (401 if the session
// itself is invalid, 403 if it's valid but belongs to a different teacher). Every
// classroom-scoped teacher-portal handler must call this before reading or writing
// anything scoped to classroomID - without it, any authenticated teacher can view,
// edit, or delete any other teacher's classroom, students, or questions just by
// changing a URL or form parameter (see SEC-06).
func assertOwnsClassroom(w http.ResponseWriter, r *http.Request, classroomID int) error {
	teacherID, err := getTeacherID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return err
	}

	ownerID, err := database.GetClassroomTeacherID(r.Context(), classroomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// A classroom ID that doesn't exist at all previously fell through to
			// the generic 500 in Make - misleading, since nothing actually went
			// wrong server-side (see FE-05).
			http.Error(w, "not found", http.StatusNotFound)
			return fmt.Errorf("classroom %d not found", classroomID)
		}
		return err
	}

	if ownerID != teacherID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return fmt.Errorf("teacher %d does not own classroom %d", teacherID, classroomID)
	}

	return nil
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

// esc escapes a string for safe interpolation into HTML markup built by hand with
// fmt.Fprintf. These fragments (classroom names/descriptions, student names, question
// text, all teacher- or registration-supplied) are the only part of the app that
// doesn't use templ, which escapes automatically - until they're ported, every %s
// argument that carries string data must go through this (see SEC-07). Safe to use on
// IDs and other non-string-typed values too; escaping plain digits is a no-op.
func esc(s string) string {
	return html.EscapeString(s)
}

// escJS renders s as a JSON string literal (quotes included), safe to embed directly
// inside a hand-built <script> block - a plain %s there could break out of the script
// via a literal quote, backslash, or "</script>". encoding/json escapes '<', '>', and
// '&' by default specifically to make its output safe to embed in HTML/script
// contexts, which a plain fmt %q or manual quoting would not do (see SEC-07).
func escJS(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// minigameKind classifies which question type a minigame ID holds. HandleMinigameIndex
// and HandleStatisticsIndex both need this exact mapping to pick which templ component
// to render; centralizing it here means adding minigame 13 is one line in one place,
// instead of a new branch in two separate 12-way if/else chains.
// minigameKind's zero value is kindInvalid, not kindFractions, so a lookup miss on
// minigameKinds (an unrecognized minigame ID) falls through to the error case in both
// handlers rather than silently rendering as if it were minigame 1.
type minigameKind int

const (
	kindInvalid minigameKind = iota
	kindFractions
	kindWorded
	kindQuiz
)

var minigameKinds = map[string]minigameKind{
	"1": kindFractions, "2": kindFractions,
	"3": kindWorded, "4": kindWorded,
	"5":  kindQuiz,
	"6":  kindFractions,
	"7":  kindFractions,
	"8":  kindFractions,
	"9":  kindFractions,
	"10": kindWorded,
	"11": kindQuiz,
	"12": kindQuiz,
}

// isLocalRedirect reports whether to is safe to redirect a logged-in-again teacher
// to (see FE-07): it must be a same-site path, not an absolute URL or a
// protocol-relative one (`//evil.com` or `/\evil.com`, both of which browsers treat
// as a scheme-relative redirect to a different host). Without this check, carrying
// the post-login destination through as a plain redirect target would be an open
// redirect: /login?to=https://evil.example would send a teacher who just typed their
// password straight to an attacker's page.
func isLocalRedirect(to string) bool {
	if to == "" || to[0] != '/' {
		return false
	}
	if strings.HasPrefix(to, "//") || strings.HasPrefix(to, "/\\") {
		return false
	}
	return true
}

// pctCorrect renders a right/wrong attempt count as a percentage string for display,
// e.g. "75%" - or "—" when there have been no attempts yet, since 0/0 would otherwise
// render as a misleading "0%" (see FE-31).
func pctCorrect(right, wrong int) string {
	total := right + wrong
	if total == 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(right)/float64(total))
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
