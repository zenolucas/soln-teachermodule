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
		session, err := store.Get(r, sessionUserKey)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			path := r.URL.Path
			http.Redirect(w, r, "/?to"+path, http.StatusSeeOther)
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// func WithAuth(next http.Handler) http.Handler {
// 	fn := func(w http.ResponseWriter, r *http.Request) {
// 		if strings.Contains(r.URL.Path, "/public") {
// 			next.ServeHTTP(w, r)
// 			return
// 		}
// 		fmt.Print("this is executed")
// 		user := GetAuthenticatedUser(r)
// 		if !user.LoggedIn {
// 			path := r.URL.Path
// 			http.Redirect(w, r, "/login?to="+path, http.StatusSeeOther)
// 			return
// 		}
// 		next.ServeHTTP(w, r)
// 	}
// 	return http.HandlerFunc(fn)
// }
