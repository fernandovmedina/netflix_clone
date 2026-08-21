package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	shareddb "github.com/fernandovmedina/netflix-clone/microservices/shared/database"
	"github.com/fernandovmedina/netflix-clone/microservices/shared/storage"
	"github.com/jackc/pgx/v5"
)

type manifest struct {
	Movies []movie  `json:"movies"`
	Series []series `json:"series"`
}
type baseTitle struct {
	Name        string   `json:"name"`
	Year        int      `json:"year_released"`
	Description string   `json:"description"`
	Genres      []string `json:"genres"`
	Cast        []string `json:"cast"`
	Director    string   `json:"director"`
	Thumbnail   string   `json:"thumbnail_url"`
}
type movie struct {
	baseTitle
	Duration int `json:"duration"`
}
type series struct {
	baseTitle
	NumberOfSeasons int      `json:"number_of_seasons"`
	Seasons         []season `json:"seasons"`
}
type season struct {
	Number           int       `json:"season_number"`
	NumberOfEpisodes int       `json:"number_of_episodes"`
	Episodes         []episode `json:"episodes"`
}
type episode struct {
	Number      int    `json:"episode_number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Duration    int    `json:"duration"`
}

func main() {
	ctx := context.Background()
	pool, err := shareddb.Open(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	mediaRoot := os.Getenv("MEDIA_ROOT")
	if mediaRoot == "" {
		mediaRoot = "/media"
	}
	store, err := storage.NewLocal(mediaRoot)
	if err != nil {
		log.Fatal(err)
	}
	seedRoot := os.Getenv("SEED_ROOT")
	if seedRoot == "" {
		seedRoot = "seed"
	}
	if err := importAll(ctx, pool, store, seedRoot); err != nil {
		log.Fatal(err)
	}
}

type dbPool interface {
	Begin(context.Context) (pgx.Tx, error)
}

func importAll(ctx context.Context, pool dbPool, store storage.Store, root string) error {
	movies, err := readManifest(filepath.Join(root, "movies", "seed.json"))
	if err != nil {
		return err
	}
	seriesData, err := readManifest(filepath.Join(root, "series", "seed.json"))
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "select pg_advisory_xact_lock($1)", int64(0x4e465853454544)); err != nil {
		return err
	}
	for _, item := range movies.Movies {
		if err := importMovie(ctx, tx, store, root, item); err != nil {
			return fmt.Errorf("movie %q: %w", item.Name, err)
		}
	}
	for _, item := range seriesData.Series {
		if err := importSeries(ctx, tx, store, root, item); err != nil {
			return fmt.Errorf("series %q: %w", item.Name, err)
		}
	}
	return tx.Commit(ctx)
}

func readManifest(path string) (manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return manifest{}, err
	}
	defer f.Close()
	var result manifest
	err = json.NewDecoder(f).Decode(&result)
	return result, err
}

func artwork(store storage.Store, root, source, prefix string) (string, error) {
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(source)))
	if err != nil {
		return "", err
	}
	defer f.Close()
	key := filepath.ToSlash(filepath.Join("thumbnails", prefix+"_"+filepath.Base(source)))
	if err := store.Put(key, f); err != nil {
		return "", err
	}
	return "/api/v1/stream/" + key, nil
}

func upsertTitle(ctx context.Context, tx pgx.Tx, store storage.Store, root string, b baseTitle, titleType, prefix string) (int, error) {
	thumb, err := artwork(store, root, b.Thumbnail, prefix)
	if err != nil {
		return 0, err
	}
	var id int
	err = tx.QueryRow(ctx, `select id_title from titles where lower(title)=lower($1) and type=$2 and deleted_at is null order by id_title limit 1`, b.Name, titleType).Scan(&id)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `insert into titles(type,title,description,director,year_released,thumbnail_url,published) values($1,$2,$3,$4,$5,$6,true) returning id_title`, titleType, b.Name, b.Description, nullString(b.Director), b.Year, thumb).Scan(&id)
	} else if err == nil {
		_, err = tx.Exec(ctx, `update titles set description=$2,director=$3,year_released=$4,thumbnail_url=$5,published=true,updated_at=now() where id_title=$1`, id, b.Description, nullString(b.Director), b.Year, thumb)
	}
	if err != nil {
		return 0, err
	}
	category := "Movies"
	if titleType == "TV Show" {
		category = "Series"
	}
	if err := linkNames(ctx, tx, id, "categories", "id_category", "title_categories", category); err != nil {
		return 0, err
	}
	for _, name := range b.Genres {
		if err := linkNames(ctx, tx, id, "genres", "id_genre", "title_genres", name); err != nil {
			return 0, err
		}
	}
	for _, name := range b.Cast {
		if err := linkNames(ctx, tx, id, "actors", "id_actor", "title_actors", name); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func linkNames(ctx context.Context, tx pgx.Tx, titleID int, table, idColumn, join, name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	query := fmt.Sprintf(`insert into %s(name) values($1) on conflict(name) do update set deleted_at=null returning %s`, table, idColumn)
	var id int
	if err := tx.QueryRow(ctx, query, name).Scan(&id); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, fmt.Sprintf(`insert into %s(id_title,%s) values($1,$2) on conflict do nothing`, join, idColumn), titleID, id)
	return err
}

func importMovie(ctx context.Context, tx pgx.Tx, store storage.Store, root string, item movie) error {
	titleID, err := upsertTitle(ctx, tx, store, root, item.baseTitle, "Movie", "movie_"+slug(item.Name))
	if err != nil {
		return err
	}
	var movieID int
	err = tx.QueryRow(ctx, `insert into movies(id_title,duration) values($1,$2) on conflict(id_title) do update set duration=excluded.duration,updated_at=now() returning id_movie`, titleID, item.Duration).Scan(&movieID)
	if err != nil {
		return err
	}
	return ensureAsset(ctx, tx, store, root, "movie", movieID)
}

func importSeries(ctx context.Context, tx pgx.Tx, store storage.Store, root string, item series) error {
	titleID, err := upsertTitle(ctx, tx, store, root, item.baseTitle, "TV Show", "series_"+slug(item.Name))
	if err != nil {
		return err
	}
	var seriesID int
	err = tx.QueryRow(ctx, `insert into series(id_title,number_of_seasons) values($1,$2) on conflict(id_title) do update set number_of_seasons=excluded.number_of_seasons,updated_at=now() returning id_series`, titleID, item.NumberOfSeasons).Scan(&seriesID)
	if err != nil {
		return err
	}
	for _, s := range item.Seasons {
		var seasonID int
		err = tx.QueryRow(ctx, `insert into seasons(id_series,season_number,number_of_episodes) values($1,$2,$3) on conflict(id_series,season_number) do update set number_of_episodes=excluded.number_of_episodes,updated_at=now() returning id_season`, seriesID, s.Number, s.NumberOfEpisodes).Scan(&seasonID)
		if err != nil {
			return err
		}
		for _, ep := range s.Episodes {
			var episodeID int
			err = tx.QueryRow(ctx, `insert into episodes(id_season,episode_number,title,description,duration) values($1,$2,$3,$4,$5) on conflict(id_season,episode_number) do update set title=excluded.title,description=excluded.description,duration=excluded.duration,updated_at=now() returning id_episode`, seasonID, ep.Number, ep.Title, ep.Description, ep.Duration).Scan(&episodeID)
			if err != nil {
				return err
			}
			if err := ensureAsset(ctx, tx, store, root, "episode", episodeID); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureAsset(ctx context.Context, tx pgx.Tx, store storage.Store, root, kind string, targetID int) error {
	column := "id_movie"
	if kind == "episode" {
		column = "id_episode"
	}
	var assetID string
	err := tx.QueryRow(ctx, fmt.Sprintf(`select id::text from video_assets where %s=$1`, column), targetID).Scan(&assetID)
	if err != pgx.ErrNoRows {
		if err != nil {
			return err
		}
	} else {
		err = tx.QueryRow(ctx, fmt.Sprintf(`insert into video_assets(kind,%s,status) values($1,$2,'pending') returning id::text`, column), kind, targetID).Scan(&assetID)
		if err != nil {
			return err
		}
	}
	sourceKey := filepath.ToSlash(filepath.Join("sources", assetID, "source.mp4"))
	f, err := os.Open(filepath.Join(root, "video", "video.mp4"))
	if err != nil {
		return err
	}
	if err = store.Put(sourceKey, f); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `update video_assets set source_path=$2,updated_at=now() where id=$1`, assetID, sourceKey); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into video_jobs(asset_id,status) select $1,'queued' where not exists(select 1 from video_jobs where asset_id=$1)`, assetID)
	return err
}

func slug(value string) string {
	return strings.Trim(strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, value), "_")
}
func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

var _ io.Reader
