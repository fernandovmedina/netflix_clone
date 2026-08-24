package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func TestInvalidSeasonAndMalformedBodiesReturnOneJSONError(t *testing.T) {
	app := &application{}
	cases := []struct {
		name, path string
		handler    http.HandlerFunc
		body       string
		pathKey    string
	}{
		{"season number", "/api/v1/admin/series/1/seasons", app.createSeason, `{"season_number":0}`, "id"},
		{"create season malformed", "/api/v1/admin/series/1/seasons", app.createSeason, `{`, "id"},
		{"patch season malformed", "/api/v1/admin/seasons/1", app.patchSeason, `{`, "id"},
		{"create episode malformed", "/api/v1/admin/seasons/1/episodes", app.createEpisode, `{`, "id"},
		{"patch episode malformed", "/api/v1/admin/episodes/1", app.patchEpisode, `{`, "id"},
		{"create genre malformed", "/api/v1/admin/genres", app.createGenre, `{`, ""},
		{"patch genre malformed", "/api/v1/admin/genres/1", app.patchGenre, `{`, "id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			if tc.pathKey != "" {
				req.SetPathValue(tc.pathKey, "1")
			}
			rec := httptest.NewRecorder()
			tc.handler(rec, req)
			if rec.Code != http.StatusBadRequest || !json.Valid(rec.Body.Bytes()) {
				t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminMovieMetadataAssignmentsAreTransactional(t *testing.T) {
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var genreID, actorID, categoryID int
	if err = pool.QueryRow(context.Background(), `select id_genre from genres where deleted_at is null order by id_genre limit 1`).Scan(&genreID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(context.Background(), `select id_actor from actors where deleted_at is null order by id_actor limit 1`).Scan(&actorID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(context.Background(), `select id_category from categories where deleted_at is null order by id_category limit 1`).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	name := "metadata-assignment-" + uuid.NewString()
	body := fmt.Sprintf(`{"title":%q,"duration":90,"genre_ids":[%d,%d],"actor_ids":[%d],"category_ids":[%d]}`, name, genreID, genreID, actorID, categoryID)
	app := &application{pool: pool}
	mux := http.NewServeMux()
	app.routes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/movies", strings.NewReader(body))
	req.Header.Set("X-User-Role", "admin")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out titleMutationResponse
	if err = json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `delete from title_genres where id_title=$1; delete from title_actors where id_title=$1; delete from title_categories where id_title=$1; delete from movies where id_movie=$2; delete from titles where id_title=$1`, out.TitleID, out.ID)
	}()
	if len(out.GenreIDs) != 1 || out.GenreIDs[0] != genreID || len(out.ActorIDs) != 1 || out.ActorIDs[0] != actorID || len(out.CategoryIDs) != 1 || out.CategoryIDs[0] != categoryID {
		t.Fatalf("unexpected create response: %#v", out)
	}
	var genres, actors, categories int
	if err = pool.QueryRow(context.Background(), `select (select count(*) from title_genres where id_title=$1),(select count(*) from title_actors where id_title=$1),(select count(*) from title_categories where id_title=$1)`, out.TitleID).Scan(&genres, &actors, &categories); err != nil {
		t.Fatal(err)
	}
	if genres != 1 || actors != 1 || categories != 1 {
		t.Fatalf("join counts genres=%d actors=%d categories=%d", genres, actors, categories)
	}
	patchBody := fmt.Sprintf(`{"title":%q,"duration":91,"genre_ids":[],"actor_ids":[%d]}`, name, actorID)
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/movies/%d", out.ID), strings.NewReader(patchBody))
	req.Header.Set("X-User-Role", "admin")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err = pool.QueryRow(context.Background(), `select (select count(*) from title_genres where id_title=$1),(select count(*) from title_actors where id_title=$1),(select count(*) from title_categories where id_title=$1)`, out.TitleID).Scan(&genres, &actors, &categories); err != nil {
		t.Fatal(err)
	}
	if genres != 0 || actors != 1 || categories != 1 {
		t.Fatalf("replacement join counts genres=%d actors=%d categories=%d", genres, actors, categories)
	}
}

func TestAdminSeriesMetadataAssignments(t *testing.T) {
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var genreID, actorID, categoryID int
	if err = pool.QueryRow(context.Background(), `select (select id_genre from genres where deleted_at is null order by id_genre limit 1),(select id_actor from actors where deleted_at is null order by id_actor limit 1),(select id_category from categories where deleted_at is null order by id_category limit 1)`).Scan(&genreID, &actorID, &categoryID); err != nil {
		t.Fatal(err)
	}
	name := "series-metadata-" + uuid.NewString()
	body := fmt.Sprintf(`{"title":%q,"number_of_seasons":1,"genre_ids":[%d],"actor_ids":[%d],"category_ids":[%d]}`, name, genreID, actorID, categoryID)
	app := &application{pool: pool}
	mux := http.NewServeMux()
	app.routes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/series", strings.NewReader(body))
	req.Header.Set("X-User-Role", "admin")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out titleMutationResponse
	if err = json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `delete from title_genres where id_title=$1; delete from title_actors where id_title=$1; delete from title_categories where id_title=$1; delete from seasons where id_series=$2; delete from series where id_series=$2; delete from titles where id_title=$1`, out.TitleID, out.ID)
	}()
	patch := fmt.Sprintf(`{"title":%q,"number_of_seasons":1,"genre_ids":[],"actor_ids":[],"category_ids":[%d]}`, name, categoryID)
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/series/%d", out.ID), strings.NewReader(patch))
	req.Header.Set("X-User-Role", "admin")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err = json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.GenreIDs) != 0 || len(out.ActorIDs) != 0 || len(out.CategoryIDs) != 1 || out.CategoryIDs[0] != categoryID {
		t.Fatalf("unexpected patch response: %#v", out)
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
	if got.Title != prefix || got.SeriesID == nil || *got.SeriesID != seriesID || len(got.Seasons) != 2 {
		t.Fatalf("unexpected series response: %#v", got)
	}
	if got.Seasons[0].SeasonNumber != 1 || got.Seasons[1].SeasonNumber != 2 ||
		len(got.Seasons[0].Episodes) != 2 || got.Seasons[0].Episodes[0].Title != "first" || got.Seasons[0].Episodes[1].Title != "second" {
		t.Fatalf("hierarchy is not ordered: %#v", got.Seasons)
	}
}

func TestMovieRouteUsesMovieIdentifier(t *testing.T) {
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
	var titleID, movieID int
	if err = pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('Movie',$1,false) returning id_title`, name).Scan(&titleID); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(context.Background(), `insert into movies(id_title) values($1) returning id_movie`, titleID).Scan(&movieID); err != nil {
		t.Fatal(err)
	}
	app := &application{pool: pool}
	mux := http.NewServeMux()
	app.routes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/"+strconv.Itoa(movieID), nil)
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
	if got.ID != titleID || got.Title != name || got.MovieID == nil || *got.MovieID != movieID {
		t.Fatalf("unexpected movie: %#v", got)
	}
}

func TestAdminSeesLatestAssetStateAndPublicDoesNot(t *testing.T) {
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	app := &application{pool: pool}
	for _, status := range []string{"pending", "processing", "failed", "ready"} {
		t.Run(status, func(t *testing.T) {
			name := "asset-state-" + status + "-" + uuid.NewString()
			var titleID, movieID int
			if err = pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('Movie',$1,true) returning id_title`, name).Scan(&titleID); err != nil {
				t.Fatal(err)
			}
			if err = pool.QueryRow(context.Background(), `insert into movies(id_title) values($1) returning id_movie`, titleID).Scan(&movieID); err != nil {
				t.Fatal(err)
			}
			if _, err = pool.Exec(context.Background(), `insert into video_assets(id,kind,id_movie,status,source_path,superseded_at) values($1,'movie',$2,'superseded','old',now())`, uuid.New(), movieID); err != nil {
				t.Fatal(err)
			}
			assetID := uuid.New()
			if _, err = pool.Exec(context.Background(), `insert into video_assets(id,kind,id_movie,status,source_path) values($1,'movie',$2,$3,'source')`, assetID, movieID, status); err != nil {
				t.Fatal(err)
			}
			fetch := func(admin bool) []titleItem {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/titles?q="+name, nil)
				if admin {
					req.Header.Set("X-User-Role", "admin")
				}
				rec := httptest.NewRecorder()
				app.listTitles(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
				}
				var items []titleItem
				if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
					t.Fatal(err)
				}
				return items
			}
			adminItems := fetch(true)
			if len(adminItems) != 1 || adminItems[0].MovieID == nil || *adminItems[0].MovieID != movieID || adminItems[0].AssetID == nil || *adminItems[0].AssetID != assetID.String() || adminItems[0].AssetStatus == nil || *adminItems[0].AssetStatus != status {
				t.Fatalf("admin projection=%#v", adminItems)
			}
			detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/movies/"+strconv.Itoa(movieID), nil)
			detailReq.SetPathValue("id", strconv.Itoa(movieID))
			detailReq.Header.Set("X-User-Role", "admin")
			detailRec := httptest.NewRecorder()
			app.getMovie(detailRec, detailReq)
			var detail titleItem
			if detailRec.Code != http.StatusOK || json.NewDecoder(detailRec.Body).Decode(&detail) != nil || detail.AssetStatus == nil || *detail.AssetStatus != status {
				t.Fatalf("admin detail status=%d projection=%#v body=%s", detailRec.Code, detail, detailRec.Body.String())
			}
			publicItems := fetch(false)
			if status != "ready" && len(publicItems) != 0 {
				t.Fatalf("public leaked %s asset: %#v", status, publicItems)
			}
			if status == "ready" && (len(publicItems) != 1 || publicItems[0].MovieID == nil || publicItems[0].AssetStatus != nil) {
				t.Fatalf("public ready projection=%#v", publicItems)
			}
		})
	}
}

func TestVideoLimitCountsFileBytesAndFailedTargetLeavesNoSource(t *testing.T) {
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	root := t.TempDir()
	store, err := storage.NewLocal(root)
	if err != nil {
		t.Fatal(err)
	}
	newMovie := func() int {
		var title, movie int
		if err = pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('Movie',$1,false) returning id_title`, "limit-"+uuid.NewString()).Scan(&title); err != nil {
			t.Fatal(err)
		}
		if err = pool.QueryRow(context.Background(), `insert into movies(id_title) values($1) returning id_movie`, title).Scan(&movie); err != nil {
			t.Fatal(err)
		}
		return movie
	}
	video := func(size int) []byte {
		data := append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}, make([]byte, size-12)...)
		return data
	}
	upload := func(target int, data []byte) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, partErr := writer.CreateFormFile("file", "video.mp4")
		if partErr != nil {
			t.Fatal(partErr)
		}
		if _, partErr = part.Write(data); partErr != nil {
			t.Fatal(partErr)
		}
		if partErr = writer.Close(); partErr != nil {
			t.Fatal(partErr)
		}
		if len(body.Bytes()) <= len(data) {
			t.Fatal("fixture lacks multipart overhead")
		}
		req := httptest.NewRequest(http.MethodPost, "/", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		app := &application{pool: pool, store: store, maxUpload: 1024}
		app.uploadVideo(rec, req, "movie", target)
		return rec
	}
	if rec := upload(newMovie(), video(1024)); rec.Code != http.StatusAccepted {
		t.Fatalf("exact-limit upload status=%d body=%s", rec.Code, rec.Body.String())
	}
	if count := countRegularFiles(t, root); count != 1 {
		t.Fatalf("exact-limit upload files=%d want 1", count)
	}
	if rec := upload(newMovie(), video(1025)); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize upload status=%d body=%s", rec.Code, rec.Body.String())
	}
	if count := countRegularFiles(t, root); count != 1 {
		t.Fatalf("oversize upload left source: files=%d want 1", count)
	}
	before := countRegularFiles(t, root)
	if rec := upload(999999999, video(1024)); rec.Code != http.StatusNotFound {
		t.Fatalf("missing target status=%d body=%s", rec.Code, rec.Body.String())
	}
	if after := countRegularFiles(t, root); after != before {
		t.Fatalf("missing target left source file: before=%d after=%d", before, after)
	}
}

func countRegularFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	if err := filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			count++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return count
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
