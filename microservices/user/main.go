package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/authctx"
	shareddb "github.com/fernandovmedina/netflix-clone/microservices/shared/database"
	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type application struct {
	pool              *pgxpool.Pool
	hostname          string
	simulationEnabled bool
}

type userKey struct{}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := shareddb.Open(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	host, _ := os.Hostname()
	simulation, err := strconv.ParseBool(env("PAYMENTS_SIMULATION_ENABLED", "true"))
	if err != nil {
		log.Fatal("PAYMENTS_SIMULATION_ENABLED must be true or false")
	}
	app := &application{pool: pool, hostname: host, simulationEnabled: simulation}
	mux := http.NewServeMux()
	app.routes(mux)
	server := &http.Server{Addr: ":" + env("PORT", "8080"), Handler: logRequests(host, mux), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("[%s] user service listening on %s", host, server.Addr)
	log.Fatal(server.ListenAndServe())
}

func (app *application) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		jsonx.Write(w, http.StatusOK, map[string]string{"status": "ok", "container": app.hostname})
	})
	api := http.NewServeMux()
	api.HandleFunc("GET /api/v1/progress/continue", app.continueWatching)
	api.HandleFunc("GET /api/v1/progress/{kind}/{id}", app.getProgress)
	api.HandleFunc("PUT /api/v1/progress/{kind}/{id}", app.putProgress)
	api.HandleFunc("GET /api/v1/favorites", app.listFavorites)
	api.HandleFunc("POST /api/v1/favorites", app.addFavorite)
	api.HandleFunc("DELETE /api/v1/favorites/{id}", app.deleteFavorite)
	api.HandleFunc("GET /api/v1/profiles", app.listProfiles)
	api.HandleFunc("POST /api/v1/profiles", app.createProfile)
	api.HandleFunc("GET /api/v1/profiles/{id}", app.getProfile)
	api.HandleFunc("PATCH /api/v1/profiles/{id}", app.patchProfile)
	api.HandleFunc("DELETE /api/v1/profiles/{id}", app.deleteProfile)
	api.HandleFunc("GET /api/v1/plans", app.listPlans)
	api.HandleFunc("POST /api/v1/discounts/validate", app.previewDiscount)
	api.HandleFunc("POST /api/v1/payments/card", app.cardPayment)
	api.HandleFunc("POST /api/v1/payments/oxxo", app.oxxoPayment)
	api.HandleFunc("POST /api/v1/payments/oxxo/{ref}/simulate-payment", app.simulateOXXOPayment)
	api.HandleFunc("GET /api/v1/payments/{id}", app.getPayment)
	mux.Handle("/api/v1/", app.authenticated(api))
}

func (app *application) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := authctx.FromHeaders(r)
		id, err := uuid.Parse(identity.ID)
		if err != nil {
			jsonx.Error(w, http.StatusUnauthorized, "authenticated user identity is required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, id)))
	})
}

func userID(r *http.Request) uuid.UUID { return r.Context().Value(userKey{}).(uuid.UUID) }

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		jsonx.Error(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func pathPositiveInt(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(name))
	if err != nil || id < 1 {
		jsonx.Error(w, http.StatusBadRequest, "invalid "+name)
		return 0, false
	}
	return id, true
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("user service error: %v", err)
	jsonx.Error(w, http.StatusInternalServerError, "internal server error")
}

func logRequests(host string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s (%s)", host, r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
