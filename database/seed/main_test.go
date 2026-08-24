package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGenerateSeedSQLIsStandaloneAndExcludesMedia(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "exec.sql")
	root := filepath.Clean(filepath.Join("..", "..", "seed"))
	if err := generateSeedSQL(root, destination); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, required := range []string{"INSERT INTO titles", "INSERT INTO movies", "INSERT INTO series", "INSERT INTO seasons", "INSERT INTO episodes", "title_genres", "title_actors", "title_categories", "ON CONFLICT"} {
		if !strings.Contains(sql, required) {
			t.Errorf("generated SQL is missing %q", required)
		}
	}
	for _, forbidden := range []string{"video_assets", "video_jobs"} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("generated SQL contains %q", forbidden)
		}
	}
}

func TestSeedVideoPathDefaultsShortAndValidatesSelection(t *testing.T) {
	root := t.TempDir()
	short := filepath.Join(root, "video", "video-short.mp4")
	long := filepath.Join(root, "video", "video.mp4")
	for selection, want := range map[string]string{"": short, "short": short, " SHORT ": short, "long": long, "LONG": long} {
		got, err := seedVideoPath(root, selection)
		if err != nil || got != want {
			t.Errorf("seedVideoPath(%q) = %q, %v; want %q", selection, got, err, want)
		}
	}
	if _, err := seedVideoPath(root, "trailer"); err == nil {
		t.Fatal("invalid video_source was accepted")
	}
}

func TestSeedManifestSourceInventory(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "seed"))
	movies, seriesData, err := manifests(root)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{"long": 0, "short": 0}
	for _, item := range movies.Movies {
		selection := strings.ToLower(strings.TrimSpace(item.VideoSource))
		if selection == "" {
			selection = "short"
		}
		if _, err = seedVideoPath(root, selection); err != nil {
			t.Fatalf("movie %q: %v", item.Name, err)
		}
		counts[selection]++
	}
	for _, item := range seriesData.Series {
		selection := strings.ToLower(strings.TrimSpace(item.VideoSource))
		if selection == "" {
			selection = "short"
		}
		if _, err = seedVideoPath(root, selection); err != nil {
			t.Fatalf("series %q: %v", item.Name, err)
		}
		for _, season := range item.Seasons {
			counts[selection] += len(season.Episodes)
		}
	}
	if counts["long"] != 6 || counts["short"] != 72 {
		t.Fatalf("source inventory=%v, want long=6 short=72", counts)
	}
	t.Logf("source inventory: long=%d short=%d", counts["long"], counts["short"])
}

func TestResetFailureRestoresDatabaseAndMedia(t *testing.T) {
	dsn := os.Getenv("PHASE6_RESET_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE6_RESET_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	mediaRoot := t.TempDir()
	markers := map[string]string{
		"hls/existing/master.m3u8":       "existing playlist\n",
		"sources/existing/source.mp4":    "existing source\n",
		"thumbnails/uploaded/custom.jpg": "existing thumbnail\n",
	}
	for name, contents := range markers {
		path := filepath.Join(mediaRoot, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, []byte(contents), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	seedRoot := t.TempDir()
	fixtures := map[string]string{
		"movies/seed.json": `{"movies":[{"name":"Broken reset","year_released":2026,"description":"failure fixture","genres":[],"cast":[],"director":"","duration":1,"thumbnail_url":"movies/data/missing.jpg"}]}`,
		"series/seed.json": `{"series":[]}`,
		"video/video.mp4":  "seed video\n",
	}
	for name, contents := range fixtures {
		path := filepath.Join(seedRoot, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := storage.NewLocal(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	databaseBefore := catalogFingerprint(t, pool)
	mediaBefore := mediaFingerprint(t, mediaRoot)
	if _, err = resetAndImport(ctx, pool, store, seedRoot, mediaRoot); err == nil || !strings.Contains(err.Error(), "missing.jpg") {
		t.Fatalf("reset error = %v, want missing artwork failure", err)
	}
	databaseAfter := catalogFingerprint(t, pool)
	mediaAfter := mediaFingerprint(t, mediaRoot)
	if !reflect.DeepEqual(databaseAfter, databaseBefore) {
		t.Fatalf("database changed after failed reset\nbefore: %#v\nafter:  %#v", databaseBefore, databaseAfter)
	}
	if !reflect.DeepEqual(mediaAfter, mediaBefore) {
		t.Fatalf("media changed after failed reset\nbefore: %#v\nafter:  %#v", mediaBefore, mediaAfter)
	}
	t.Logf("failed reset preserved all %d catalog table fingerprints and %d media entries", len(databaseBefore), len(mediaBefore))
}

func catalogFingerprint(t *testing.T, pool *pgxpool.Pool) map[string]string {
	t.Helper()
	tables := []string{"video_jobs", "video_assets", "watch_progress", "favorites", "title_actors", "title_categories", "title_genres", "episodes", "seasons", "movies", "series", "titles", "actors", "categories", "genres"}
	result := make(map[string]string, len(tables))
	for _, table := range tables {
		query := fmt.Sprintf(`select md5(coalesce(string_agg(to_jsonb(row_data)::text,E'\n' order by to_jsonb(row_data)::text),'')) from %s row_data`, table)
		var fingerprint string
		if err := pool.QueryRow(context.Background(), query).Scan(&fingerprint); err != nil {
			t.Fatalf("fingerprint %s: %v", table, err)
		}
		result[table] = fingerprint
	}
	return result
}

func mediaFingerprint(t *testing.T, root string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	if err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if info.IsDir() {
			result[filepath.ToSlash(relative)+"/"] = fmt.Sprintf("dir:%#o", info.Mode().Perm())
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(contents)
		result[filepath.ToSlash(relative)] = fmt.Sprintf("file:%#o:%x", info.Mode().Perm(), digest)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return result
}
