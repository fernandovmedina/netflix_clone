package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/storage"
	"github.com/google/uuid"
)

func TestSeedVideoEndToEnd(t *testing.T) {
	if os.Getenv("PHASE2_E2E") != "1" {
		t.Skip("PHASE2_E2E not set")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	pool := testPool(t)
	source := os.Getenv("PHASE2_SEED_VIDEO")
	if source == "" {
		source = filepath.Join("..", "..", "seed", "video", "video.mp4")
	}
	input, err := os.Open(source)
	if os.IsNotExist(err) {
		// Seed video is not committed; point PHASE2_SEED_VIDEO at a local clip.
		t.Skipf("seed video %s not present", source)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	root := t.TempDir()
	store, err := storage.NewLocal(root)
	if err != nil {
		t.Fatal(err)
	}
	assetID, jobID := uuid.New(), uuid.New()
	key := filepath.ToSlash(filepath.Join("sources", assetID.String(), "source.mp4"))
	if err = store.Put(key, input); err != nil {
		t.Fatal(err)
	}
	var title, movie int
	if err = pool.QueryRow(context.Background(), `insert into titles(type,title,published,created_at) values('Movie',$1,true,now()-interval '2 days') returning id_title`, "e2e-"+assetID.String()).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(context.Background(), `insert into movies(id_title) values($1) returning id_movie`, title).Scan(&movie); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `insert into video_assets(id,kind,id_movie,status,source_path,created_at) values($1,'movie',$2,'pending',$3,now()-interval '2 days')`, assetID, movie, key); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `insert into video_jobs(id,asset_id,status,created_at) values($1,$2,'queued',now()-interval '2 days')`, jobID, assetID); err != nil {
		t.Fatal(err)
	}
	w := &worker{pool: pool, store: store, root: root, id: "e2e", lease: 30}
	claimed, err := w.claim(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil || claimed.ID != jobID {
		t.Fatalf("claimed %#v want %s", claimed, jobID)
	}
	if err = w.process(context.Background(), *claimed); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "hls", assetID.String(), "master.m3u8")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, quality := range []string{"144p", "240p", "360p", "480p", "720p", "1080p"} {
		if !strings.Contains(string(manifest), quality+"/playlist.m3u8") {
			t.Errorf("manifest missing %s:\n%s", quality, manifest)
		}
	}
	if strings.Contains(string(manifest), "1440p") || !strings.Contains(string(manifest), "mp4a.40.2") {
		t.Fatalf("unexpected ladder or missing audio codec:\n%s", manifest)
	}
	segment := filepath.Join(root, "hls", assetID.String(), "1080p", "seg_00000.ts")
	f, err := os.Open(segment)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	n, err := io.Copy(io.Discard, io.LimitReader(f, 188))
	if err != nil || n == 0 {
		t.Fatalf("segment unreadable: bytes=%d err=%v", n, err)
	}
}

func TestSyntheticSourcesEndToEnd(t *testing.T) {
	dir := os.Getenv("PHASE2_SYNTHETIC_DIR")
	if dir == "" {
		t.Skip("PHASE2_SYNTHETIC_DIR not set")
	}
	pool := testPool(t)
	expected := map[int][]string{360: {"144p", "240p", "360p"}, 720: {"144p", "240p", "360p", "480p", "720p"}, 1080: {"144p", "240p", "360p", "480p", "720p", "1080p"}, 1440: {"144p", "240p", "360p", "480p", "720p", "1080p", "1440p"}}
	for height, want := range expected {
		t.Run(fmt.Sprintf("%dp", height), func(t *testing.T) {
			path := filepath.Join(dir, fmt.Sprintf("source-%dp.mp4", height))
			input, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			root := t.TempDir()
			store, err := storage.NewLocal(root)
			if err != nil {
				t.Fatal(err)
			}
			assetID, jobID := uuid.New(), uuid.New()
			key := filepath.ToSlash(filepath.Join("sources", assetID.String(), "source.mp4"))
			if err = store.Put(key, input); err != nil {
				t.Fatal(err)
			}
			var title, movie int
			if err = pool.QueryRow(context.Background(), `insert into titles(type,title,published,created_at) values('Movie',$1,true,now()-interval '3 days') returning id_title`, "synthetic-"+assetID.String()).Scan(&title); err != nil {
				t.Fatal(err)
			}
			if err = pool.QueryRow(context.Background(), `insert into movies(id_title) values($1) returning id_movie`, title).Scan(&movie); err != nil {
				t.Fatal(err)
			}
			if _, err = pool.Exec(context.Background(), `insert into video_assets(id,kind,id_movie,status,source_path,created_at) values($1,'movie',$2,'pending',$3,now()-interval '3 days')`, assetID, movie, key); err != nil {
				t.Fatal(err)
			}
			if _, err = pool.Exec(context.Background(), `insert into video_jobs(id,asset_id,status,created_at) values($1,$2,'queued',now()-interval '3 days')`, jobID, assetID); err != nil {
				t.Fatal(err)
			}
			w := &worker{pool: pool, store: store, root: root, id: "synthetic"}
			claimed, err := w.claim(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if claimed == nil || claimed.ID != jobID {
				t.Fatalf("claimed %#v want %s", claimed, jobID)
			}
			if err = w.process(context.Background(), *claimed); err != nil {
				t.Fatal(err)
			}
			var raw []byte
			if err = pool.QueryRow(context.Background(), `select qualities from video_assets where id=$1 and status='ready'`, assetID).Scan(&raw); err != nil {
				t.Fatal(err)
			}
			var got []string
			if err = json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("qualities=%v want %v", got, want)
			}
			master, err := os.ReadFile(filepath.Join(root, "hls", assetID.String(), "master.m3u8"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(master), "mp4a.40.2") {
				t.Fatalf("audio codec absent:\n%s", master)
			}
			if _, err = os.Stat(filepath.Join(root, "hls", assetID.String(), want[len(want)-1], "seg_00001.ts")); err != nil {
				t.Fatalf("60-second source did not produce seekable multiple segments: %v", err)
			}
		})
	}
}
