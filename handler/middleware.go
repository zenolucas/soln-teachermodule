package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"soln-teachermodule/database"
)

func WithAuth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionUserKey)

		// A first-time visitor has no "authenticated" value in the session at all, so
		// Values["authenticated"] is nil - an unchecked type assertion to bool panics
		// on that instead of falling through to the redirect below.
		authenticated, ok := session.Values["authenticated"].(bool)
		if ok && authenticated {
			valid, err := sessionStillValid(r, session.Values)
			if err != nil {
				slog.Error("checking session version", "err", err, "path", r.URL.Path)
				http.Error(w, "Something went wrong on our end. Please try again.", http.StatusInternalServerError)
				return
			}
			authenticated = valid
		}
		if !ok || !authenticated {
			// RequestURI (not just Path) keeps the query string too, e.g.
			// ?minigameID=5&classroomID=1 on a statistics page - dropping it here is
			// what made FE-07's post-login redirect land back on a page with no idea
			// which classroom/minigame to show. QueryEscape it since it becomes the
			// value of our own ?to= query parameter.
			to := url.QueryEscape(r.URL.RequestURI())
			hxRedirect(w, r, "/login?to="+to)
			return
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// sessionStillValid reports whether the cookie's session_version matches the teacher's current one.
// Logout bumps the stored version, so any cookie issued before it (including copies) is rejected.
// Cookies from before this check existed carry no version and are rejected too: one extra login.
func sessionStillValid(r *http.Request, values map[interface{}]interface{}) (bool, error) {
	teacherID, ok := values["teacherID"].(int)
	if !ok {
		return false, nil
	}
	version, ok := values["sessionVersion"].(int)
	if !ok {
		return false, nil
	}
	current, err := database.GetSessionVersion(r.Context(), teacherID)
	if err != nil {
		return false, err
	}
	return version == current, nil
}
