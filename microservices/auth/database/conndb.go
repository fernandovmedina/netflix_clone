// Package database talks to the Supabase backend: the Postgres database
// (through pgx) and the Supabase Auth (GoTrue) REST API, which owns user
// records and issues the JWTs the whole platform validates against.
package database

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnDB opens a connection pool to the Supabase Postgres instance using
// DATABASE_URL. A pool (rather than a single *pgx.Conn) is required because
// the HTTP server serves requests concurrently.
func ConnDB(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
