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

		store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
		session, _ := store.Get(r, sessionUserKey)

		if !session.Values["authenticated"].(bool) {
			path := r.URL.Path
			hxRedirect(w, r, "/?to"+path)
			return
		}

		// fmt.Print("authenticated is : ", session.Values["authenticated"])
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
