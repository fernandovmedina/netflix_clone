package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
