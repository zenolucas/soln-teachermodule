package handler

import (
	"net/http"
	"net/url"
)

func WithAuth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, sessionUserKey)

		// A first-time visitor has no "authenticated" value in the session at all, so
		// Values["authenticated"] is nil - an unchecked type assertion to bool panics
		// on that instead of falling through to the redirect below.
		authenticated, ok := session.Values["authenticated"].(bool)
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

// WithCrossOriginProtection rejects cross-site state-changing requests (CSRF, SEC-08), following
// the same rules as Go 1.25's http.CrossOriginProtection. Browsers send Sec-Fetch-Site on every
// request; without it, a present Origin must match the Host. A request with neither isn't from a
// browser (curl, the Godot game), so it can't be CSRF and passes. The session cookie's SameSite=Lax
// stays as a second layer.
func WithCrossOriginProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !crossOriginAllowed(r) {
			http.Error(w, "cross-origin request rejected", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func crossOriginAllowed(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "none":
		return true
	case "":
		// Older browser or non-browser client: fall back to Origin.
	default: // "cross-site", "same-site"
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host
}
