package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	shareddb "github.com/fernandovmedina/netflix-clone/microservices/shared/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// hostname identifies which container is serving a request in the logs.
var hostname string

type application struct {
	pool           *pgxpool.Pool
	repo           *repository
	tokens         *tokenManager
	cookieSecure   bool
	dummyHash      []byte
	httpClient     *http.Client
	proxyTransport http.RoundTripper
}

func main() {
	// Inside docker-compose the variables arrive through env_file, so a
	// missing .env.local is fine there.
	if err := godotenv.Load(".env.local"); err != nil {
		log.Println("no .env.local found, using environment variables")
	}

	hostname, _ = os.Hostname()

	tokens, err := newTokenManager()
	if err != nil {
		log.Fatalf("[%s] jwt setup failed: %v", hostname, err)
	}

	// Every replica uses the same database; no authentication state is local.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := shareddb.Open(ctx)
	cancel()
	if err != nil {
		log.Fatalf("[%s] could not connect to Postgres: %v", hostname, err)
	}
	defer pool.Close()
	repo, err := newRepository(pool, tokens)
	if err != nil {
		log.Fatal(err)
	}
	cookieSecure, err := strconv.ParseBool(getenv("COOKIE_SECURE", "false"))
	if err != nil {
		log.Fatalf("COOKIE_SECURE must be true or false: %v", err)
	}
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("constant-time-dummy-password"), 12)
	if err != nil {
		log.Fatal(err)
	}
	app := &application{pool: pool, repo: repo, tokens: tokens, cookieSecure: cookieSecure, dummyHash: dummyHash, httpClient: &http.Client{Timeout: 10 * time.Second}, proxyTransport: http.DefaultTransport}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /api/v1/auth/signup", app.handleSignup)
	mux.HandleFunc("POST /api/v1/auth/login", app.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", app.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/logout", app.handleLogout)
	mux.Handle("GET /api/v1/auth/me", app.requireAuth(http.HandlerFunc(app.handleMe)))
	mux.HandleFunc("GET /api/v1/auth/google", app.handleGoogleStart)
	mux.HandleFunc("GET /api/v1/auth/google/callback", app.handleGoogleCallback)
	mux.Handle("/api/v1/admin", app.requireAdmin(http.HandlerFunc(app.handleProxy)))
	mux.Handle("/api/v1/admin/", app.requireAdmin(http.HandlerFunc(app.handleProxy)))
	// Everything else under /api/v1/ belongs to another microservice: check
	// the session, then redirect to the correspondent service.
	mux.Handle("/api/v1/", app.requireAuth(http.HandlerFunc(app.handleProxy)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           logRequests(withCORS(mux)),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("[%s] auth service listening on :%s", hostname, port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[%s] server stopped: %v", hostname, err)
	}
}
