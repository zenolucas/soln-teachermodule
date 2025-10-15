package handler

import (
	"net/http"
	"strings"
)

// WithCORS adds CORS headers for local + dev tunnel environments
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow local and Dev Tunnel origins
		if origin == "http://localhost:3000" ||
			strings.HasPrefix(origin, "https://fzpj515d-") && strings.HasSuffix(origin, ".asse.devtunnels.ms") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Common CORS headers
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests (OPTIONS)
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
