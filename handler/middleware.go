package handler

import (
	"net/http"
	"strings"
)

// WithCORS adds CORS headers for local development
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow from localhost:3000 (adjust if needed)
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func WithAuth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/public") {
			next.ServeHTTP(w, r)
			return
		}

		session, _ := store.Get(r, sessionUserKey)

		if !session.Values["authenticated"].(bool) {
			path := r.URL.Path
			hxRedirect(w, r, "/login?to"+path)
			return
		}

		// fmt.Print("authenticated is : ", session.Values["authenticated"])
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
