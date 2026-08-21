package jwtutil

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
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
