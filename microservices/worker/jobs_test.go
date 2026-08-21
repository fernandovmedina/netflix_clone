package main

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("PHASE2_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PHASE2_TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
func fixtureJob(t *testing.T, pool *pgxpool.Pool, status string, expired bool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	suffix := uuid.NewString()
	var title, movie int
	err := pool.QueryRow(context.Background(), `insert into titles(type,title,published) values('Movie',$1,false) returning id_title`, "claim-"+suffix).Scan(&title)
	if err != nil {
		t.Fatal(err)
	}
	err = pool.QueryRow(context.Background(), `insert into movies(id_title) values($1) returning id_movie`, title).Scan(&movie)
	if err != nil {
		t.Fatal(err)
	}
	asset := uuid.New()
	_, err = pool.Exec(context.Background(), `insert into video_assets(id,kind,id_movie,status,source_path) values($1,'movie',$2,'pending','sources/test/source.mp4')`, asset, movie)
	if err != nil {
		t.Fatal(err)
	}
	jobID := uuid.New()
	if status == "queued" {
		_, err = pool.Exec(context.Background(), `insert into video_jobs(id,asset_id,status) values($1,$2,'queued')`, jobID, asset)
	} else if expired {
		_, err = pool.Exec(context.Background(), `insert into video_jobs(id,asset_id,status,locked_by,lease_expires_at) values($1,$2,'leased','dead',now()-interval '1 minute')`, jobID, asset)
	} else {
		_, err = pool.Exec(context.Background(), `insert into video_jobs(id,asset_id,status,locked_by,lease_expires_at) values($1,$2,'leased','live',now()+interval '1 hour')`, jobID, asset)
	}
	if err != nil {
		t.Fatal(err)
	}
	return jobID, asset
}
func TestConcurrentClaimersClaimEachJobOnce(t *testing.T) {
	pool := testPool(t)
	const n = 12
	expected := map[uuid.UUID]bool{}
	for i := 0; i < n; i++ {
		id, _ := fixtureJob(t, pool, "queued", false)
		expected[id] = true
	}
	claimed := make(chan uuid.UUID, n)
	var wg sync.WaitGroup
	for i := 0; i < n*2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := &worker{pool: pool, id: uuid.NewString()}
			j, err := w.claim(context.Background())
			if err != nil {
				t.Errorf("claim: %v", err)
				return
			}
			if j != nil {
				claimed <- j.ID
			}
		}(i)
	}
	wg.Wait()
	close(claimed)
	seen := map[uuid.UUID]bool{}
	for id := range claimed {
		if seen[id] {
			t.Fatalf("job %s claimed twice", id)
		}
		seen[id] = true
	}
	if len(seen) != n {
		t.Fatalf("claimed %d jobs, want %d", len(seen), n)
	}
	for id := range expected {
		if !seen[id] {
			t.Errorf("job %s was not claimed", id)
		}
	}
}
func TestLeaseReclaim(t *testing.T) {
	pool := testPool(t)
	expired, _ := fixtureJob(t, pool, "leased", true)
	live, _ := fixtureJob(t, pool, "leased", false)
	w := &worker{pool: pool, id: "reclaimer"}
	j, err := w.claim(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if j == nil || j.ID != expired {
		t.Fatalf("claimed %#v want expired %s", j, expired)
	}
	var status string
	if err = pool.QueryRow(context.Background(), `select status::text from video_jobs where id=$1`, live).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "leased" {
		t.Fatalf("live lease changed to %s", status)
	}
}
