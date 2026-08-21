package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// serviceRoutes maps API path prefixes to the microservice that owns them.
// The target address of each service comes from the environment so it can
// differ between local runs and docker-compose.
var serviceRoutes = []struct {
	prefix  string
	service string
	envVar  string
}{
	{"/api/v1/titles", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/movies", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/movie", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/series", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/serie", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/seasons", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/episodes", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/actors", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/categories", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/genres", "catalog", "CATALOG_SERVICE_URL"},
	{"/api/v1/stream", "streaming", "STREAMING_SERVICE_URL"},
	{"/api/v1/progress", "user", "USER_SERVICE_URL"},
	{"/api/v1/favorites", "user", "USER_SERVICE_URL"},
	{"/api/v1/profiles", "user", "USER_SERVICE_URL"},
}

// handleProxy forwards an already-authenticated request to the microservice
// that owns the route. The verified user id and email travel along in headers
// so downstream services do not have to re-parse the JWT.
func handleProxy(w http.ResponseWriter, r *http.Request) {
	for _, route := range serviceRoutes {
		if r.URL.Path != route.prefix && !strings.HasPrefix(r.URL.Path, route.prefix+"/") {
			continue
		}

		rawURL := os.Getenv(route.envVar)
		if rawURL == "" {
			log.Printf("[%s] no target for %s %s: %s service not configured (%s)",
				hostname, r.Method, r.URL.Path, route.service, route.envVar)
			writeError(w, http.StatusServiceUnavailable, route.service+" service is not available yet")
			return
		}

		target, err := url.Parse(rawURL)
		if err != nil {
			log.Printf("[%s] invalid %s: %v", hostname, route.envVar, err)
			writeError(w, http.StatusInternalServerError, "invalid "+route.service+" service address")
			return
		}

		claims := claimsFrom(r)
		log.Printf("[%s] user %s logged in, redirecting %s %s -> %s service @ %s",
			hostname, claims.Email, r.Method, r.URL.Path, route.service, target)

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("[%s] proxy to %s service failed: %v", hostname, route.service, err)
			writeError(w, http.StatusBadGateway, route.service+" service did not respond")
		}

		r.Header.Set("X-User-Id", claims.Subject)
		r.Header.Set("X-User-Email", claims.Email)
		proxy.ServeHTTP(w, r)
		return
	}

	writeError(w, http.StatusNotFound, "no service owns this route")
}
