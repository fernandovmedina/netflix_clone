package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUnsafeRequestPathsReturnBadRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := rejectUnsafePath(next)
	cases := []string{"/api/v1/stream/../secret", "/api/v1/stream/..%2fsecret", "/api/v1/stream//etc/passwd", "/api/v1/stream/x/%00", "/api/v1/stream/．．/x", "/api/v1/stream/x/..\\secret"}
	for _, target := range cases {
		req := httptest.NewRequest(http.MethodGet, "http://stream.test/", nil)
		req.RequestURI = target
		req.URL.Path = target
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%q returned %d", target, rec.Code)
		}
	}
}

func TestSymlinkOutsideMediaRootIsRefusedAndMissingIsNotFound(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	asset := uuid.NewString()
	dir := filepath.Join(root, "hls", asset)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("must not leak"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(dir, "master.m3u8")); err != nil {
		t.Fatal(err)
	}
	app := &application{root: root}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.serve(rec, req, []string{"hls", asset, "master.m3u8"}, "application/vnd.apple.mpegurl", "public, max-age=10")
	if rec.Code == http.StatusOK || rec.Body.String() == "must not leak" {
		t.Fatalf("symlink escaped media root: status=%d body=%q", rec.Code, rec.Body.String())
	}
	missing := httptest.NewRecorder()
	app.serve(missing, req, []string{"hls", asset, "missing.m3u8"}, "application/vnd.apple.mpegurl", "public, max-age=10")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing file status=%d body=%s", missing.Code, missing.Body.String())
	}
}

func TestUnpublishedReadyAssetIsNotStreamableOnAnyMediaRoute(t *testing.T) {
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var title, movie int
	if err = pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('Movie',$1,false) returning id_title`, "unpublished-stream-"+uuid.NewString()).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(context.Background(), `insert into movies(id_title) values($1) returning id_movie`, title).Scan(&movie); err != nil {
		t.Fatal(err)
	}
	asset := uuid.New()
	if _, err = pool.Exec(context.Background(), `insert into video_assets(id,kind,id_movie,status,source_path) values($1,'movie',$2,'ready','source')`, asset, movie); err != nil {
		t.Fatal(err)
	}
	app := &application{pool: pool, root: t.TempDir()}
	for name, call := range map[string]func(http.ResponseWriter, *http.Request){"master": app.master, "playlist": app.playlist, "segment": app.segment} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetPathValue("asset", asset.String())
			req.SetPathValue("quality", "720p")
			req.SetPathValue("segment", "seg_00000.ts")
			rec := httptest.NewRecorder()
			call(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPathTraversalRejected(t *testing.T) {
	invalid := []string{"../", "..%2f", "/etc/passwd", "\x00", "%00", "．．", "..\\secret"}
	for _, value := range invalid {
		if _, err := uuid.Parse(value); err == nil {
			t.Errorf("asset accepted %q", value)
		}
		if qualityPattern.MatchString(value) {
			t.Errorf("quality accepted %q", value)
		}
		if segmentPattern.MatchString(value) {
			t.Errorf("segment accepted %q", value)
		}
		if validThumbnail(value) {
			t.Errorf("thumbnail accepted %q", value)
		}
	}
}
func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	for _, parts := range [][]string{{"..", "secret"}, {"/etc/passwd"}, {"hls", "..", "secret"}, {"hls", "\x00"}} {
		if path, ok := safeJoin(root, parts...); ok {
			t.Errorf("safeJoin accepted %q as %q", parts, path)
		}
	}
	want := filepath.Join(root, "hls", "asset", "master.m3u8")
	if got, ok := safeJoin(root, "hls", "asset", "master.m3u8"); !ok || got != want {
		t.Fatalf("safe path got %q,%v want %q", got, ok, want)
	}
}
