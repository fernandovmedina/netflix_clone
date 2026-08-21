package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jwtutil"
	"github.com/google/uuid"
)

type tokenManager struct {
	secret    []byte
	accessTTL time.Duration
}

func newTokenManager() (*tokenManager, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 bytes")
	}
	ttl := 15 * time.Minute
	if value := os.Getenv("ACCESS_TOKEN_TTL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return nil, fmt.Errorf("ACCESS_TOKEN_TTL: %w", err)
		}
		ttl = parsed
	}
	return &tokenManager{secret: secret, accessTTL: ttl}, nil
}

func (m *tokenManager) sign(user User) (string, error) {
	return jwtutil.Sign(m.secret, user.ID, user.Email, user.Role, time.Now(), m.accessTTL, uuid.NewString())
}
func (m *tokenManager) verify(raw string) (*jwtutil.Claims, error) {
	return jwtutil.Verify(m.secret, raw)
}
