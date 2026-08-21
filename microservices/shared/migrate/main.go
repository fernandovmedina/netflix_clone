package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const advisoryLockID int64 = 0x4e46584d494752 // "NFXMIGR"

func run(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "select pg_advisory_lock($1)", advisoryLockID); err != nil {
		return err
	}
	defer func() { _, _ = conn.Exec(context.Background(), "select pg_advisory_unlock($1)", advisoryLockID) }()
	if _, err := conn.Exec(ctx, `create table if not exists schema_migrations (version text primary key, applied_at timestamptz not null default now())`); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var applied bool
		if err := conn.QueryRow(ctx, "select exists(select 1 from schema_migrations where version=$1)", name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sql, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(sql)); err == nil {
			_, err = tx.Exec(ctx, "insert into schema_migrations(version) values($1)", name)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		log.Printf("applied migration %s", name)
	}
	return bootstrapAdmin(ctx, conn.Conn())
}

func bootstrapAdmin(ctx context.Context, conn *pgx.Conn) error {
	email, hash, err := hashAdminPassword()
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, `
		insert into users(email,password_hash,name,role,email_verified)
		values($1,$2,'Administrator','admin',true)
		on conflict(email) do update set role='admin', password_hash=coalesce(users.password_hash,excluded.password_hash), updated_at=now()`, email, hash)
	return err
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "database/migrations"
	}
	if err := run(ctx, pool, dir); err != nil {
		log.Fatal(err)
	}
}

func adminCredentials() (string, string) {
	email, password := os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")
	if email == "" {
		email = "admin@netflix.local"
	}
	if password == "" {
		password = "admin12345"
	}
	return strings.TrimSpace(strings.ToLower(email)), password
}

func hashAdminPassword() (string, string, error) {
	email, password := adminCredentials()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return email, string(hash), err
}
