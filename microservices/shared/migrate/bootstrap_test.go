package main

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBootstrapAdminDoesNotRewriteCurrentAdmin(t *testing.T) {
	dsn := os.Getenv("PHASE8_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE8_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	email, _ := adminCredentials()
	var before string
	if err = pool.QueryRow(ctx, `select md5(to_jsonb(users)::text) from users where email=$1`, email).Scan(&before); err != nil {
		t.Fatal(err)
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if err = bootstrapAdmin(ctx, conn.Conn()); err != nil {
		t.Fatal(err)
	}
	var after string
	if err = pool.QueryRow(ctx, `select md5(to_jsonb(users)::text) from users where email=$1`, email).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("bootstrap rewrote current admin: before=%s after=%s", before, after)
	}
	t.Logf("admin row fingerprint unchanged: %s", after)
}
