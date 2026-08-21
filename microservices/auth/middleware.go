package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jwtutil"
)

type contextKey string

const (
	claimsKey contextKey = "claims"
)

// requireAuth verifies the access token and stores its signed claims in the
// request context for handlers and the proxy.
func (app *application) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := accessTokenFrom(r)
		if token == "" {
			app.error(w, http.StatusUnauthorized, "authentication required")
			return
		}

		claims, err := app.tokens.verify(token)
		if err != nil {
			log.Printf("[%s] rejected token for %s %s: %v", hostname, r.Method, r.URL.Path, err)
			app.error(w, http.StatusUnauthorized, "invalid or expired access token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) requireAdmin(next http.Handler) http.Handler {
	return app.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claimsFrom(r).Role != "admin" {
			app.error(w, http.StatusForbidden, "administrator access required")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func claimsFrom(r *http.Request) *jwtutil.Claims {
	claims, _ := r.Context().Value(claimsKey).(*jwtutil.Claims)
	return claims
}

func accessTokenFrom(r *http.Request) string {
	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok && token != "" {
		return token
	}
	if cookie, err := r.Cookie("access_token"); err == nil {
		return cookie.Value
	}
	return ""
}

// statusRecorder captures the status code a handler writes so the request
// log can include it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

// logRequests prints one line per request, tagged with the container
// hostname so it is visible which container behind the load balancer served
// which request.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.Printf("[%s] %s %s from %s -> %d (%s)",
			hostname, r.Method, r.URL.Path, r.RemoteAddr, rec.status, time.Since(start).Round(time.Millisecond))
	})
}
