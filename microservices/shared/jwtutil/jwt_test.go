package jwtutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestSignVerify(t *testing.T) {
	secret := []byte(strings.Repeat("a", 32))
	raw, err := Sign(secret, "user-id", "a@example.com", "admin", time.Now(), time.Minute, "jti")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Verify(secret, raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != "admin" || claims.Subject != "user-id" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if _, err := Verify([]byte(strings.Repeat("b", 32)), raw); err == nil {
		t.Fatal("wrong secret accepted")
	}
	parts := strings.Split(raw, ".")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	payload = []byte(strings.Replace(string(payload), `"role":"admin"`, `"role":"user"`, 1))
	parts[1] = base64.RawURLEncoding.EncodeToString(payload)
	if _, err := Verify(secret, strings.Join(parts, ".")); err == nil {
		t.Fatal("tampered token accepted")
	}
}

func TestVerifyRejectsAlgorithmConfusionAndNone(t *testing.T) {
	secret := []byte(strings.Repeat("a", 32))
	payload := validPayload(t)
	for name, header := range map[string]string{
		"RS256 with HMAC signature": `{"alg":"RS256","typ":"JWT"}`,
		"none":                      `{"alg":"none","typ":"JWT"}`,
	} {
		t.Run(name, func(t *testing.T) {
			unsigned := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." + base64.RawURLEncoding.EncodeToString(payload)
			raw := unsigned + "."
			if name != "none" {
				mac := hmac.New(sha256.New, secret)
				_, _ = mac.Write([]byte(unsigned))
				raw += base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
			}
			if _, err := Verify(secret, raw); err == nil {
				t.Fatal("token accepted")
			}
		})
	}
}

func TestVerifyRejectsWrongIssuerAndAudience(t *testing.T) {
	secret := []byte(strings.Repeat("a", 32))
	for name, mutate := range map[string]func(*Claims){
		"issuer":   func(c *Claims) { c.Issuer = "attacker" },
		"audience": func(c *Claims) { c.Audience = jwt.ClaimStrings{"attacker"} },
	} {
		t.Run(name, func(t *testing.T) {
			claims := validClaims()
			mutate(&claims)
			raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Verify(secret, raw); err == nil {
				t.Fatal("token accepted")
			}
		})
	}
}

func validClaims() Claims {
	now := time.Now()
	return Claims{Email: "a@example.com", Role: "user", RegisteredClaims: jwt.RegisteredClaims{
		Subject: "user-id", Issuer: Issuer, Audience: jwt.ClaimStrings{Audience}, ID: "jti",
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
	}}
}

func validPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := json.Marshal(validClaims())
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestExpiredRejected(t *testing.T) {
	secret := []byte(strings.Repeat("a", 32))
	raw, err := Sign(secret, "user-id", "a@example.com", "user", time.Now().Add(-time.Hour), time.Minute, "jti")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(secret, raw); err == nil {
		t.Fatal("expired token accepted")
	}
}
