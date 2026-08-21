package main

import (
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
