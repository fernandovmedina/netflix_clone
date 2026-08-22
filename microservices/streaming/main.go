package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	shareddb "github.com/fernandovmedina/netflix-clone/microservices/shared/database"
	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var qualityPattern = regexp.MustCompile(`^\d{3,4}p$`)
var segmentPattern = regexp.MustCompile(`^seg_\d{5}\.ts$`)
var thumbnailPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

type application struct {
	pool     *pgxpool.Pool
	root     string
	hostname string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := shareddb.Open(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	root, err := filepath.Abs(env("MEDIA_ROOT", "/media"))
	if err != nil {
		log.Fatal(err)
	}
	host, _ := os.Hostname()
	app := &application{pool: pool, root: root, hostname: host}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		jsonx.Write(w, 200, map[string]string{"status": "ok", "container": host})
	})
	mux.HandleFunc("GET /api/v1/stream/{path...}", app.dispatch)
	server := &http.Server{Addr: ":" + env("PORT", "8080"), Handler: rejectUnsafePath(mux), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("[%s] streaming listening on %s", host, server.Addr)
	log.Fatal(server.ListenAndServe())
}
func rejectUnsafePath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.ToLower(strings.SplitN(r.RequestURI, "?", 2)[0])
		if strings.Contains(r.URL.Path, "\x00") || strings.Contains(r.URL.Path, "..") || strings.Contains(r.URL.Path, "．．") || strings.Contains(r.URL.Path, "\\") || strings.Contains(r.URL.Path, "//") || strings.Contains(raw, "%00") || strings.Contains(raw, "%2e") || strings.Contains(raw, "%2f") || strings.Contains(raw, "%5c") {
			jsonx.Error(w, 400, "invalid path")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (app *application) dispatch(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.PathValue("path"), "/")
	switch {
	case len(parts) == 2 && parts[0] == "thumbnails":
		r.SetPathValue("file", parts[1])
		app.thumbnail(w, r)
	case len(parts) == 2 && parts[1] == "master.m3u8":
		r.SetPathValue("asset", parts[0])
		app.master(w, r)
	case len(parts) == 3 && parts[2] == "playlist.m3u8":
		r.SetPathValue("asset", parts[0])
		r.SetPathValue("quality", parts[1])
		app.playlist(w, r)
	case len(parts) == 3:
		r.SetPathValue("asset", parts[0])
		r.SetPathValue("quality", parts[1])
		r.SetPathValue("segment", parts[2])
		app.segment(w, r)
	default:
		jsonx.Error(w, 400, "invalid streaming path")
	}
}
func env(k, f string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return f
}
func (app *application) master(w http.ResponseWriter, r *http.Request) {
	id, ok := app.asset(w, r)
	if !ok {
		return
	}
	app.serve(w, r, []string{"hls", id.String(), "master.m3u8"}, "application/vnd.apple.mpegurl", "public, max-age=10")
}
func (app *application) playlist(w http.ResponseWriter, r *http.Request) {
	id, ok := app.asset(w, r)
	if !ok {
		return
	}
	quality := r.PathValue("quality")
	if !qualityPattern.MatchString(quality) {
		jsonx.Error(w, 400, "invalid quality")
		return
	}
	app.serve(w, r, []string{"hls", id.String(), quality, "playlist.m3u8"}, "application/vnd.apple.mpegurl", "public, max-age=10")
}
func (app *application) segment(w http.ResponseWriter, r *http.Request) {
	id, ok := app.asset(w, r)
	if !ok {
		return
	}
	quality, segment := r.PathValue("quality"), r.PathValue("segment")
	if !qualityPattern.MatchString(quality) || !segmentPattern.MatchString(segment) {
		jsonx.Error(w, 400, "invalid segment path")
		return
	}
	app.serve(w, r, []string{"hls", id.String(), quality, segment}, "video/mp2t", "public, max-age=31536000, immutable")
}
func (app *application) thumbnail(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if !validThumbnail(file) {
		jsonx.Error(w, 400, "invalid thumbnail path")
		return
	}
	app.serve(w, r, []string{"thumbnails", file}, mimeForImage(file), "public, max-age=86400")
}
func (app *application) asset(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("asset"))
	if err != nil {
		jsonx.Error(w, 400, "invalid asset id")
		return uuid.Nil, false
	}
	ready, err := app.ready(r.Context(), id)
	if err != nil {
		jsonx.Error(w, 500, "internal server error")
		return uuid.Nil, false
	}
	if !ready {
		jsonx.Error(w, 404, "asset not ready")
		return uuid.Nil, false
	}
	return id, true
}
func (app *application) ready(ctx context.Context, id uuid.UUID) (bool, error) {
	var ready bool
	err := app.pool.QueryRow(ctx, `select exists(
		select 1 from video_assets va
		join movies m on m.id_movie=va.id_movie and m.deleted_at is null
		join titles t on t.id_title=m.id_title and t.deleted_at is null and t.published=true
		where va.id=$1 and va.status='ready'
		union all
		select 1 from video_assets va
		join episodes e on e.id_episode=va.id_episode and e.deleted_at is null
		join seasons se on se.id_season=e.id_season and se.deleted_at is null
		join series s on s.id_series=se.id_series and s.deleted_at is null
		join titles t on t.id_title=s.id_title and t.deleted_at is null and t.published=true
		where va.id=$1 and va.status='ready'
	)`, id).Scan(&ready)
	return ready, err
}
func (app *application) serve(w http.ResponseWriter, r *http.Request, components []string, contentType, cacheControl string) {
	path, ok := safeJoin(app.root, components...)
	if !ok {
		jsonx.Error(w, 400, "invalid path")
		return
	}
	resolved, err := resolvedPath(app.root, path)
	if errors.Is(err, fs.ErrNotExist) {
		jsonx.Error(w, 404, "file not found")
		return
	}
	if err != nil {
		jsonx.Error(w, 400, "invalid path")
		return
	}
	file, err := os.Open(resolved)
	if errors.Is(err, fs.ErrNotExist) {
		jsonx.Error(w, 404, "file not found")
		return
	}
	if err != nil {
		jsonx.Error(w, 500, "internal server error")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		jsonx.Error(w, 404, "file not found")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", cacheControl)
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}

func resolvedPath(root, path string) (string, error) {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("resolved path escapes media root")
	}
	return resolved, nil
}
func safeJoin(root string, components ...string) (string, bool) {
	for _, component := range components {
		if component == "" || component == "." || component == ".." || strings.ContainsRune(component, '\x00') || strings.ContainsAny(component, `/\`) || filepath.IsAbs(component) {
			return "", false
		}
	}
	path := filepath.Join(append([]string{root}, components...)...)
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return absolute, true
}
func validThumbnail(file string) bool {
	return thumbnailPattern.MatchString(file) && file != "." && file != ".." && !strings.Contains(file, "..")
}
func mimeForImage(file string) string {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
