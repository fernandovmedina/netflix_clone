package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	VideoSource string   `json:"video_source"`
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
	generateSQL := flag.String("generate-sql", "", "write standalone catalog seed SQL and exit")
	generateShortVideo := flag.String("generate-short-video", "", "generate the short seed clip at the given path and exit")
	resetCatalog := flag.Bool("reset-catalog", false, "replace catalog data and catalog media before importing")
	flag.Parse()
	seedRoot := os.Getenv("SEED_ROOT")
	if seedRoot == "" {
		seedRoot = "seed"
		info, err := os.Stat(seedRoot)
		if err != nil || !info.IsDir() {
			seedRoot = filepath.Join("..", "..", "seed")
		}
	}
	if *generateSQL != "" {
		if err := generateSeedSQL(seedRoot, *generateSQL); err != nil {
			log.Fatal(err)
		}
		log.Printf("generated %s", *generateSQL)
		return
	}
	if *generateShortVideo != "" {
		if err := generateShortSeedVideo(seedRoot, *generateShortVideo); err != nil {
			log.Fatal(err)
		}
		log.Printf("generated %s", *generateShortVideo)
		return
	}
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
	if *resetCatalog {
		current, checkErr := catalogIsCurrent(ctx, pool, seedRoot, mediaRoot)
		if checkErr != nil {
			log.Fatal(checkErr)
		}
		if current {
			if err := importAll(ctx, pool, store, seedRoot); err != nil {
				log.Fatal(err)
			}
			log.Printf("catalog and seed video already current; destructive reset skipped")
			return
		}
		counts, err := resetAndImport(ctx, pool, store, seedRoot, mediaRoot)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("catalog reset removed: %s", counts)
		return
	}
	if err := importAll(ctx, pool, store, seedRoot); err != nil {
		log.Fatal(err)
	}
}

func resetAndImport(ctx context.Context, pool dbPool, store storage.Store, root, mediaRoot string) (counts string, returnErr error) {
	movies, seriesData, err := manifests(root)
	if err != nil {
		return "", err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "select pg_advisory_xact_lock($1)", int64(0x4e465853454544)); err != nil {
		return "", err
	}
	// Preserve user library state when it points to stable seeded identities.
	statements := []string{
		`create temporary table seed_saved_favorites on commit drop as select f.user_id,t.type,t.title from favorites f join titles t on t.id_title=f.id_title`,
		`create temporary table seed_saved_movie_progress on commit drop as select wp.user_id,t.title,wp.current_time_seconds from watch_progress wp join movies m on m.id_movie=wp.id_movie join titles t on t.id_title=m.id_title where wp.id_movie is not null`,
		`create temporary table seed_saved_episode_progress on commit drop as select wp.user_id,t.title series_title,s.season_number,e.episode_number,wp.current_time_seconds from watch_progress wp join episodes e on e.id_episode=wp.id_episode join seasons s on s.id_season=e.id_season join series sr on sr.id_series=s.id_series join titles t on t.id_title=sr.id_title where wp.id_episode is not null`,
	}
	for _, statement := range statements {
		if _, err = tx.Exec(ctx, statement); err != nil {
			return "", err
		}
	}
	tables := []string{"video_jobs", "video_assets", "watch_progress", "favorites", "title_actors", "title_categories", "title_genres", "episodes", "seasons", "movies", "series", "titles", "actors", "categories", "genres"}
	removed := make([]string, 0, len(tables))
	for _, table := range tables {
		tag, execErr := tx.Exec(ctx, "delete from "+table)
		if execErr != nil {
			return "", execErr
		}
		removed = append(removed, fmt.Sprintf("%s=%d", table, tag.RowsAffected()))
	}
	swap, err := beginMediaSwap(mediaRoot)
	if err != nil {
		return "", err
	}
	mediaCommitted := false
	defer func() {
		if !mediaCommitted {
			returnErr = errors.Join(returnErr, swap.rollback())
		}
	}()
	if err = importData(ctx, tx, store, root, movies, seriesData); err != nil {
		return "", err
	}
	restore := []string{
		`insert into favorites(user_id,id_title) select f.user_id,t.id_title from seed_saved_favorites f join titles t on t.type=f.type and lower(t.title)=lower(f.title) on conflict do nothing`,
		`insert into watch_progress(user_id,id_movie,current_time_seconds) select p.user_id,m.id_movie,p.current_time_seconds from seed_saved_movie_progress p join titles t on t.type='Movie' and lower(t.title)=lower(p.title) join movies m on m.id_title=t.id_title on conflict(user_id,id_movie) where id_movie is not null do update set current_time_seconds=excluded.current_time_seconds,updated_at=now()`,
		`insert into watch_progress(user_id,id_episode,current_time_seconds) select p.user_id,e.id_episode,p.current_time_seconds from seed_saved_episode_progress p join titles t on t.type='TV Show' and lower(t.title)=lower(p.series_title) join series sr on sr.id_title=t.id_title join seasons s on s.id_series=sr.id_series and s.season_number=p.season_number join episodes e on e.id_season=s.id_season and e.episode_number=p.episode_number on conflict(user_id,id_episode) where id_episode is not null do update set current_time_seconds=excluded.current_time_seconds,updated_at=now()`,
	}
	for _, statement := range restore {
		if _, err = tx.Exec(ctx, statement); err != nil {
			return "", err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	mediaCommitted = true
	counts = strings.Join(removed, ", ")
	if err = swap.discardBackup(); err != nil {
		return counts, fmt.Errorf("catalog committed but old media backup could not be removed: %w", err)
	}
	return counts, nil
}

type mediaTree struct {
	name, target, backup string
	existed              bool
}

type mediaSwap struct {
	backupRoot string
	trees      []mediaTree
}

func beginMediaSwap(mediaRoot string) (*mediaSwap, error) {
	root := filepath.Clean(mediaRoot)
	backupRoot, err := os.MkdirTemp(root, ".seed-reset-backup-")
	if err != nil {
		return nil, err
	}
	swap := &mediaSwap{backupRoot: backupRoot}
	succeeded := false
	defer func() {
		if !succeeded {
			_ = swap.rollback()
		}
	}()
	for _, name := range []string{"hls", "sources", "thumbnails"} {
		tree := mediaTree{name: name, target: filepath.Join(root, name), backup: filepath.Join(backupRoot, name)}
		if filepath.Dir(tree.target) != root || filepath.Dir(tree.backup) != backupRoot {
			return nil, fmt.Errorf("refusing unsafe media swap target %q", tree.target)
		}
		info, statErr := os.Lstat(tree.target)
		if statErr == nil {
			if !info.IsDir() {
				return nil, fmt.Errorf("media tree %q is not a directory", tree.target)
			}
			tree.existed = true
			if err = os.Rename(tree.target, tree.backup); err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(statErr) {
			return nil, statErr
		}
		swap.trees = append(swap.trees, tree)
		if name == "thumbnails" && tree.existed {
			if err = copyTree(tree.backup, tree.target); err != nil {
				return nil, err
			}
		} else if err = os.MkdirAll(tree.target, 0o755); err != nil {
			return nil, err
		}
	}
	succeeded = true
	return swap, nil
}

func (swap *mediaSwap) rollback() error {
	if swap == nil {
		return nil
	}
	var result error
	for index := len(swap.trees) - 1; index >= 0; index-- {
		tree := swap.trees[index]
		if err := os.RemoveAll(tree.target); err != nil {
			result = errors.Join(result, err)
			continue
		}
		if tree.existed {
			if err := os.Rename(tree.backup, tree.target); err != nil {
				result = errors.Join(result, err)
			}
		}
	}
	if err := os.RemoveAll(swap.backupRoot); err != nil {
		result = errors.Join(result, err)
	}
	return result
}

func (swap *mediaSwap) discardBackup() error {
	return os.RemoveAll(swap.backupRoot)
}

func copyTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in media tree: %s", path)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		return errors.Join(copyErr, input.Close(), output.Close())
	})
}

type dbPool interface {
	Begin(context.Context) (pgx.Tx, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func catalogIsCurrent(ctx context.Context, pool dbPool, root, mediaRoot string) (bool, error) {
	movies, seriesData, err := manifests(root)
	if err != nil {
		return false, err
	}
	expectedSeasons, expectedEpisodes := 0, 0
	movieNames := make([]string, 0, len(movies.Movies))
	seriesNames := make([]string, 0, len(seriesData.Series))
	for _, item := range movies.Movies {
		movieNames = append(movieNames, item.Name)
	}
	for _, item := range seriesData.Series {
		seriesNames = append(seriesNames, item.Name)
		for _, season := range item.Seasons {
			expectedSeasons++
			expectedEpisodes += len(season.Episodes)
		}
	}
	// Only entries whose seed clip is actually on disk are seed-managed. Every
	// other asset belongs to an administrator upload, which a reset must not
	// treat as drift and must not destroy.
	expectedSources := make(map[string]string, len(movies.Movies)+expectedEpisodes)
	for _, item := range movies.Movies {
		source, present, sourceErr := seedVideoSource(root, item.VideoSource)
		if sourceErr != nil {
			return false, sourceErr
		}
		if present {
			expectedSources[assetIdentity("movie", item.Name, 0, 0)] = source
		}
	}
	for _, item := range seriesData.Series {
		source, present, sourceErr := seedVideoSource(root, item.VideoSource)
		if sourceErr != nil {
			return false, sourceErr
		}
		if !present {
			continue
		}
		for _, itemSeason := range item.Seasons {
			for _, itemEpisode := range itemSeason.Episodes {
				expectedSources[assetIdentity("episode", item.Name, itemSeason.Number, itemEpisode.Number)] = source
			}
		}
	}
	var titles, matchingMovies, matchingSeries, movieCount, seriesCount, seasonCount, episodeCount int
	err = pool.QueryRow(ctx, `select
		(select count(*) from titles),
		(select count(*) from titles where type='Movie' and title=any($1)),
		(select count(*) from titles where type='TV Show' and title=any($2)),
		(select count(*) from movies),(select count(*) from series),(select count(*) from seasons),(select count(*) from episodes)`, movieNames, seriesNames).Scan(&titles, &matchingMovies, &matchingSeries, &movieCount, &seriesCount, &seasonCount, &episodeCount)
	if err != nil {
		return false, err
	}
	if titles != len(movieNames)+len(seriesNames) || matchingMovies != len(movieNames) || matchingSeries != len(seriesNames) || movieCount != len(movieNames) || seriesCount != len(seriesNames) || seasonCount != expectedSeasons || episodeCount != expectedEpisodes {
		return false, nil
	}
	rows, err := pool.Query(ctx, `select va.kind,coalesce(movie_title.title,series_title.title),coalesce(se.season_number,0),coalesce(e.episode_number,0),coalesce(va.source_path,'')
		from video_assets va
		left join movies m on m.id_movie=va.id_movie left join titles movie_title on movie_title.id_title=m.id_title
		left join episodes e on e.id_episode=va.id_episode left join seasons se on se.id_season=e.id_season left join series sr on sr.id_series=se.id_series left join titles series_title on series_title.id_title=sr.id_title
		where va.status<>'superseded'`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	expectedHashes := make(map[string][sha256.Size]byte, 2)
	seen := make(map[string]bool, len(expectedSources))
	for rows.Next() {
		var kind, title, sourcePath string
		var seasonNumber, episodeNumber int
		if err = rows.Scan(&kind, &title, &seasonNumber, &episodeNumber, &sourcePath); err != nil {
			return false, err
		}
		identity := assetIdentity(kind, title, seasonNumber, episodeNumber)
		expectedPath, ok := expectedSources[identity]
		if !ok {
			// An administrator upload, or a title whose seed clip is absent.
			// Neither is seed-managed, so neither counts as drift.
			continue
		}
		if seen[identity] || sourcePath == "" {
			return false, nil
		}
		seen[identity] = true
		expectedHash, ok := expectedHashes[expectedPath]
		if !ok {
			expectedHash, err = fileSHA256(expectedPath)
			if err != nil {
				return false, err
			}
			expectedHashes[expectedPath] = expectedHash
		}
		storedHash, hashErr := fileSHA256(filepath.Join(mediaRoot, filepath.FromSlash(sourcePath)))
		if os.IsNotExist(hashErr) || hashErr == nil && storedHash != expectedHash {
			return false, nil
		}
		if hashErr != nil {
			return false, hashErr
		}
	}
	if err = rows.Err(); err != nil {
		return false, err
	}
	return len(seen) == len(expectedSources), nil
}

func assetIdentity(kind, title string, seasonNumber, episodeNumber int) string {
	return fmt.Sprintf("%s|%s|%d|%d", kind, strings.ToLower(strings.TrimSpace(title)), seasonNumber, episodeNumber)
}

func fileSHA256(path string) ([sha256.Size]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return [sha256.Size]byte{}, err
	}
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result, nil
}

func importAll(ctx context.Context, pool dbPool, store storage.Store, root string) error {
	movies, seriesData, err := manifests(root)
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
	if err = importData(ctx, tx, store, root, movies, seriesData); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func manifests(root string) (manifest, manifest, error) {
	movies, err := readManifest(filepath.Join(root, "movies", "seed.json"))
	if err != nil {
		return manifest{}, manifest{}, err
	}
	seriesData, err := readManifest(filepath.Join(root, "series", "seed.json"))
	return movies, seriesData, err
}

func importData(ctx context.Context, tx pgx.Tx, store storage.Store, root string, movies, seriesData manifest) error {
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
	return nil
}

func generateSeedSQL(root, destination string) error {
	movieData, err := readManifest(filepath.Join(root, "movies", "seed.json"))
	if err != nil {
		return err
	}
	seriesData, err := readManifest(filepath.Join(root, "series", "seed.json"))
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("-- Generated by: cd database/seed && go run . -generate-sql ../exec.sql\n")
	b.WriteString("-- Execute from the repository root: docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U netflix -d netflix < database/exec.sql\n")
	b.WriteString("-- Generated from seed/movies/seed.json and seed/series/seed.json. Do not edit by hand.\n\nBEGIN;\n")
	fmt.Fprintf(&b, "SELECT pg_advisory_xact_lock(%d);\n", int64(0x4e465853454544))
	for _, item := range movieData.Movies {
		writeSQLTitle(&b, item.baseTitle, "Movie")
		fmt.Fprintf(&b, "INSERT INTO movies(id_title,duration) SELECT id_title,%d FROM titles WHERE type='Movie' AND lower(title)=lower(%s) ORDER BY id_title LIMIT 1 ON CONFLICT(id_title) DO UPDATE SET duration=excluded.duration,deleted_at=NULL,updated_at=now();\n", item.Duration, sqlLiteral(item.Name))
	}
	for _, item := range seriesData.Series {
		writeSQLTitle(&b, item.baseTitle, "TV Show")
		fmt.Fprintf(&b, "INSERT INTO series(id_title,number_of_seasons) SELECT id_title,%d FROM titles WHERE type='TV Show' AND lower(title)=lower(%s) ORDER BY id_title LIMIT 1 ON CONFLICT(id_title) DO UPDATE SET number_of_seasons=excluded.number_of_seasons,deleted_at=NULL,updated_at=now();\n", item.NumberOfSeasons, sqlLiteral(item.Name))
		for _, season := range item.Seasons {
			fmt.Fprintf(&b, "INSERT INTO seasons(id_series,season_number,number_of_episodes) SELECT s.id_series,%d,%d FROM series s JOIN titles t ON t.id_title=s.id_title WHERE t.type='TV Show' AND lower(t.title)=lower(%s) ORDER BY s.id_series LIMIT 1 ON CONFLICT(id_series,season_number) DO UPDATE SET number_of_episodes=excluded.number_of_episodes,deleted_at=NULL,updated_at=now();\n", season.Number, season.NumberOfEpisodes, sqlLiteral(item.Name))
			for _, ep := range season.Episodes {
				fmt.Fprintf(&b, "INSERT INTO episodes(id_season,episode_number,title,description,duration) SELECT se.id_season,%d,%s,%s,%d FROM seasons se JOIN series s ON s.id_series=se.id_series JOIN titles t ON t.id_title=s.id_title WHERE t.type='TV Show' AND lower(t.title)=lower(%s) AND se.season_number=%d ORDER BY se.id_season LIMIT 1 ON CONFLICT(id_season,episode_number) DO UPDATE SET title=excluded.title,description=excluded.description,duration=excluded.duration,deleted_at=NULL,updated_at=now();\n", ep.Number, sqlLiteral(ep.Title), sqlLiteral(ep.Description), ep.Duration, sqlLiteral(item.Name), season.Number)
			}
		}
	}
	b.WriteString("COMMIT;\n")
	return os.WriteFile(destination, []byte(b.String()), 0o644)
}

func generateShortSeedVideo(root, destination string) error {
	source := filepath.Join(root, "video", "video.mp4")
	destination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".video-short-*.mp4")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	if err = temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	succeeded := false
	defer func() {
		if !succeeded {
			_ = os.Remove(temporaryPath)
		}
	}()
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", source, "-t", "6", "-map", "0:v:0", "-map", "0:a:0", "-map_metadata", "-1", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-profile:v", "main", "-preset", "veryfast", "-crf", "23", "-g", "50", "-keyint_min", "50", "-sc_threshold", "0", "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", "-f", "mp4", temporaryPath}
	if output, commandErr := exec.Command("ffmpeg", args...).CombinedOutput(); commandErr != nil {
		return fmt.Errorf("ffmpeg: %w: %s", commandErr, strings.TrimSpace(string(output)))
	}
	if err = os.Chmod(temporaryPath, 0o644); err != nil {
		return err
	}
	if err = os.Rename(temporaryPath, destination); err != nil {
		return err
	}
	succeeded = true
	return nil
}

func writeSQLTitle(b *strings.Builder, item baseTitle, kind string) {
	director := "NULL"
	if strings.TrimSpace(item.Director) != "" {
		director = sqlLiteral(item.Director)
	}
	prefix := "movie"
	if kind == "TV Show" {
		prefix = "series"
	}
	thumbnail := "/api/v1/stream/thumbnails/" + prefix + "_" + slug(item.Name) + "_" + filepath.Base(item.Thumbnail)
	fmt.Fprintf(b, "INSERT INTO titles(type,title,description,director,year_released,thumbnail_url,published) SELECT %s,%s,%s,%s,%d,%s,true WHERE NOT EXISTS (SELECT 1 FROM titles WHERE type=%s AND lower(title)=lower(%s));\n", sqlLiteral(kind), sqlLiteral(item.Name), sqlLiteral(item.Description), director, item.Year, sqlLiteral(thumbnail), sqlLiteral(kind), sqlLiteral(item.Name))
	fmt.Fprintf(b, "UPDATE titles SET description=%s,director=%s,year_released=%d,thumbnail_url=%s,published=true,deleted_at=NULL,updated_at=now() WHERE id_title=(SELECT id_title FROM titles WHERE type=%s AND lower(title)=lower(%s) ORDER BY id_title LIMIT 1);\n", sqlLiteral(item.Description), director, item.Year, sqlLiteral(thumbnail), sqlLiteral(kind), sqlLiteral(item.Name))
	for _, join := range []string{"title_genres", "title_actors", "title_categories"} {
		fmt.Fprintf(b, "DELETE FROM %s WHERE id_title=(SELECT id_title FROM titles WHERE type=%s AND lower(title)=lower(%s) ORDER BY id_title LIMIT 1);\n", join, sqlLiteral(kind), sqlLiteral(item.Name))
	}
	for _, pair := range []struct {
		table, id, join string
		names           []string
	}{{"genres", "id_genre", "title_genres", item.Genres}, {"actors", "id_actor", "title_actors", item.Cast}, {"categories", "id_category", "title_categories", []string{map[string]string{"Movie": "Movies", "TV Show": "Series"}[kind]}}} {
		names := append([]string(nil), pair.names...)
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(b, "INSERT INTO %s(name) VALUES(%s) ON CONFLICT(name) DO UPDATE SET deleted_at=NULL,updated_at=now();\n", pair.table, sqlLiteral(name))
			fmt.Fprintf(b, "INSERT INTO %s(id_title,%s) SELECT t.id_title,v.%s FROM titles t JOIN %s v ON v.name=%s WHERE t.type=%s AND lower(t.title)=lower(%s) ON CONFLICT DO NOTHING;\n", pair.join, pair.id, pair.id, pair.table, sqlLiteral(name), sqlLiteral(kind), sqlLiteral(item.Name))
		}
	}
}

func sqlLiteral(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }

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
	return ensureAsset(ctx, tx, store, root, "movie", movieID, item.VideoSource)
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
			if err := ensureAsset(ctx, tx, store, root, "episode", episodeID, item.VideoSource); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureAsset(ctx context.Context, tx pgx.Tx, store storage.Store, root, kind string, targetID int, videoSource string) error {
	column := "id_movie"
	if kind == "episode" {
		column = "id_episode"
	}
	videoPath, present, err := seedVideoSource(root, videoSource)
	if err != nil {
		return err
	}
	var assetID string
	err = tx.QueryRow(ctx, fmt.Sprintf(`select id::text from video_assets where %s=$1`, column), targetID).Scan(&assetID)
	if err == nil {
		return nil
	}
	if err != pgx.ErrNoRows {
		return err
	}
	// Seed media is deliberately not distributed with the repository. Without a
	// clip on disk the catalog still imports and the title simply carries no
	// video until an administrator uploads one.
	if !present {
		return nil
	}
	err = tx.QueryRow(ctx, fmt.Sprintf(`insert into video_assets(kind,%s,status) values($1,$2,'pending') returning id::text`, column), kind, targetID).Scan(&assetID)
	if err != nil {
		return err
	}
	sourceKey := filepath.ToSlash(filepath.Join("sources", assetID, "source.mp4"))
	f, err := os.Open(videoPath)
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

// seedVideoSource resolves the clip a manifest entry selects and reports
// whether it exists locally. A fresh clone has no seed media at all.
func seedVideoSource(root, selection string) (string, bool, error) {
	path, err := seedVideoPath(root, selection)
	if err != nil {
		return "", false, err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return path, false, nil
	}
	if err != nil {
		return "", false, err
	}
	return path, !info.IsDir() && info.Size() > 0, nil
}

func seedVideoPath(root, selection string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(selection)) {
	case "", "short":
		return filepath.Join(root, "video", "video-short.mp4"), nil
	case "long":
		return filepath.Join(root, "video", "video.mp4"), nil
	default:
		return "", fmt.Errorf("invalid video_source %q (want short or long)", selection)
	}
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
