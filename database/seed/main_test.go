package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
