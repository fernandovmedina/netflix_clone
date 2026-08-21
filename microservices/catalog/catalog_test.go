package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestVideoValidationRejectsDisguisedFiles(t *testing.T) {
	for name, data := range map[string][]byte{"php": []byte("<?php echo 'owned'; ?>"), "elf": append([]byte{0x7f, 'E', 'L', 'F'}, make([]byte, 508)...)} {
		mime := sniffed{kind: http.DetectContentType(data), head: data}
		if validVideo(".mp4", mime) {
			t.Errorf("accepted disguised %s as %s", name, mime.kind)
		}
	}
}

func TestUploadRejectsDisguisedMP4(t *testing.T) {
	for name, data := range map[string][]byte{"php": []byte("<?php echo 'owned'; ?>"), "elf": append([]byte{0x7f, 'E', 'L', 'F'}, make([]byte, 508)...)} {
		t.Run(name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", "attack.mp4")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = part.Write(data); err != nil {
				t.Fatal(err)
			}
			if err = writer.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/movies/1/video", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			rec := httptest.NewRecorder()
			app := &application{maxUpload: 1 << 20}
			app.uploadVideo(rec, req, "movie", 1)
			if rec.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
func TestNonAdminVisibility(t *testing.T) {
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	insert := func(name string, published bool, status string) {
		var title, movie int
		if err := pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('Movie',$1,$2) returning id_title`, name, published).Scan(&title); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(context.Background(), `insert into movies(id_title) values($1) returning id_movie`, title).Scan(&movie); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), `insert into video_assets(kind,id_movie,status,source_path) values('movie',$1,$2,'x')`, movie, status); err != nil {
			t.Fatal(err)
		}
	}
	prefix := "visibility-" + uuid.NewString()
	insert(prefix+"-unpublished", false, "ready")
	insert(prefix+"-pending", true, "pending")
	insert(prefix+"-visible", true, "ready")
	app := &application{pool: pool}
	req := httptest.NewRequest("GET", "/api/v1/titles?q="+prefix, nil)
	rec := httptest.NewRecorder()
	app.listTitles(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var items []titleItem
	if err = json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != prefix+"-visible" {
		t.Fatalf("unexpected visible items: %#v", items)
	}
}

func TestSeriesUsesResolvedSeriesIDAndOrdersHierarchy(t *testing.T) {
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	prefix := "series-hierarchy-" + uuid.NewString()
	// Deliberately consume a title ID so id_title and id_series cannot align by accident.
	if _, err = pool.Exec(context.Background(), `insert into titles(type,title,published) values('Movie',$1,false)`, prefix+"-offset"); err != nil {
		t.Fatal(err)
	}
	var titleID, seriesID int
	if err = pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('TV Show',$1,false) returning id_title`, prefix).Scan(&titleID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(context.Background(), `insert into series(id_title,number_of_seasons) values($1,2) returning id_series`, titleID).Scan(&seriesID); err != nil {
		t.Fatal(err)
	}
	if titleID == seriesID {
		t.Fatal("test fixture did not create distinct title and series identifiers")
	}
	var seasonOne, seasonTwo int
	if err = pool.QueryRow(context.Background(), `insert into seasons(id_series,season_number,number_of_episodes) values($1,2,1) returning id_season`, seriesID).Scan(&seasonTwo); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(context.Background(), `insert into seasons(id_series,season_number,number_of_episodes) values($1,1,2) returning id_season`, seriesID).Scan(&seasonOne); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `insert into episodes(id_season,episode_number,title) values($1,2,'second'),($1,1,'first'),($2,1,'third')`, seasonOne, seasonTwo); err != nil {
		t.Fatal(err)
	}
	app := &application{pool: pool}
	mux := http.NewServeMux()
	app.routes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/series/"+strconv.Itoa(titleID), nil)
	req.Header.Set("X-User-Role", "admin")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got seriesResponse
	if err = json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Title != prefix || got.SeriesID != seriesID || len(got.Seasons) != 2 {
		t.Fatalf("unexpected series response: %#v", got)
	}
	if got.Seasons[0].SeasonNumber != 1 || got.Seasons[1].SeasonNumber != 2 ||
		len(got.Seasons[0].Episodes) != 2 || got.Seasons[0].Episodes[0].Title != "first" || got.Seasons[0].Episodes[1].Title != "second" {
		t.Fatalf("hierarchy is not ordered: %#v", got.Seasons)
	}
}

func TestMovieRouteUsesTitleIdentifier(t *testing.T) {
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	name := "movie-id-contract-" + uuid.NewString()
	var titleID int
	if err = pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('Movie',$1,false) returning id_title`, name).Scan(&titleID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `insert into movies(id_title) values($1)`, titleID); err != nil {
		t.Fatal(err)
	}
	app := &application{pool: pool}
	mux := http.NewServeMux()
	app.routes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/"+strconv.Itoa(titleID), nil)
	req.Header.Set("X-User-Role", "admin")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got titleItem
	if err = json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != titleID || got.Title != name {
		t.Fatalf("unexpected movie: %#v", got)
	}
}

func TestReuploadSupersedesAssetAndRetainsHistory(t *testing.T) {
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, movieID := func() (int, int) {
		var title, movie int
		if err = pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('Movie',$1,false) returning id_title`, "reupload-"+uuid.NewString()).Scan(&title); err != nil {
			t.Fatal(err)
		}
		if err = pool.QueryRow(context.Background(), `insert into movies(id_title) values($1) returning id_movie`, title).Scan(&movie); err != nil {
			t.Fatal(err)
		}
		return title, movie
	}()
	previous, job := uuid.New(), uuid.New()
	if _, err = pool.Exec(context.Background(), `insert into video_assets(id,kind,id_movie,status,source_path,manifest_path) values($1,'movie',$2,'ready','sources/old/source.mp4','hls/old/master.m3u8')`, previous, movieID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `insert into video_jobs(id,asset_id,status) values($1,$2,'queued')`, job, previous); err != nil {
		t.Fatal(err)
	}
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "replacement.mp4")
	if err != nil {
		t.Fatal(err)
	}
	video := append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}, make([]byte, 500)...)
	if _, err = part.Write(video); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/movies/1/video", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	app := &application{pool: pool, store: store, maxUpload: 1 << 20}
	app.uploadVideo(rec, req, "movie", movieID)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		AssetID uuid.UUID `json:"asset_id"`
	}
	if err = json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.AssetID == previous {
		t.Fatal("re-upload reused the historical asset id")
	}
	var oldStatus, oldManifest, jobStatus, newStatus string
	var supersededAt *time.Time
	if err = pool.QueryRow(context.Background(), `select status::text,manifest_path,superseded_at from video_assets where id=$1`, previous).Scan(&oldStatus, &oldManifest, &supersededAt); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(context.Background(), `select status::text from video_jobs where id=$1`, job).Scan(&jobStatus); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(context.Background(), `select status::text from video_assets where id=$1`, response.AssetID).Scan(&newStatus); err != nil {
		t.Fatal(err)
	}
	if oldStatus != "superseded" || supersededAt == nil || oldManifest != "hls/old/master.m3u8" || jobStatus != "failed" || newStatus != "pending" {
		t.Fatalf("old=%s superseded_at=%v manifest=%s job=%s new=%s", oldStatus, supersededAt, oldManifest, jobStatus, newStatus)
	}
}
