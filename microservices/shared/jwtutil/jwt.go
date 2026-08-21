package jwtutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const Issuer = "netflix-clone"
const Audience = "netflix-clone"

type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func Sign(secret []byte, userID, email, role string, now time.Time, ttl time.Duration, jti string) (string, error) {
	claims := Claims{Email: email, Role: role, RegisteredClaims: jwt.RegisteredClaims{
		Subject: userID, Issuer: Issuer, Audience: jwt.ClaimStrings{Audience}, ID: jti,
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func Verify(secret []byte, raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(Issuer), jwt.WithAudience(Audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if claims.Role != "user" && claims.Role != "admin" {
		return nil, fmt.Errorf("invalid role claim")
	}
	if claims.Subject == "" || claims.Email == "" || claims.ID == "" {
		return nil, fmt.Errorf("missing required claims")
	}
	return claims, nil
}
