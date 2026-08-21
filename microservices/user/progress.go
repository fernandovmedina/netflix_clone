package main

import (
	"net/http"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/jackc/pgx/v5"
)

type progressResponse struct {
	Kind        string    `json:"kind"`
	ContentID   int       `json:"content_id"`
	CurrentTime int       `json:"current_time_seconds"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func progressTarget(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	kind := r.PathValue("kind")
	if kind != "movie" && kind != "episode" {
		jsonx.Error(w, http.StatusBadRequest, "kind must be movie or episode")
		return "", 0, false
	}
	id, ok := pathPositiveInt(w, r, "id")
	return kind, id, ok
}

func (app *application) getProgress(w http.ResponseWriter, r *http.Request) {
	kind, id, ok := progressTarget(w, r)
	if !ok {
		return
	}
	column := "id_movie"
	if kind == "episode" {
		column = "id_episode"
	}
	var out progressResponse
	out.Kind, out.ContentID = kind, id
	err := app.pool.QueryRow(r.Context(), `select current_time_seconds,updated_at from watch_progress where user_id=$1 and `+column+`=$2`, userID(r), id).Scan(&out.CurrentTime, &out.UpdatedAt)
	if err == pgx.ErrNoRows {
		jsonx.Error(w, http.StatusNotFound, "progress not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, http.StatusOK, out)
}

func (app *application) putProgress(w http.ResponseWriter, r *http.Request) {
	kind, id, ok := progressTarget(w, r)
	if !ok {
		return
	}
	var in struct {
		CurrentTime int `json:"current_time_seconds"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.CurrentTime < 0 {
		jsonx.Error(w, http.StatusBadRequest, "current_time_seconds must not be negative")
		return
	}
	var out progressResponse
	out.Kind, out.ContentID = kind, id
	var err error
	if kind == "movie" {
		err = app.pool.QueryRow(r.Context(), `insert into watch_progress(user_id,id_movie,current_time_seconds) values($1,$2,$3)
			on conflict(user_id,id_movie) where id_movie is not null do update set current_time_seconds=excluded.current_time_seconds,updated_at=now()
			returning current_time_seconds,updated_at`, userID(r), id, in.CurrentTime).Scan(&out.CurrentTime, &out.UpdatedAt)
	} else {
		err = app.pool.QueryRow(r.Context(), `insert into watch_progress(user_id,id_episode,current_time_seconds) values($1,$2,$3)
			on conflict(user_id,id_episode) where id_episode is not null do update set current_time_seconds=excluded.current_time_seconds,updated_at=now()
			returning current_time_seconds,updated_at`, userID(r), id, in.CurrentTime).Scan(&out.CurrentTime, &out.UpdatedAt)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, http.StatusOK, out)
}

type continueItem struct {
	Kind        string    `json:"kind"`
	ContentID   int       `json:"content_id"`
	TitleID     int       `json:"title_id"`
	Title       string    `json:"title"`
	Thumbnail   string    `json:"thumbnail_url"`
	CurrentTime int       `json:"current_time_seconds"`
	Duration    int       `json:"duration_seconds"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (app *application) continueWatching(w http.ResponseWriter, r *http.Request) {
	rows, err := app.pool.Query(r.Context(), `
		select kind,content_id,title_id,title,thumbnail,current_time_seconds,duration_seconds,updated_at from (
			select 'movie' kind,m.id_movie content_id,t.id_title title_id,t.title,coalesce(t.thumbnail_url,'') thumbnail,
				wp.current_time_seconds,coalesce(m.duration,0)*60 duration_seconds,wp.updated_at
			from watch_progress wp join movies m on m.id_movie=wp.id_movie join titles t on t.id_title=m.id_title
			where wp.user_id=$1 and m.deleted_at is null and t.deleted_at is null and coalesce(m.duration,0)>0
				and wp.current_time_seconds*100 <= coalesce(m.duration,0)*60*95
			union all
			select 'episode',e.id_episode,t.id_title,t.title||' — '||e.title,coalesce(e.thumbnail_url,t.thumbnail_url,''),
				wp.current_time_seconds,coalesce(e.duration,0)*60,wp.updated_at
			from watch_progress wp join episodes e on e.id_episode=wp.id_episode join seasons se on se.id_season=e.id_season
			join series s on s.id_series=se.id_series join titles t on t.id_title=s.id_title
			where wp.user_id=$1 and e.deleted_at is null and se.deleted_at is null and s.deleted_at is null and t.deleted_at is null
				and coalesce(e.duration,0)>0 and wp.current_time_seconds*100 <= coalesce(e.duration,0)*60*95
		) items order by updated_at desc limit 20`, userID(r))
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	items := []continueItem{}
	for rows.Next() {
		var item continueItem
		if err = rows.Scan(&item.Kind, &item.ContentID, &item.TitleID, &item.Title, &item.Thumbnail, &item.CurrentTime, &item.Duration, &item.UpdatedAt); err != nil {
			serverError(w, err)
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, http.StatusOK, items)
}
