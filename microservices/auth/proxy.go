package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/authctx"
)

// serviceRoutes maps API path prefixes to the microservice that owns them.
// The target address of each service comes from the environment so it can
// differ between local runs and docker-compose.
var serviceRoutes = []struct {
	prefix  string
	service string
	envVar  string
}{
	{"/api/v1/admin", "catalog", "CATALOG_SERVICE_URL"},
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
	{"/api/v1/plans", "user", "USER_SERVICE_URL"},
	{"/api/v1/discounts", "user", "USER_SERVICE_URL"},
	{"/api/v1/payments", "user", "USER_SERVICE_URL"},
	{"/api/v1/home", "catalog", "CATALOG_SERVICE_URL"},
}

// handleProxy forwards an already-authenticated request to the microservice
// that owns the route. The verified user id and email travel along in headers
// so downstream services do not have to re-parse the JWT.
func (app *application) handleProxy(w http.ResponseWriter, r *http.Request) {
	for _, route := range serviceRoutes {
		if r.URL.Path != route.prefix && !strings.HasPrefix(r.URL.Path, route.prefix+"/") {
			continue
		}

		rawURL := os.Getenv(route.envVar)
		if rawURL == "" {
			log.Printf("[%s] no target for %s %s: %s service not configured (%s)",
				hostname, r.Method, r.URL.Path, route.service, route.envVar)
			app.error(w, http.StatusServiceUnavailable, route.service+" service is not available yet")
			return
		}

		target, err := url.Parse(rawURL)
		if err != nil {
			log.Printf("[%s] invalid %s: %v", hostname, route.envVar, err)
			app.error(w, http.StatusInternalServerError, "invalid "+route.service+" service address")
			return
		}

		claims := claimsFrom(r)
		log.Printf("[%s] user %s logged in, redirecting %s %s -> %s service @ %s",
			hostname, claims.Email, r.Method, r.URL.Path, route.service, target)

		proxy := httputil.NewSingleHostReverseProxy(target)
		if app.proxyTransport != nil {
			proxy.Transport = app.proxyTransport
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("[%s] proxy to %s service failed: %v", hostname, route.service, err)
			app.error(w, http.StatusBadGateway, route.service+" service did not respond")
		}

		authctx.Inject(r, authctx.User{ID: claims.Subject, Email: claims.Email, Role: claims.Role})
		proxy.ServeHTTP(w, r)
		return
	}

	app.error(w, http.StatusNotFound, "no service owns this route")
}
