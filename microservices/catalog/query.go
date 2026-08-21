package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/jackc/pgx/v5"
)

type titleItem struct {
	ID          int     `json:"id"`
	TitleID     int     `json:"title_id"`
	Title       string  `json:"title,omitempty"`
	ContentType string  `json:"content_type"`
	Description string  `json:"description,omitempty"`
	Director    string  `json:"director,omitempty"`
	Year        *int16  `json:"year_released,omitempty"`
	Thumbnail   string  `json:"thumbnail_url"`
	Published   bool    `json:"published,omitempty"`
	AssetID     *string `json:"asset_id"`
}

const assetJoin = `left join lateral (
 select va.id::text asset_id from video_assets va
 where va.status='ready' and ((m.id_movie is not null and va.id_movie=m.id_movie) or
 (s.id_series is not null and va.id_episode in (select e.id_episode from seasons se join episodes e on e.id_season=se.id_season where se.id_series=s.id_series and e.deleted_at is null)))
 order by va.updated_at desc nulls last,va.created_at desc limit 1
) asset on true`

func adminRequest(r *http.Request) bool { return r.Header.Get("X-User-Role") == "admin" }
func visibility(admin bool) string {
	if admin {
		return "true"
	}
	return "t.published=true and asset.asset_id is not null"
}

func (app *application) listTitles(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50, 1, 100)
	offset := queryInt(r, "offset", 0, 0, 100000)
	kind := r.URL.Query().Get("type")
	genre := r.URL.Query().Get("genre")
	q := r.URL.Query().Get("q")
	rows, err := app.pool.Query(r.Context(), `select t.id_title,t.id_title,t.title,t.type::text,coalesce(t.description,''),coalesce(t.director,''),t.year_released,coalesce(t.thumbnail_url,''),t.published,asset.asset_id from titles t left join movies m on m.id_title=t.id_title and m.deleted_at is null left join series s on s.id_title=t.id_title and s.deleted_at is null `+assetJoin+` where t.deleted_at is null and (`+visibility(adminRequest(r))+`) and ($1='' or t.type::text=$1) and ($2='' or exists(select 1 from title_genres tg join genres g on g.id_genre=tg.id_genre where tg.id_title=t.id_title and lower(g.name)=lower($2))) and ($3='' or t.title ilike '%'||$3||'%') order by t.created_at desc limit $4 offset $5`, kind, genre, q, limit, offset)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	items, err := scanTitles(rows)
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, 200, items)
}

func (app *application) getTitle(w http.ResponseWriter, r *http.Request) {
	app.singleTitle(w, r, "t.id_title")
}
func (app *application) singleTitle(w http.ResponseWriter, r *http.Request, column string) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	row := app.pool.QueryRow(r.Context(), `select t.id_title,t.id_title,t.title,t.type::text,coalesce(t.description,''),coalesce(t.director,''),t.year_released,coalesce(t.thumbnail_url,''),t.published,asset.asset_id from titles t left join movies m on m.id_title=t.id_title and m.deleted_at is null left join series s on s.id_title=t.id_title and s.deleted_at is null `+assetJoin+` where t.deleted_at is null and `+column+`=$1 and (`+visibility(adminRequest(r))+`)`, id)
	item, err := scanTitle(row)
	if err == pgx.ErrNoRows {
		jsonx.Error(w, 404, "title not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, 200, item)
}
func (app *application) getMovie(w http.ResponseWriter, r *http.Request) {
	app.singleTitle(w, r, "t.id_title")
}

type episodeResponse struct {
	ID            int     `json:"id"`
	EpisodeNumber int     `json:"episode_number"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Duration      int     `json:"duration"`
	Thumbnail     string  `json:"thumbnail_url"`
	AssetID       *string `json:"asset_id"`
}
type seasonResponse struct {
	ID           int               `json:"id"`
	SeasonNumber int               `json:"season_number"`
	Episodes     []episodeResponse `json:"episodes"`
}
type seriesResponse struct {
	titleItem
	SeriesID        int              `json:"series_id"`
	NumberOfSeasons int              `json:"number_of_seasons"`
	Seasons         []seasonResponse `json:"seasons"`
}

func (app *application) getSeries(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	row := app.pool.QueryRow(r.Context(), `select t.id_title,t.id_title,t.title,t.type::text,coalesce(t.description,''),coalesce(t.director,''),t.year_released,coalesce(t.thumbnail_url,''),t.published,asset.asset_id,s.id_series,coalesce(s.number_of_seasons,0) from titles t join series s on s.id_title=t.id_title and s.deleted_at is null left join movies m on false `+assetJoin+` where t.deleted_at is null and t.id_title=$1 and (`+visibility(adminRequest(r))+`)`, id)
	var out seriesResponse
	err := row.Scan(&out.ID, &out.TitleID, &out.Title, &out.ContentType, &out.Description, &out.Director, &out.Year, &out.Thumbnail, &out.Published, &out.AssetID, &out.SeriesID, &out.NumberOfSeasons)
	if err == pgx.ErrNoRows {
		jsonx.Error(w, 404, "series not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	rows, err := app.pool.Query(r.Context(), `select se.id_season,se.season_number,e.id_episode,e.episode_number,e.title,coalesce(e.description,''),coalesce(e.duration,0),coalesce(e.thumbnail_url,''),va.id::text from seasons se left join episodes e on e.id_season=se.id_season and e.deleted_at is null left join video_assets va on va.id_episode=e.id_episode and va.status='ready' where se.id_series=$1 and se.deleted_at is null order by se.season_number,e.episode_number`, id)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	var current *seasonResponse
	for rows.Next() {
		var sid, sn int
		var eid, en *int
		var title, desc, thumb *string
		var dur *int
		var asset *string
		if err := rows.Scan(&sid, &sn, &eid, &en, &title, &desc, &dur, &thumb, &asset); err != nil {
			serverError(w, err)
			return
		}
		if current == nil || current.ID != sid {
			out.Seasons = append(out.Seasons, seasonResponse{ID: sid, SeasonNumber: sn, Episodes: []episodeResponse{}})
			current = &out.Seasons[len(out.Seasons)-1]
		}
		if eid != nil {
			current.Episodes = append(current.Episodes, episodeResponse{ID: *eid, EpisodeNumber: *en, Title: deref(title), Description: deref(desc), Duration: derefInt(dur), Thumbnail: deref(thumb), AssetID: asset})
		}
	}
	jsonx.Write(w, 200, out)
}

type refItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (app *application) listRefs(w http.ResponseWriter, r *http.Request, query string) {
	rows, err := app.pool.Query(r.Context(), query)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	out := []refItem{}
	for rows.Next() {
		var item refItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			serverError(w, err)
			return
		}
		out = append(out, item)
	}
	jsonx.Write(w, 200, out)
}
func (app *application) listGenres(w http.ResponseWriter, r *http.Request) {
	app.listRefs(w, r, `select id_genre,name from genres where deleted_at is null order by name`)
}
func (app *application) listCategories(w http.ResponseWriter, r *http.Request) {
	app.listRefs(w, r, `select id_category,name from categories where deleted_at is null order by name`)
}
func (app *application) listActors(w http.ResponseWriter, r *http.Request) {
	app.listRefs(w, r, `select id_actor,name from actors where deleted_at is null order by name`)
}

type homeRow struct {
	ID    string      `json:"id"`
	Title string      `json:"title"`
	Items []titleItem `json:"items"`
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	admin := adminRequest(r)
	base := `select t.id_title,t.id_title,t.title,t.type::text,coalesce(t.description,''),coalesce(t.director,''),t.year_released,coalesce(t.thumbnail_url,''),t.published,asset.asset_id from titles t left join movies m on m.id_title=t.id_title and m.deleted_at is null left join series s on s.id_title=t.id_title and s.deleted_at is null ` + assetJoin + ` where t.deleted_at is null and (` + visibility(admin) + `) `
	result := []homeRow{}
	userID := r.Header.Get("X-User-Id")
	if userID != "" {
		rows, err := app.pool.Query(r.Context(), base+` and exists(select 1 from watch_progress wp where wp.user_id=$1 and (wp.id_movie=m.id_movie or wp.id_episode in(select e.id_episode from seasons se join episodes e on e.id_season=se.id_season where se.id_series=s.id_series))) order by t.updated_at desc nulls last limit 20`, userID)
		if err == nil {
			items, _ := scanTitles(rows)
			rows.Close()
			if len(items) > 0 {
				result = append(result, homeRow{ID: "continue", Title: "Continue watching", Items: items})
			}
		}
	}
	for _, spec := range []struct{ id, title, where string }{{"trending", "Trending", " order by t.created_at desc limit 20"}, {"movies", "Movies", " and t.type='Movie' order by t.created_at desc limit 20"}, {"series", "Series", " and t.type='TV Show' order by t.created_at desc limit 20"}} {
		rows, err := app.pool.Query(r.Context(), base+spec.where)
		if err != nil {
			serverError(w, err)
			return
		}
		items, err := scanTitles(rows)
		rows.Close()
		if err != nil {
			serverError(w, err)
			return
		}
		result = append(result, homeRow{ID: spec.id, Title: spec.title, Items: items})
	}
	genres, err := app.pool.Query(r.Context(), `select id_genre,name from genres where deleted_at is null and exists(select 1 from title_genres where id_genre=genres.id_genre) order by name`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer genres.Close()
	for genres.Next() {
		var id int
		var name string
		if err := genres.Scan(&id, &name); err != nil {
			serverError(w, err)
			return
		}
		rows, err := app.pool.Query(r.Context(), base+` and exists(select 1 from title_genres tg where tg.id_title=t.id_title and tg.id_genre=$1) order by t.created_at desc limit 20`, id)
		if err != nil {
			serverError(w, err)
			return
		}
		items, err := scanTitles(rows)
		rows.Close()
		if err != nil {
			serverError(w, err)
			return
		}
		if len(items) > 0 {
			result = append(result, homeRow{ID: "genre-" + strconv.Itoa(id), Title: name, Items: items})
		}
	}
	jsonx.Write(w, 200, result)
}

type scanner interface{ Scan(...any) error }

func scanTitle(s scanner) (titleItem, error) {
	var i titleItem
	err := s.Scan(&i.ID, &i.TitleID, &i.Title, &i.ContentType, &i.Description, &i.Director, &i.Year, &i.Thumbnail, &i.Published, &i.AssetID)
	return i, err
}
func scanTitles(rows pgx.Rows) ([]titleItem, error) {
	out := []titleItem{}
	for rows.Next() {
		item, err := scanTitle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
func queryInt(r *http.Request, key string, fallback, min, max int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func pathInt(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
	v, err := strconv.Atoi(r.PathValue(key))
	if err != nil || v < 1 {
		jsonx.Error(w, 400, "invalid "+key)
		return 0, false
	}
	return v, true
}
func serverError(w http.ResponseWriter, err error) {
	log.Printf("catalog error: %v", err)
	jsonx.Error(w, 500, "internal server error")
}
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		jsonx.Error(w, 400, "invalid JSON body")
		return false
	}
	return true
}
func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
