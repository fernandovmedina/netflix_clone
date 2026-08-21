package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRefreshReuseRevokesFamily(t *testing.T) {
	dsn := os.Getenv("AUTH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AUTH_TEST_DATABASE_URL not set; integration test requires migrated throwaway Postgres")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tokens := &tokenManager{secret: []byte(strings.Repeat("x", 32)), accessTTL: time.Minute}
	repo := &repository{pool: pool, tokens: tokens, refreshTTL: time.Hour}
	var user User
	err = pool.QueryRow(context.Background(), `insert into users(email,name,password_hash) values('refresh-test@example.invalid','test','x') on conflict(email) do update set name=excluded.name returning id::text,name,email::text,role::text`).Scan(&user.ID, &user.Name, &user.Email, &user.Role)
	if err != nil {
		t.Fatal(err)
	}
	first, err := repo.login(context.Background(), user, "test", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.rotate(context.Background(), first.Refresh, "test", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.rotate(context.Background(), first.Refresh, "test", ""); err != errRefreshReuse {
		t.Fatalf("got %v want reuse error", err)
	}
	var active int
	if err = pool.QueryRow(context.Background(), `select count(*) from sessions where user_id=$1 and revoked_at is null`, user.ID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 0 {
		t.Fatalf("reuse left %d active family sessions", active)
	}
}
