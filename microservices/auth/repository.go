package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errInvalidCredentials = errors.New("invalid credentials")
	errEmailExists        = errors.New("email already exists")
	errInvalidRefresh     = errors.New("invalid refresh token")
	errRefreshReuse       = errors.New("refresh token reuse detected")
)

type User struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
}
type sessionTokens struct {
	Access  string
	Refresh string
	User    User
}
type repository struct {
	pool       *pgxpool.Pool
	tokens     *tokenManager
	refreshTTL time.Duration
}

func newRepository(pool *pgxpool.Pool, tokens *tokenManager) (*repository, error) {
	ttl := 720 * time.Hour
	if value := getenv("REFRESH_TOKEN_TTL", "720h"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return nil, err
		}
		ttl = parsed
	}
	return &repository{pool: pool, tokens: tokens, refreshTTL: ttl}, nil
}

func newRefreshToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}
func refreshHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (repo *repository) createUser(ctx context.Context, name, email, passwordHash, userAgent, ip string) (sessionTokens, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return sessionTokens{}, err
	}
	defer tx.Rollback(ctx)
	var user User
	err = tx.QueryRow(ctx, `insert into users(name,email,password_hash,email_verified) values($1,$2,$3,true) returning id::text,name,email::text,role::text`, name, email, passwordHash).Scan(&user.ID, &user.Name, &user.Email, &user.Role)
	if err != nil {
		if isUniqueViolation(err) {
			return sessionTokens{}, errEmailExists
		}
		return sessionTokens{}, err
	}
	tokens, err := repo.issue(ctx, tx, user, uuid.New(), uuid.Nil, userAgent, ip)
	if err != nil {
		return sessionTokens{}, err
	}
	return tokens, tx.Commit(ctx)
}

func (repo *repository) findByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := repo.pool.QueryRow(ctx, `select id::text,name,email::text,role::text,coalesce(password_hash,'') from users where email=$1 and deleted_at is null`, email).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.PasswordHash)
	return u, err
}

func (repo *repository) login(ctx context.Context, user User, userAgent, ip string) (sessionTokens, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return sessionTokens{}, err
	}
	defer tx.Rollback(ctx)
	tokens, err := repo.issue(ctx, tx, user, uuid.New(), uuid.Nil, userAgent, ip)
	if err != nil {
		return sessionTokens{}, err
	}
	return tokens, tx.Commit(ctx)
}

func (repo *repository) issue(ctx context.Context, tx pgx.Tx, user User, family, rotatedFrom uuid.UUID, userAgent, ip string) (sessionTokens, error) {
	refresh, hash, err := newRefreshToken()
	if err != nil {
		return sessionTokens{}, err
	}
	access, err := repo.tokens.sign(user)
	if err != nil {
		return sessionTokens{}, err
	}
	var rotated any
	if rotatedFrom != uuid.Nil {
		rotated = rotatedFrom
	}
	_, err = tx.Exec(ctx, `insert into sessions(id,user_id,session_family,refresh_token_hash,rotated_from,user_agent,ip,expires_at) values($1,$2,$3,$4,$5,$6,$7,$8)`, uuid.New(), user.ID, family, hash, rotated, nullString(userAgent), parseIP(ip), time.Now().Add(repo.refreshTTL))
	if err != nil {
		return sessionTokens{}, err
	}
	return sessionTokens{Access: access, Refresh: refresh, User: user}, nil
}

func (repo *repository) rotate(ctx context.Context, raw, userAgent, ip string) (sessionTokens, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return sessionTokens{}, err
	}
	defer tx.Rollback(ctx)
	var oldID, family uuid.UUID
	var revoked *time.Time
	var expires time.Time
	var user User
	err = tx.QueryRow(ctx, `select s.id,s.session_family,s.revoked_at,s.expires_at,u.id::text,u.name,u.email::text,u.role::text from sessions s join users u on u.id=s.user_id where s.refresh_token_hash=$1`, refreshHash(raw)).Scan(&oldID, &family, &revoked, &expires, &user.ID, &user.Name, &user.Email, &user.Role)
	if err == pgx.ErrNoRows {
		return sessionTokens{}, errInvalidRefresh
	}
	if err != nil {
		return sessionTokens{}, err
	}
	if revoked != nil {
		if _, err = tx.Exec(ctx, `update sessions set revoked_at=coalesce(revoked_at,now()) where session_family=$1`, family); err != nil {
			return sessionTokens{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return sessionTokens{}, err
		}
		return sessionTokens{}, errRefreshReuse
	}
	if time.Now().After(expires) {
		_, _ = tx.Exec(ctx, `update sessions set revoked_at=now() where id=$1 and revoked_at is null`, oldID)
		_ = tx.Commit(ctx)
		return sessionTokens{}, errInvalidRefresh
	}
	var won uuid.UUID
	err = tx.QueryRow(ctx, `update sessions set revoked_at=now() where id=$1 and revoked_at is null returning id`, oldID).Scan(&won)
	if err == pgx.ErrNoRows {
		if _, err = tx.Exec(ctx, `update sessions set revoked_at=coalesce(revoked_at,now()) where session_family=$1`, family); err != nil {
			return sessionTokens{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return sessionTokens{}, err
		}
		return sessionTokens{}, errRefreshReuse
	}
	if err != nil {
		return sessionTokens{}, err
	}
	tokens, err := repo.issue(ctx, tx, user, family, oldID, userAgent, ip)
	if err != nil {
		return sessionTokens{}, err
	}
	return tokens, tx.Commit(ctx)
}

func (repo *repository) logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	_, err := repo.pool.Exec(ctx, `update sessions set revoked_at=coalesce(revoked_at,now()) where refresh_token_hash=$1`, refreshHash(raw))
	return err
}
func (repo *repository) userByID(ctx context.Context, id string) (User, error) {
	var u User
	err := repo.pool.QueryRow(ctx, `select id::text,name,email::text,role::text from users where id=$1 and deleted_at is null`, id).Scan(&u.ID, &u.Name, &u.Email, &u.Role)
	return u, err
}

func parseIP(remote string) any {
	host, _, err := net.SplitHostPort(remote)
	if err == nil {
		return net.ParseIP(host)
	}
	return net.ParseIP(remote)
}
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
