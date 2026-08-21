package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/fernandovmedina/netflix-clone/microservices/auth/database"
)

// hostname identifies which container is serving a request in the logs.
var hostname string

func main() {
	// Inside docker-compose the variables arrive through env_file, so a
	// missing .env.local is fine there.
	if err := godotenv.Load(".env.local"); err != nil {
		log.Println("no .env.local found, using environment variables")
	}

	hostname, _ = os.Hostname()

	if err := initJWT(); err != nil {
		log.Fatalf("[%s] jwt setup failed: %v", hostname, err)
	}
	if jwksCleanup != nil {
		defer jwksCleanup()
	}

	// Direct Postgres pool. No endpoint queries it yet, but it is the
	// connection future handlers (and health details) will use.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := database.ConnDB(ctx)
	cancel()
	if err != nil {
		log.Printf("[%s] warning: could not connect to Supabase Postgres: %v", hostname, err)
	} else {
		defer pool.Close()
		log.Printf("[%s] connected to Supabase Postgres", hostname)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /api/v1/auth/signup", handleSignup)
	mux.HandleFunc("POST /api/v1/auth/login", handleLogin)
	mux.Handle("GET /api/v1/auth/user", requireAuth(http.HandlerFunc(handleUser)))
	// Everything else under /api/v1/ belongs to another microservice: check
	// the session, then redirect to the correspondent service.
	mux.Handle("/api/v1/", requireAuth(http.HandlerFunc(handleProxy)))

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
