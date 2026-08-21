package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// SupabaseClaims are the claims Supabase Auth embeds in every access token.
type SupabaseClaims struct {
	jwt.RegisteredClaims
	Email        string         `json:"email"`
	Role         string         `json:"role"`
	UserMetadata map[string]any `json:"user_metadata"`
}

// Name returns the display name stored in the token's user metadata.
func (c *SupabaseClaims) Name() string {
	if name, ok := c.UserMetadata["name"].(string); ok {
		return name
	}
	return ""
}

var (
	jwtKeyfunc  jwt.Keyfunc
	jwtMethods  []string
	jwksCleanup func()
)

// initJWT prepares local verification of Supabase access tokens. Projects on
// asymmetric signing keys (the current Supabase default) are verified against
// the project's JWKS endpoint; legacy projects can set SUPABASE_JWT_SECRET to
// verify with the shared HS256 secret instead.
func initJWT() error {
	if secret := os.Getenv("SUPABASE_JWT_SECRET"); secret != "" {
		jwtKeyfunc = func(t *jwt.Token) (any, error) { return []byte(secret), nil }
		jwtMethods = []string{"HS256"}
		return nil
	}

	baseURL := os.Getenv("SUPABASE_URL")
	if baseURL == "" {
		return fmt.Errorf("SUPABASE_URL must be set to fetch the JWKS")
	}

	ctx, cancel := context.WithCancel(context.Background())
	jwks, err := keyfunc.NewDefaultCtx(ctx, []string{baseURL + "/auth/v1/.well-known/jwks.json"})
	if err != nil {
		cancel()
		return fmt.Errorf("fetching Supabase JWKS: %w", err)
	}

	jwtKeyfunc = jwks.Keyfunc
	jwtMethods = []string{"ES256", "RS256"}
	jwksCleanup = cancel
	return nil
}

// verifyToken checks the signature, expiry and audience of a Supabase access
// token and returns its claims.
func verifyToken(tokenString string) (*SupabaseClaims, error) {
	claims := &SupabaseClaims{}
	_, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		jwtKeyfunc,
		jwt.WithValidMethods(jwtMethods),
		jwt.WithAudience("authenticated"),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
