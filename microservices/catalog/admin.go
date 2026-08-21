package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (app *application) admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-Role") != "admin" {
			jsonx.Error(w, 403, "administrator access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type titleInput struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Director        string `json:"director"`
	Year            int16  `json:"year_released"`
	Duration        int    `json:"duration"`
	NumberOfSeasons int    `json:"number_of_seasons"`
}

func validateTitle(w http.ResponseWriter, in titleInput) bool {
	if strings.TrimSpace(in.Title) == "" {
		jsonx.Error(w, 400, "title is required")
		return false
	}
	return true
}

func (app *application) createMovie(w http.ResponseWriter, r *http.Request) {
	var in titleInput
	if !decodeJSON(w, r, &in) || !validateTitle(w, in) {
		return
	}
	tx, err := app.pool.Begin(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var titleID, movieID int
	err = tx.QueryRow(r.Context(), `insert into titles(type,title,description,director,year_released) values('Movie',$1,$2,$3,$4) returning id_title`, strings.TrimSpace(in.Title), in.Description, nullText(in.Director), nullYear(in.Year)).Scan(&titleID)
	if err == nil {
		err = tx.QueryRow(r.Context(), `insert into movies(id_title,duration) values($1,$2) returning id_movie`, titleID, nullInt(in.Duration)).Scan(&movieID)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, 201, map[string]any{"id": movieID, "title_id": titleID})
}
func (app *application) patchMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var in titleInput
	if !decodeJSON(w, r, &in) || !validateTitle(w, in) {
		return
	}
	tag, err := app.pool.Exec(r.Context(), `update titles t set title=$2,description=$3,director=$4,year_released=$5,updated_at=now() from movies m where m.id_movie=$1 and m.id_title=t.id_title and m.deleted_at is null`, id, strings.TrimSpace(in.Title), in.Description, nullText(in.Director), nullYear(in.Year))
	if err == nil {
		_, err = app.pool.Exec(r.Context(), `update movies set duration=$2,updated_at=now() where id_movie=$1 and deleted_at is null`, id, nullInt(in.Duration))
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, 404, "movie not found")
		return
	}
	w.WriteHeader(204)
}
func (app *application) deleteMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	tag, err := app.pool.Exec(r.Context(), `update titles t set deleted_at=now(),updated_at=now() from movies m where m.id_movie=$1 and m.id_title=t.id_title and m.deleted_at is null`, id)
	if err == nil {
		_, err = app.pool.Exec(r.Context(), `update movies set deleted_at=now(),updated_at=now() where id_movie=$1 and deleted_at is null`, id)
	}
	deleted(w, tag, err)
}

func (app *application) createSeries(w http.ResponseWriter, r *http.Request) {
	var in titleInput
	if !decodeJSON(w, r, &in) || !validateTitle(w, in) {
		return
	}
	if in.NumberOfSeasons < 0 || in.NumberOfSeasons > 100 {
		jsonx.Error(w, 400, "invalid number_of_seasons")
		return
	}
	tx, err := app.pool.Begin(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var titleID, seriesID int
	err = tx.QueryRow(r.Context(), `insert into titles(type,title,description,director,year_released) values('TV Show',$1,$2,$3,$4) returning id_title`, strings.TrimSpace(in.Title), in.Description, nullText(in.Director), nullYear(in.Year)).Scan(&titleID)
	if err == nil {
		err = tx.QueryRow(r.Context(), `insert into series(id_title,number_of_seasons) values($1,$2) returning id_series`, titleID, in.NumberOfSeasons).Scan(&seriesID)
	}
	for n := 1; err == nil && n <= in.NumberOfSeasons; n++ {
		_, err = tx.Exec(r.Context(), `insert into seasons(id_series,season_number,number_of_episodes) values($1,$2,0)`, seriesID, n)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, 201, map[string]any{"id": seriesID, "title_id": titleID})
}
func (app *application) patchSeries(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var in titleInput
	if !decodeJSON(w, r, &in) || !validateTitle(w, in) {
		return
	}
	if in.NumberOfSeasons < 0 || in.NumberOfSeasons > 100 {
		jsonx.Error(w, 400, "invalid number_of_seasons")
		return
	}
	tx, err := app.pool.Begin(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `update titles t set title=$2,description=$3,director=$4,year_released=$5,updated_at=now() from series s where s.id_series=$1 and s.id_title=t.id_title and s.deleted_at is null`, id, strings.TrimSpace(in.Title), in.Description, nullText(in.Director), nullYear(in.Year))
	if err == nil {
		_, err = tx.Exec(r.Context(), `update series set number_of_seasons=$2,updated_at=now() where id_series=$1`, id, in.NumberOfSeasons)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `insert into seasons(id_series,season_number,number_of_episodes) select $1,n,0 from generate_series(1,$2) n on conflict(id_series,season_number) do update set deleted_at=null`, id, in.NumberOfSeasons)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `update seasons set deleted_at=now(),updated_at=now() where id_series=$1 and season_number>$2 and deleted_at is null`, id, in.NumberOfSeasons)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, 404, "series not found")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (app *application) deleteSeries(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	tag, err := app.pool.Exec(r.Context(), `update titles t set deleted_at=now(),updated_at=now() from series s where s.id_series=$1 and s.id_title=t.id_title and s.deleted_at is null`, id)
	if err == nil {
		_, err = app.pool.Exec(r.Context(), `update series set deleted_at=now(),updated_at=now() where id_series=$1 and deleted_at is null`, id)
	}
	deleted(w, tag, err)
}

type seasonInput struct {
	SeasonNumber     int `json:"season_number"`
	NumberOfEpisodes int `json:"number_of_episodes"`
}

func (app *application) createSeason(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var in seasonInput
	if !decodeJSON(w, r, &in) || in.SeasonNumber < 1 {
		return
	}
	var seasonID int
	err := app.pool.QueryRow(r.Context(), `insert into seasons(id_series,season_number,number_of_episodes) values($1,$2,$3) on conflict(id_series,season_number) do update set number_of_episodes=excluded.number_of_episodes,deleted_at=null,updated_at=now() returning id_season`, id, in.SeasonNumber, in.NumberOfEpisodes).Scan(&seasonID)
	if err != nil {
		serverError(w, err)
		return
	}
	_, _ = app.pool.Exec(r.Context(), `update series set number_of_seasons=(select count(*) from seasons where id_series=$1 and deleted_at is null),updated_at=now() where id_series=$1`, id)
	jsonx.Write(w, 201, map[string]int{"id": seasonID})
}

func (app *application) patchSeason(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var in seasonInput
	if !decodeJSON(w, r, &in) || in.SeasonNumber < 1 {
		jsonx.Error(w, 400, "season_number is required")
		return
	}
	tag, err := app.pool.Exec(r.Context(), `update seasons set season_number=$2,number_of_episodes=$3,updated_at=now() where id_season=$1 and deleted_at is null`, id, in.SeasonNumber, in.NumberOfEpisodes)
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, 404, "season not found")
		return
	}
	w.WriteHeader(204)
}

func (app *application) deleteSeason(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	tx, err := app.pool.Begin(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var seriesID int
	err = tx.QueryRow(r.Context(), `update seasons set deleted_at=now(),updated_at=now() where id_season=$1 and deleted_at is null returning id_series`, id).Scan(&seriesID)
	if err == pgx.ErrNoRows {
		jsonx.Error(w, 404, "season not found")
		return
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `update series set number_of_seasons=(select count(*) from seasons where id_series=$1 and deleted_at is null),updated_at=now() where id_series=$1`, seriesID)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

type episodeInput struct {
	EpisodeNumber int    `json:"episode_number"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Duration      int    `json:"duration"`
}

func (app *application) createEpisode(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var in episodeInput
	if !decodeJSON(w, r, &in) || in.EpisodeNumber < 1 || strings.TrimSpace(in.Title) == "" {
		jsonx.Error(w, 400, "episode_number and title are required")
		return
	}
	var episodeID int
	err := app.pool.QueryRow(r.Context(), `insert into episodes(id_season,episode_number,title,description,duration) values($1,$2,$3,$4,$5) on conflict(id_season,episode_number) do update set title=excluded.title,description=excluded.description,duration=excluded.duration,deleted_at=null,updated_at=now() returning id_episode`, id, in.EpisodeNumber, strings.TrimSpace(in.Title), in.Description, nullInt(in.Duration)).Scan(&episodeID)
	if err != nil {
		serverError(w, err)
		return
	}
	_, _ = app.pool.Exec(r.Context(), `update seasons set number_of_episodes=(select count(*) from episodes where id_season=$1 and deleted_at is null),updated_at=now() where id_season=$1`, id)
	jsonx.Write(w, 201, map[string]int{"id": episodeID})
}

func (app *application) patchEpisode(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var in episodeInput
	if !decodeJSON(w, r, &in) || in.EpisodeNumber < 1 || strings.TrimSpace(in.Title) == "" {
		jsonx.Error(w, 400, "episode_number and title are required")
		return
	}
	tag, err := app.pool.Exec(r.Context(), `update episodes set episode_number=$2,title=$3,description=$4,duration=$5,updated_at=now() where id_episode=$1 and deleted_at is null`, id, in.EpisodeNumber, strings.TrimSpace(in.Title), in.Description, nullInt(in.Duration))
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, 404, "episode not found")
		return
	}
	w.WriteHeader(204)
}

func (app *application) deleteEpisode(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	tx, err := app.pool.Begin(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var seasonID int
	err = tx.QueryRow(r.Context(), `update episodes set deleted_at=now(),updated_at=now() where id_episode=$1 and deleted_at is null returning id_season`, id).Scan(&seasonID)
	if err == pgx.ErrNoRows {
		jsonx.Error(w, 404, "episode not found")
		return
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `update seasons set number_of_episodes=(select count(*) from episodes where id_season=$1 and deleted_at is null),updated_at=now() where id_season=$1`, seasonID)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(204)
}

type genreInput struct {
	Name string `json:"name"`
}

func (app *application) createGenre(w http.ResponseWriter, r *http.Request) {
	var in genreInput
	if !decodeJSON(w, r, &in) || strings.TrimSpace(in.Name) == "" {
		jsonx.Error(w, 400, "name is required")
		return
	}
	var id int
	err := app.pool.QueryRow(r.Context(), `insert into genres(name) values($1) on conflict(name) do update set deleted_at=null,updated_at=now() returning id_genre`, strings.TrimSpace(in.Name)).Scan(&id)
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, 201, map[string]int{"id": id})
}
func (app *application) patchGenre(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var in genreInput
	if !decodeJSON(w, r, &in) || strings.TrimSpace(in.Name) == "" {
		jsonx.Error(w, 400, "name is required")
		return
	}
	tag, err := app.pool.Exec(r.Context(), `update genres set name=$2,updated_at=now() where id_genre=$1 and deleted_at is null`, id, strings.TrimSpace(in.Name))
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, 404, "genre not found")
		return
	}
	w.WriteHeader(204)
}
func (app *application) deleteGenre(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	tag, err := app.pool.Exec(r.Context(), `update genres set deleted_at=now(),updated_at=now() where id_genre=$1 and deleted_at is null`, id)
	deleted(w, tag, err)
}

func (app *application) uploadMovieVideo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if ok {
		app.uploadVideo(w, r, "movie", id)
	}
}
func (app *application) uploadEpisodeVideo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if ok {
		app.uploadVideo(w, r, "episode", id)
	}
}
func (app *application) uploadVideo(w http.ResponseWriter, r *http.Request, kind string, target int) {
	r.Body = http.MaxBytesReader(w, r.Body, app.maxUpload)
	part, ext, mime, err := videoPart(r)
	if err != nil {
		jsonx.Error(w, 400, err.Error())
		return
	}
	defer part.Close()
	if !validVideo(ext, mime) {
		jsonx.Error(w, 415, "file content is not an allowed video")
		return
	}
	tx, err := app.pool.Begin(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	assetID := uuid.New()
	var previous uuid.UUID
	if kind == "movie" {
		err = tx.QueryRow(r.Context(), `select id from video_assets where id_movie=$1 and status in('pending','processing','ready','failed') order by created_at desc limit 1 for update`, target).Scan(&previous)
	} else {
		err = tx.QueryRow(r.Context(), `select id from video_assets where id_episode=$1 and status in('pending','processing','ready','failed') order by created_at desc limit 1 for update`, target).Scan(&previous)
	}
	if err != nil && err != pgx.ErrNoRows {
		serverError(w, err)
		return
	}
	key := filepath.ToSlash(filepath.Join("sources", assetID.String(), "source"+ext))
	if err = app.store.Put(key, io.MultiReader(bytes.NewReader(mime.head), part)); err != nil {
		writeUploadError(w, err)
		return
	}
	if previous != uuid.Nil {
		_, err = tx.Exec(r.Context(), `update video_assets set status='superseded',superseded_at=now(),error='superseded by re-upload',updated_at=now() where id=$1`, previous)
		if err == nil {
			_, err = tx.Exec(r.Context(), `update video_jobs set status='failed',lease_expires_at=null,last_error='superseded by re-upload',updated_at=now() where asset_id=$1 and status in('queued','leased')`, previous)
		}
	}
	if err == nil && kind == "movie" {
		_, err = tx.Exec(r.Context(), `insert into video_assets(id,kind,id_movie,status,source_path) values($1,'movie',$2,'pending',$3)`, assetID, target, key)
	} else if err == nil {
		_, err = tx.Exec(r.Context(), `insert into video_assets(id,kind,id_episode,status,source_path) values($1,'episode',$2,'pending',$3)`, assetID, target, key)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), `insert into video_jobs(asset_id,status) values($1,'queued')`, assetID)
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, 202, map[string]any{"asset_id": assetID, "status": "pending"})
}

type sniffed struct {
	kind string
	head []byte
}

func videoPart(r *http.Request) (*multipart.Part, string, sniffed, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, "", sniffed{}, fmt.Errorf("multipart body required")
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", sniffed{}, err
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}
		ext := strings.ToLower(filepath.Ext(part.FileName()))
		head := make([]byte, 512)
		n, readErr := io.ReadFull(part, head)
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return nil, "", sniffed{}, readErr
		}
		head = head[:n]
		return part, ext, sniffed{kind: http.DetectContentType(head), head: head}, nil
	}
	return nil, "", sniffed{}, fmt.Errorf("file field is required")
}
func validVideo(ext string, m sniffed) bool {
	allowedExt := map[string]bool{".mp4": true, ".mov": true, ".mkv": true, ".webm": true}
	allowedMIME := map[string]bool{"video/mp4": true, "video/quicktime": true, "video/webm": true, "video/x-matroska": true}
	return allowedExt[ext] && allowedMIME[strings.Split(m.kind, ";")[0]]
}

func (app *application) uploadThumbnail(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	part, ext, mime, err := videoPart(r)
	if err != nil {
		jsonx.Error(w, 400, err.Error())
		return
	}
	defer part.Close()
	allowedExt := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	allowedMime := map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}
	if !allowedExt[ext] || !allowedMime[strings.Split(mime.kind, ";")[0]] {
		jsonx.Error(w, 415, "file content is not an allowed image")
		return
	}
	key := filepath.ToSlash(filepath.Join("thumbnails", fmt.Sprintf("title-%d-%s%s", id, uuid.NewString(), ext)))
	if err = app.store.Put(key, io.MultiReader(bytes.NewReader(mime.head), part)); err != nil {
		writeUploadError(w, err)
		return
	}
	tag, err := app.pool.Exec(r.Context(), `update titles set thumbnail_url=$2,updated_at=now() where id_title=$1 and deleted_at is null`, id, "/api/v1/stream/"+key)
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, 404, "title not found")
		return
	}
	jsonx.Write(w, 200, map[string]string{"thumbnail_url": "/api/v1/stream/" + key})
}
func (app *application) publishTitle(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(w, r, "id")
	if !ok {
		return
	}
	var in struct {
		Published bool `json:"published"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	tag, err := app.pool.Exec(r.Context(), `update titles set published=$2,updated_at=now() where id_title=$1 and deleted_at is null`, id, in.Published)
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, 404, "title not found")
		return
	}
	jsonx.Write(w, 200, map[string]bool{"published": in.Published})
}
func (app *application) assetStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonx.Error(w, 400, "invalid asset id")
		return
	}
	var out struct {
		Status    string   `json:"status"`
		Qualities []string `json:"qualities"`
		Error     *string  `json:"error"`
		Duration  *float64 `json:"duration"`
		Width     *int     `json:"source_width"`
		Height    *int     `json:"source_height"`
	}
	var qualities []byte
	err = app.pool.QueryRow(r.Context(), `select status::text,qualities,error,duration_seconds::float8,source_width,source_height from video_assets where id=$1`, id).Scan(&out.Status, &qualities, &out.Error, &out.Duration, &out.Width, &out.Height)
	if err == pgx.ErrNoRows {
		jsonx.Error(w, 404, "asset not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	_ = json.Unmarshal(qualities, &out.Qualities)
	jsonx.Write(w, 200, out)
}

func deleted(w http.ResponseWriter, tag pgconn.CommandTag, err error) {
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, 404, "not found")
		return
	}
	w.WriteHeader(204)
}
func nullText(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}
func nullInt(v int) any {
	if v <= 0 {
		return nil
	}
	return v
}
func nullYear(v int16) any {
	if v <= 0 {
		return nil
	}
	return v
}
func writeUploadError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		jsonx.Error(w, http.StatusRequestEntityTooLarge, "upload exceeds size limit")
		return
	}
	serverError(w, err)
}
