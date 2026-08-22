package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type job struct {
	ID, AssetID           uuid.UUID
	SourcePath            string
	Attempts, MaxAttempts int
}

func (w *worker) claim(ctx context.Context) (*job, error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var j job
	err = tx.QueryRow(ctx, `update video_jobs set status='leased',locked_by=$1,locked_at=now(),lease_expires_at=now()+interval '30 minutes',attempts=attempts+1,updated_at=now() where id=(select id from video_jobs where (status='queued' or (status='leased' and lease_expires_at<now())) and attempts<max_attempts order by created_at for update skip locked limit 1) returning id,asset_id,attempts,max_attempts`, w.id).Scan(&j.ID, &j.AssetID, &j.Attempts, &j.MaxAttempts)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	err = tx.QueryRow(ctx, `update video_assets va set status='processing',error=null,updated_at=now()
		where va.id=$1 and va.status<>'superseded' and exists(
			select 1 from video_jobs vj where vj.id=$2 and vj.status='leased' and vj.locked_by=$3 and vj.lease_expires_at>now()
		) returning source_path`, j.AssetID, j.ID, w.id).Scan(&j.SourcePath)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &j, nil
}
func (w *worker) heartbeat(ctx context.Context, j job, done <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			tag, err := w.pool.Exec(ctx, `update video_jobs set lease_expires_at=now()+interval '30 minutes',updated_at=now() where id=$1 and status='leased' and locked_by=$2 and lease_expires_at>now()`, j.ID, w.id)
			if err != nil || tag.RowsAffected() != 1 {
				return
			}
		}
	}
}
func (w *worker) complete(ctx context.Context, j job, meta probeResult, qualities []string, manifest string, size int64) error {
	qualityJSON, err := json.Marshal(qualities)
	if err != nil {
		return err
	}
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `update video_assets va set status='ready',manifest_path=$2,qualities=$3,duration_seconds=$4,source_width=$5,source_height=$6,source_fps=$7,size_bytes=$8,error=null,updated_at=now()
		where va.id=$1 and va.status='processing' and exists(
			select 1 from video_jobs vj where vj.id=$9 and vj.asset_id=va.id and vj.status='leased' and vj.locked_by=$10 and vj.lease_expires_at>now()
		)`, j.AssetID, manifest, qualityJSON, meta.Duration, meta.Width, meta.Height, meta.FPS, size, j.ID, w.id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return nil
	}
	tag, err = tx.Exec(ctx, `update video_jobs set status='done',lease_expires_at=null,updated_at=now() where id=$1 and asset_id=$2 and status='leased' and locked_by=$3 and lease_expires_at>now()`, j.ID, j.AssetID, w.id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return nil
	}
	return tx.Commit(ctx)
}
func (w *worker) fail(ctx context.Context, j job, cause error) error {
	message := truncate(cause.Error(), 4000)
	if j.Attempts >= j.MaxAttempts {
		tx, err := w.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		tag, err := tx.Exec(ctx, `update video_assets va set status='failed',error=$2,updated_at=now()
			where va.id=$1 and va.status='processing' and exists(
				select 1 from video_jobs vj where vj.id=$3 and vj.asset_id=va.id and vj.status='leased' and vj.locked_by=$4 and vj.lease_expires_at>now()
			)`, j.AssetID, message, j.ID, w.id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return nil
		}
		tag, err = tx.Exec(ctx, `update video_jobs set status='failed',last_error=$2,lease_expires_at=null,updated_at=now()
			where id=$1 and asset_id=$3 and status='leased' and locked_by=$4 and lease_expires_at>now()`, j.ID, message, j.AssetID, w.id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return nil
		}
		return tx.Commit(ctx)
	}
	backoff := time.Duration(1<<min(j.Attempts, 6)) * time.Second
	tag, err := w.pool.Exec(ctx, `update video_jobs set status='leased',locked_by=null,locked_at=null,lease_expires_at=now()+$2::interval,last_error=$3,updated_at=now()
		where id=$1 and asset_id=$4 and status='leased' and locked_by=$5 and lease_expires_at>now()`, j.ID, fmt.Sprintf("%f seconds", backoff.Seconds()), message, j.AssetID, w.id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return nil
	}
	return nil
}
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
