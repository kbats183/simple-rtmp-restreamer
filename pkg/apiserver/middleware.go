package apiserver

import (
	"crypto/subtle"
	"net/http"
	"os"
)

func ContentTypeJson(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := os.Getenv("BASIC_AUTH_USER")
		password := os.Getenv("BASIC_AUTH_PASS")
		
		// Skip auth if credentials not configured
		if username == "" || password == "" {
			next.ServeHTTP(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Use subtle.ConstantTimeCompare to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1

		if !usernameMatch || !passwordMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
