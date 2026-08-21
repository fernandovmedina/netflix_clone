package main

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashVerifyRoundTrip(t *testing.T) {
	password := []byte("correct horse battery staple")
	hash, err := bcrypt.GenerateFromPassword(password, 12)
	if err != nil {
		t.Fatal(err)
	}
	if cost, err := bcrypt.Cost(hash); err != nil || cost != 12 {
		t.Fatalf("cost=%d err=%v", cost, err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, password); err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte("wrong")); err == nil {
		t.Fatal("wrong password accepted")
	}
}

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  USER@Example.COM ")
	if err != nil || got != "user@example.com" {
		t.Fatalf("got %q, err=%v", got, err)
	}
	for _, invalid := range []string{"not-an-email", "Foo <a@b.c>", "a@example.com, b@example.com"} {
		if _, err := normalizeEmail(invalid); err == nil {
			t.Errorf("accepted invalid email %q", invalid)
		}
	}
}

func TestPasswordLengthIsValidatedInBytes(t *testing.T) {
	for _, password := range []string{"short", strings.Repeat("a", 73), strings.Repeat("😀", 30)} {
		if err := validatePassword(password); err == nil {
			t.Errorf("accepted %d-byte password", len([]byte(password)))
		}
	}
	if err := validatePassword(strings.Repeat("a", 72)); err != nil {
		t.Fatalf("rejected 72-byte password: %v", err)
	}
}

func TestEveryLoginCredentialPathPerformsOneComparison(t *testing.T) {
	var hashes [][]byte
	dummy := []byte("dummy-hash")
	app := &application{
		dummyHash: dummy,
		compareHashAndPassword: func(hash, password []byte) error {
			hashes = append(hashes, append([]byte(nil), hash...))
			return errors.New("mismatch")
		},
	}
	// A missing user and an OAuth-only user both supply an empty stored hash;
	// a password-backed user supplies the real hash. Each path must compare once.
	for _, hash := range []string{"", "", "stored-password-hash"} {
		if app.passwordMatches(hash, "valid-password") {
			t.Fatal("mismatched password accepted")
		}
	}
	if len(hashes) != 3 {
		t.Fatalf("comparisons=%d, want 3", len(hashes))
	}
	if string(hashes[0]) != string(dummy) || string(hashes[1]) != string(dummy) || string(hashes[2]) != "stored-password-hash" {
		t.Fatalf("comparison hashes=%q", hashes)
	}
}
