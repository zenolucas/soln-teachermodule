package handler

import (
	"fmt"
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

		// fmt.Print("authenticatd is : ", session.Values["authenticated"])

		if !session.Values["authenticated"].(bool) {
			fmt.Print("this is executed!")
			path := r.URL.Path
			http.Redirect(w, r, "/?to"+path, http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
