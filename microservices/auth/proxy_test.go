package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jwtutil"
	"github.com/golang-jwt/jwt/v5"
)

func TestProxyStripsForgedUserHeaders(t *testing.T) {
	var got http.Header
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		got = r.Header.Clone()
		return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	t.Setenv("CATALOG_SERVICE_URL", "http://catalog.test")
	app := &application{proxyTransport: transport}
	req := httptest.NewRequest(http.MethodGet, "http://gateway/api/v1/titles", nil)
	req.Header.Set("X-User-Id", "attacker")
	req.Header.Set("X-User-Email", "attacker@example.com")
	req.Header.Set("X-User-Role", "admin")
	req.Header["X_User_Role"] = []string{"admin"}
	req.Header["X-User_Role"] = []string{"admin"}
	req.Header["x_user_id"] = []string{"attacker"}
	claims := &jwtutil.Claims{Email: "real@example.com", Role: "user", RegisteredClaims: jwt.RegisteredClaims{Subject: "real-user"}}
	req = req.WithContext(context.WithValue(req.Context(), claimsKey, claims))
	rec := httptest.NewRecorder()
	app.handleProxy(rec, req)
	if got.Get("X-User-Role") != "user" || got.Get("X-User-Id") != "real-user" || got.Get("X-User-Email") != "real@example.com" {
		t.Fatalf("forged headers survived: %#v", got)
	}
	allowed := map[string]bool{"X-User-Id": true, "X-User-Email": true, "X-User-Role": true}
	for key := range got {
		normalized := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
		if (normalized == "x-user-id" || normalized == "x-user-email" || normalized == "x-user-role") && !allowed[key] {
			t.Fatalf("non-canonical identity header survived: %q", key)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }
