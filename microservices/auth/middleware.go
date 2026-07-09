package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const (
	claimsKey contextKey = "claims"
	tokenKey  contextKey = "token"
)

// requireAuth rejects requests that do not carry a valid Supabase access
// token in the Authorization header, and stores the verified claims in the
// request context for handlers and the proxy.
func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "missing bearer token: the user is not logged in")
			return
		}

		claims, err := verifyToken(token)
		if err != nil {
			log.Printf("[%s] rejected token for %s %s: %v", hostname, r.Method, r.URL.Path, err)
			writeError(w, http.StatusUnauthorized, "invalid or expired session: the user is not logged in")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		ctx = context.WithValue(ctx, tokenKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func claimsFrom(r *http.Request) *SupabaseClaims {
	claims, _ := r.Context().Value(claimsKey).(*SupabaseClaims)
	return claims
}

func tokenFrom(r *http.Request) string {
	token, _ := r.Context().Value(tokenKey).(string)
	return token
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
