package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	shareddb "github.com/fernandovmedina/netflix-clone/microservices/shared/database"
	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/fernandovmedina/netflix-clone/microservices/shared/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type application struct {
	pool      *pgxpool.Pool
	store     storage.Store
	maxUpload int64
	hostname  string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := shareddb.Open(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	store, err := storage.NewLocal(env("MEDIA_ROOT", "/media"))
	if err != nil {
		log.Fatal(err)
	}
	maxUpload, err := strconv.ParseInt(env("MAX_UPLOAD_BYTES", "5368709120"), 10, 64)
	if err != nil || maxUpload < 1 {
		log.Fatal("invalid MAX_UPLOAD_BYTES")
	}
	host, _ := os.Hostname()
	app := &application{pool: pool, store: store, maxUpload: maxUpload, hostname: host}
	mux := http.NewServeMux()
	app.routes(mux)
	server := &http.Server{Addr: ":" + env("PORT", "8080"), Handler: logRequests(host, mux), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("[%s] catalog listening on %s", host, server.Addr)
	log.Fatal(server.ListenAndServe())
}

func (app *application) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		jsonx.Write(w, 200, map[string]string{"status": "ok", "container": app.hostname})
	})
	mux.HandleFunc("GET /api/v1/titles", app.listTitles)
	mux.HandleFunc("GET /api/v1/titles/{id}", app.getTitle)
	mux.HandleFunc("GET /api/v1/movies/{id}", app.getMovie)
	mux.HandleFunc("GET /api/v1/series/{id}", app.getSeries)
	mux.HandleFunc("GET /api/v1/genres", app.listGenres)
	mux.HandleFunc("GET /api/v1/categories", app.listCategories)
	mux.HandleFunc("GET /api/v1/actors", app.listActors)
	mux.HandleFunc("GET /api/v1/home", app.home)
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("POST /api/v1/admin/movies", app.createMovie)
	adminMux.HandleFunc("PATCH /api/v1/admin/movies/{id}", app.patchMovie)
	adminMux.HandleFunc("DELETE /api/v1/admin/movies/{id}", app.deleteMovie)
	adminMux.HandleFunc("POST /api/v1/admin/series", app.createSeries)
	adminMux.HandleFunc("PATCH /api/v1/admin/series/{id}", app.patchSeries)
	adminMux.HandleFunc("DELETE /api/v1/admin/series/{id}", app.deleteSeries)
	adminMux.HandleFunc("POST /api/v1/admin/series/{id}/seasons", app.createSeason)
	adminMux.HandleFunc("POST /api/v1/admin/seasons/{id}/episodes", app.createEpisode)
	adminMux.HandleFunc("PATCH /api/v1/admin/seasons/{id}", app.patchSeason)
	adminMux.HandleFunc("DELETE /api/v1/admin/seasons/{id}", app.deleteSeason)
	adminMux.HandleFunc("PATCH /api/v1/admin/episodes/{id}", app.patchEpisode)
	adminMux.HandleFunc("DELETE /api/v1/admin/episodes/{id}", app.deleteEpisode)
	adminMux.HandleFunc("POST /api/v1/admin/genres", app.createGenre)
	adminMux.HandleFunc("PATCH /api/v1/admin/genres/{id}", app.patchGenre)
	adminMux.HandleFunc("DELETE /api/v1/admin/genres/{id}", app.deleteGenre)
	adminMux.HandleFunc("POST /api/v1/admin/movies/{id}/video", app.uploadMovieVideo)
	adminMux.HandleFunc("POST /api/v1/admin/episodes/{id}/video", app.uploadEpisodeVideo)
	adminMux.HandleFunc("POST /api/v1/admin/titles/{id}/thumbnail", app.uploadThumbnail)
	adminMux.HandleFunc("POST /api/v1/admin/titles/{id}/publish", app.publishTitle)
	adminMux.HandleFunc("GET /api/v1/admin/assets/{id}", app.assetStatus)
	mux.Handle("/api/v1/admin/", app.admin(adminMux))
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func logRequests(host string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &recorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		log.Printf("[%s] %s %s -> %d (%s)", host, r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Millisecond))
	})
}

type recorder struct {
	http.ResponseWriter
	status int
}

func (r *recorder) WriteHeader(status int) { r.status = status; r.ResponseWriter.WriteHeader(status) }
