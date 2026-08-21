package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	shareddb "github.com/fernandovmedina/netflix-clone/microservices/shared/database"
	"github.com/fernandovmedina/netflix-clone/microservices/shared/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type worker struct {
	pool     *pgxpool.Pool
	store    storage.Store
	root, id string
	lease    time.Duration
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := shareddb.Open(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	root := env("MEDIA_ROOT", "/media")
	store, err := storage.NewLocal(root)
	if err != nil {
		log.Fatal(err)
	}
	host, _ := os.Hostname()
	concurrency, err := strconv.Atoi(env("WORKER_CONCURRENCY", "1"))
	if err != nil || concurrency < 1 || concurrency > 16 {
		log.Fatal("invalid WORKER_CONCURRENCY")
	}
	w := &worker{pool: pool, store: store, root: root, id: host, lease: 30 * time.Minute}
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(slot int) { defer wg.Done(); w.loop(ctx, slot) }(i)
	}
	log.Printf("[%s] worker started with concurrency %d", host, concurrency)
	wg.Wait()
}
func (w *worker) loop(ctx context.Context, slot int) {
	for {
		if ctx.Err() != nil {
			return
		}
		job, err := w.claim(ctx)
		if err != nil {
			log.Printf("[%s:%d] claim: %v", w.id, slot, err)
			sleep(ctx, 2*time.Second)
			continue
		}
		if job == nil {
			sleep(ctx, time.Second)
			continue
		}
		log.Printf("[%s:%d] processing job %s asset %s attempt %d", w.id, slot, job.ID, job.AssetID, job.Attempts)
		if err = w.process(ctx, *job); err != nil {
			log.Printf("[%s:%d] job %s failed: %v", w.id, slot, job.ID, err)
			if markErr := w.fail(ctx, *job, err); markErr != nil {
				log.Printf("[%s:%d] marking failure: %v", w.id, slot, markErr)
			}
		}
	}
}
func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
