package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/sessions"
)

func WithAuth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/public") {
			next.ServeHTTP(w, r)
			return
		}

		// Set cache control headers
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
		session, _ := store.Get(r, sessionUserKey)

		// fmt.Print("authenticatd is : ", session.Values["authenticated"])

		if !session.Values["authenticated"].(bool) {
			path := r.URL.Path
			hxRedirect(w, r, "/?to"+path)
			return
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
