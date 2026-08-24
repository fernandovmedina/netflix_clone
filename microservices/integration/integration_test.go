//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type suite struct {
	base, adminEmail, adminPassword, jwtSecret string
	db                                         *pgxpool.Pool
}

type apiClient struct {
	t *testing.T
	c *http.Client
	s *suite
}

type response struct {
	Status int
	Header http.Header
	Body   []byte
}

type user struct {
	ID, Email, Password string
	API                 *apiClient
}

func TestMain(m *testing.M) {
	env := loadEnv()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = env["DATABASE_URL"]
		u, err := url.Parse(dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "integration: parse DATABASE_URL: %v\n", err)
			os.Exit(1)
		}
		u.Host = "localhost:5433"
		dsn = u.String()
	}
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: open database: %v\n", err)
		os.Exit(1)
	}
	if err = db.Ping(context.Background()); err != nil {
		db.Close()
		fmt.Fprintf(os.Stderr, "integration: ping database: %v\n", err)
		os.Exit(1)
	}
	titleCounts := func() (int, int, error) {
		var public, total int
		err := db.QueryRow(context.Background(), `select count(*) from titles t where t.deleted_at is null and t.published=true and (
			exists(select 1 from movies m join video_assets va on va.id_movie=m.id_movie and va.status='ready' where m.id_title=t.id_title and m.deleted_at is null)
			or exists(select 1 from series s join seasons se on se.id_series=s.id_series and se.deleted_at is null join episodes e on e.id_season=se.id_season and e.deleted_at is null join video_assets va on va.id_episode=e.id_episode and va.status='ready' where s.id_title=t.id_title and s.deleted_at is null)
		)`).Scan(&public)
		if err == nil {
			err = db.QueryRow(context.Background(), `select count(*) from titles`).Scan(&total)
		}
		return public, total, err
	}
	beforePublic, beforeTotal, err := titleCounts()
	if err != nil {
		db.Close()
		fmt.Fprintf(os.Stderr, "integration: count public titles before suite: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	afterPublic, afterTotal, err := titleCounts()
	db.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: count public titles after suite: %v\n", err)
		os.Exit(1)
	}
	if afterPublic != beforePublic || afterTotal != beforeTotal {
		fmt.Fprintf(os.Stderr, "integration: title counts changed (public %d→%d, total %d→%d); fixture cleanup is incomplete\n", beforePublic, afterPublic, beforeTotal, afterTotal)
		code = 1
	}
	os.Exit(code)
}

func loadEnv() map[string]string {
	out := map[string]string{}
	data, _ := os.ReadFile(filepath.Join("..", "..", ".env"))
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			out[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return out
}

func newSuite(t *testing.T) *suite {
	t.Helper()
	env := loadEnv()
	base := os.Getenv("BASE_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = env["DATABASE_URL"]
		u, err := url.Parse(dsn)
		if err != nil {
			t.Fatalf("parse DATABASE_URL: %v", err)
		}
		u.Host = "localhost:5433"
		dsn = u.String()
	}
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err = db.Ping(context.Background()); err != nil {
		db.Close()
		t.Fatalf("ping database: %v", err)
	}
	t.Cleanup(db.Close)
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = env["ADMIN_EMAIL"]
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = env["ADMIN_PASSWORD"]
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = env["JWT_SECRET"]
	}
	return &suite{base: strings.TrimRight(base, "/"), adminEmail: adminEmail, adminPassword: adminPassword, jwtSecret: jwtSecret, db: db}
}

func newAPI(t *testing.T, s *suite) *apiClient {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &apiClient{t: t, s: s, c: &http.Client{Jar: jar, Timeout: 30 * time.Second}}
}

func (a *apiClient) do(method, path string, body any, headers map[string]string) response {
	a.t.Helper()
	var reader io.Reader
	var rendered string
	if body != nil {
		switch value := body.(type) {
		case []byte:
			reader, rendered = bytes.NewReader(value), string(value)
		case string:
			reader, rendered = strings.NewReader(value), value
		default:
			data, err := json.Marshal(value)
			if err != nil {
				a.t.Fatalf("marshal %s %s: %v", method, path, err)
			}
			reader, rendered = bytes.NewReader(data), string(data)
		}
	}
	req, err := http.NewRequest(method, a.s.base+path, reader)
	if err != nil {
		a.t.Fatalf("build %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := a.c.Do(req)
	if err != nil {
		a.t.Fatalf("%s %s body=%s: %v", method, path, rendered, err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		a.t.Fatalf("read %s %s response: %v", method, path, err)
	}
	return response{Status: res.StatusCode, Header: res.Header.Clone(), Body: data}
}

func (a *apiClient) authStrict(method, path string, body any) response {
	a.t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for {
		result := a.do(method, path, body, nil)
		if result.Status != http.StatusTooManyRequests || time.Now().After(deadline) {
			return result
		}
		time.Sleep(5 * time.Second)
	}
}

func requireStatus(t *testing.T, method, path string, got response, want ...int) {
	t.Helper()
	for _, status := range want {
		if got.Status == status {
			return
		}
	}
	t.Fatalf("request: %s %s\nstatus: %d (want %v)\nresponse body: %s", method, path, got.Status, want, got.Body)
}

func decodeBody[T any](t *testing.T, got response) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(got.Body, &out); err != nil {
		t.Fatalf("decode status=%d body=%q: %v", got.Status, got.Body, err)
	}
	return out
}

func suffix() string { return strings.ReplaceAll(uuid.NewString(), "-", "")[:12] }

func (s *suite) signup(t *testing.T, label string) user {
	t.Helper()
	a := newAPI(t, s)
	email := fmt.Sprintf("integration-%s-%s@example.test", label, suffix())
	password := "Integration-Password-42!"
	r := a.authStrict(http.MethodPost, "/api/v1/auth/signup", map[string]any{"name": "Integration " + label, "email": email, "password": password})
	requireStatus(t, http.MethodPost, "/api/v1/auth/signup", r, http.StatusCreated)
	var payload struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	payload = decodeBody[typeofSignup](t, r).asPayload()
	u := user{ID: payload.User.ID, Email: email, Password: password, API: a}
	t.Cleanup(func() { s.cleanupUser(u.ID) })
	return u
}

func (s *suite) arrangedUser(t *testing.T, label string) user {
	t.Helper()
	email := fmt.Sprintf("integration-%s-%s@example.test", label, suffix())
	var id string
	if err := s.db.QueryRow(context.Background(), `insert into users(email,name,role) values($1,$2,'user') returning id::text`, email, "Integration "+label).Scan(&id); err != nil {
		t.Fatal(err)
	}
	a := newAPI(t, s)
	setAccessCookie(t, a, s.accessToken(id, email, "user"))
	u := user{ID: id, Email: email, API: a}
	t.Cleanup(func() { s.cleanupUser(id) })
	return u
}

func (s *suite) accessToken(id, email, role string) string {
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	now := time.Now().Unix()
	payload, _ := json.Marshal(map[string]any{"sub": id, "email": email, "role": role, "iat": now, "exp": now + 900, "jti": uuid.NewString(), "iss": "netflix-clone", "aud": "netflix-clone"})
	encode := func(data []byte) string { return base64.RawURLEncoding.EncodeToString(data) }
	unsigned := encode(header) + "." + encode(payload)
	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + encode(mac.Sum(nil))
}

func setAccessCookie(t *testing.T, a *apiClient, token string) {
	t.Helper()
	base, _ := url.Parse(a.s.base)
	a.c.Jar.SetCookies(base, []*http.Cookie{{Name: "access_token", Value: token, Path: "/"}})
}

// Named type keeps generic JSON decoding readable on Go versions without anonymous-type inference.
type typeofSignup struct {
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (v typeofSignup) asPayload() struct {
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
} {
	return v
}

func (s *suite) cleanupUser(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)
	_, _ = tx.Exec(ctx, `delete from discount_redemptions where user_id=$1`, id)
	_, _ = tx.Exec(ctx, `delete from subscriptions where user_id=$1`, id)
	_, _ = tx.Exec(ctx, `delete from payments where user_id=$1`, id)
	_, _ = tx.Exec(ctx, `delete from favorites where user_id=$1`, id)
	_, _ = tx.Exec(ctx, `delete from watch_progress where user_id=$1`, id)
	_, _ = tx.Exec(ctx, `delete from users where id=$1`, id)
	_ = tx.Commit(ctx)
}

func (s *suite) cleanupTitle(id int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)
	_, _ = tx.Exec(ctx, `delete from video_jobs where asset_id in (select id from video_assets where id_movie in(select id_movie from movies where id_title=$1) or id_episode in(select e.id_episode from series s join seasons se on se.id_series=s.id_series join episodes e on e.id_season=se.id_season where s.id_title=$1))`, id)
	_, _ = tx.Exec(ctx, `delete from video_assets where id_movie in (select id_movie from movies where id_title=$1) or id_episode in(select e.id_episode from series s join seasons se on se.id_series=s.id_series join episodes e on e.id_season=se.id_season where s.id_title=$1)`, id)
	_, _ = tx.Exec(ctx, `delete from favorites where id_title=$1`, id)
	_, _ = tx.Exec(ctx, `delete from watch_progress where id_movie in (select id_movie from movies where id_title=$1) or id_episode in(select e.id_episode from series s join seasons se on se.id_series=s.id_series join episodes e on e.id_season=se.id_season where s.id_title=$1)`, id)
	_, _ = tx.Exec(ctx, `delete from title_actors where id_title=$1`, id)
	_, _ = tx.Exec(ctx, `delete from title_categories where id_title=$1`, id)
	_, _ = tx.Exec(ctx, `delete from title_genres where id_title=$1`, id)
	_, _ = tx.Exec(ctx, `delete from episodes where id_season in(select se.id_season from series s join seasons se on se.id_series=s.id_series where s.id_title=$1)`, id)
	_, _ = tx.Exec(ctx, `delete from seasons where id_series in(select id_series from series where id_title=$1)`, id)
	_, _ = tx.Exec(ctx, `delete from series where id_title=$1`, id)
	_, _ = tx.Exec(ctx, `delete from movies where id_title=$1`, id)
	_, _ = tx.Exec(ctx, `delete from titles where id_title=$1`, id)
	_ = tx.Commit(ctx)
}

func (s *suite) admin(t *testing.T) *apiClient {
	t.Helper()
	a := newAPI(t, s)
	var id, role string
	if err := s.db.QueryRow(context.Background(), `select id::text,role::text from users where email=$1 and deleted_at is null`, s.adminEmail).Scan(&id, &role); err != nil {
		t.Fatalf("load admin: %v", err)
	}
	setAccessCookie(t, a, s.accessToken(id, s.adminEmail, role))
	return a
}

func cookieValue(t *testing.T, a *apiClient, name string) string {
	t.Helper()
	u, _ := url.Parse(a.s.base + "/api/v1/auth/refresh")
	for _, c := range a.c.Jar.Cookies(u) {
		if c.Name == name {
			return c.Value
		}
	}
	t.Fatalf("cookie %q not found", name)
	return ""
}

func TestAuthAcrossInstances(t *testing.T) {
	s := newSuite(t)
	u := s.signup(t, "auth")
	for i := 0; i < 12; i++ {
		r := u.API.do(http.MethodGet, "/api/v1/auth/me", nil, nil)
		requireStatus(t, http.MethodGet, "/api/v1/auth/me", r, http.StatusOK)
	}
	r := u.API.do(http.MethodPost, "/api/v1/auth/refresh", nil, nil)
	requireStatus(t, http.MethodPost, "/api/v1/auth/refresh", r, http.StatusOK)
	for i := 0; i < 12; i++ {
		requireStatus(t, http.MethodGet, "/api/v1/auth/me", u.API.do(http.MethodGet, "/api/v1/auth/me", nil, nil), http.StatusOK)
	}
	login := newAPI(t, s)
	requireStatus(t, http.MethodPost, "/api/v1/auth/login", login.authStrict(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": u.Email, "password": u.Password}), http.StatusOK)
	loggedInRefresh := cookieValue(t, login, "refresh_token")
	requireStatus(t, http.MethodPost, "/api/v1/auth/logout", login.do(http.MethodPost, "/api/v1/auth/logout", nil, nil), http.StatusNoContent)
	requireStatus(t, http.MethodGet, "/api/v1/auth/me", login.do(http.MethodGet, "/api/v1/auth/me", nil, nil), http.StatusUnauthorized)
	loggedOut := newAPI(t, s)
	refreshURL, _ := url.Parse(s.base + "/api/v1/auth/refresh")
	loggedOut.c.Jar.SetCookies(refreshURL, []*http.Cookie{{Name: "refresh_token", Value: loggedInRefresh, Path: "/api/v1/auth"}})
	requireStatus(t, http.MethodPost, "/api/v1/auth/refresh", loggedOut.do(http.MethodPost, "/api/v1/auth/refresh", nil, nil), http.StatusUnauthorized)
	garbage := newAPI(t, s)
	base, _ := url.Parse(s.base + "/api/v1/auth/refresh")
	garbage.c.Jar.SetCookies(base, []*http.Cookie{{Name: "access_token", Value: "garbage.expired.token", Path: "/"}})
	requireStatus(t, http.MethodGet, "/api/v1/auth/me", garbage.do(http.MethodGet, "/api/v1/auth/me", nil, nil), http.StatusUnauthorized)
}

func TestRefreshRotationConcurrent(t *testing.T) {
	s := newSuite(t)
	u := s.signup(t, "rotation")
	refresh := cookieValue(t, u.API, "refresh_token")
	base, _ := url.Parse(s.base)
	clients := []*apiClient{newAPI(t, s), newAPI(t, s)}
	for _, c := range clients {
		c.c.Jar.SetCookies(base, []*http.Cookie{{Name: "refresh_token", Value: refresh, Path: "/api/v1/auth"}})
	}
	start := make(chan struct{})
	results := make(chan response, 2)
	var wg sync.WaitGroup
	for _, c := range clients {
		wg.Add(1)
		go func(c *apiClient) {
			defer wg.Done()
			<-start
			results <- c.do(http.MethodPost, "/api/v1/auth/refresh", nil, nil)
		}(c)
	}
	close(start)
	wg.Wait()
	close(results)
	counts := map[int]int{}
	var rotated string
	for r := range results {
		counts[r.Status]++
		if r.Status == http.StatusOK {
			for _, c := range clients {
				if value := findCookie(c, base, "refresh_token"); value != "" && value != refresh {
					rotated = value
				}
			}
		}
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusUnauthorized] != 1 {
		t.Fatalf("concurrent refresh statuses=%v; want exactly one 200 and one 401", counts)
	}
	reuse := newAPI(t, s)
	reuse.c.Jar.SetCookies(base, []*http.Cookie{{Name: "refresh_token", Value: rotated, Path: "/api/v1/auth"}})
	requireStatus(t, http.MethodPost, "/api/v1/auth/refresh", reuse.do(http.MethodPost, "/api/v1/auth/refresh", nil, nil), http.StatusUnauthorized)
}

func findCookie(a *apiClient, base *url.URL, name string) string {
	for _, c := range a.c.Jar.Cookies(base) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

type movieFixture struct {
	TitleID, MovieID int
	AssetID          string
	Name             string
}

type seriesFixture struct {
	TitleID, SeriesID, SeasonID, EpisodeID int
	AssetID, Name                          string
}

func (s *suite) movieFixture(t *testing.T, published bool, status string) movieFixture {
	t.Helper()
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	f := movieFixture{Name: "integration-title-" + suffix(), AssetID: uuid.NewString()}
	if err = tx.QueryRow(ctx, `insert into titles(type,title,description,published) values('Movie',$1,'integration fixture',$2) returning id_title`, f.Name, published).Scan(&f.TitleID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `insert into movies(id_title,duration) values($1,2) returning id_movie`, f.TitleID).Scan(&f.MovieID); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `insert into video_assets(id,kind,id_movie,status,manifest_path,qualities,source_width,source_height) values($1,'movie',$2,$3,$4,'["144p"]',256,144)`, f.AssetID, f.MovieID, status, "hls/"+f.AssetID+"/master.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.cleanupTitle(f.TitleID) })
	return f
}

func (s *suite) seriesFixture(t *testing.T, published bool, status string) seriesFixture {
	t.Helper()
	ctx := context.Background()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	f := seriesFixture{Name: "integration-series-" + suffix(), AssetID: uuid.NewString()}
	if err = tx.QueryRow(ctx, `insert into titles(type,title,description,published) values('TV Show',$1,'integration fixture',$2) returning id_title`, f.Name, published).Scan(&f.TitleID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `insert into series(id_title,number_of_seasons) values($1,1) returning id_series`, f.TitleID).Scan(&f.SeriesID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `insert into seasons(id_series,season_number,number_of_episodes) values($1,1,1) returning id_season`, f.SeriesID).Scan(&f.SeasonID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `insert into episodes(id_season,episode_number,title,duration) values($1,1,'Integration episode',2) returning id_episode`, f.SeasonID).Scan(&f.EpisodeID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `insert into video_assets(id,kind,id_episode,status,manifest_path,qualities,source_width,source_height) values($1,'episode',$2,$3,$4,'["144p"]',256,144)`, f.AssetID, f.EpisodeID, status, "hls/"+f.AssetID+"/master.m3u8"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.cleanupTitle(f.TitleID) })
	return f
}

// TestCatalogVisibility pins the public catalog contract: publication alone
// decides whether a title is browsable, so a published title whose video is
// still pending appears with artwork and metadata and renders as "no video
// yet". What must never escape is the asset id, because that is the handle the
// player would give the streaming service.
func TestCatalogVisibility(t *testing.T) {
	s := newSuite(t)
	normal := s.arrangedUser(t, "catalog")
	admin := s.admin(t)
	unpublished := s.movieFixture(t, false, "ready")
	pending := s.movieFixture(t, true, "pending")

	// Unpublished stays completely hidden, ready asset or not.
	for _, path := range []string{
		"/api/v1/titles?q=" + url.QueryEscape(unpublished.Name),
		"/api/v1/titles/" + strconv.Itoa(unpublished.TitleID),
		"/api/v1/movies/" + strconv.Itoa(unpublished.MovieID),
		"/api/v1/home",
		"/api/v1/titles?limit=100",
	} {
		r := normal.API.do(http.MethodGet, path, nil, nil)
		if strings.Contains(path, "?q=") || path == "/api/v1/home" || strings.Contains(path, "limit=100") {
			requireStatus(t, http.MethodGet, path, r, http.StatusOK)
			if bytes.Contains(r.Body, []byte(unpublished.Name)) {
				t.Fatalf("GET %s leaked unpublished title %q: %s", path, unpublished.Name, r.Body)
			}
			continue
		}
		requireStatus(t, http.MethodGet, path, r, http.StatusNotFound)
	}

	// Published but still transcoding: browsable, never playable.
	for _, path := range []string{
		"/api/v1/titles?q=" + url.QueryEscape(pending.Name),
		"/api/v1/titles/" + strconv.Itoa(pending.TitleID),
		"/api/v1/movies/" + strconv.Itoa(pending.MovieID),
	} {
		r := normal.API.do(http.MethodGet, path, nil, nil)
		requireStatus(t, http.MethodGet, path, r, http.StatusOK)
		if !bytes.Contains(r.Body, []byte(pending.Name)) {
			t.Fatalf("GET %s hid published pending title %q: %s", path, pending.Name, r.Body)
		}
		if !bytes.Contains(r.Body, []byte(`"asset_id":null`)) {
			t.Fatalf("GET %s exposed an asset id for a pending title: %s", path, r.Body)
		}
		if bytes.Contains(r.Body, []byte(pending.AssetID)) {
			t.Fatalf("GET %s leaked the pending asset id %q: %s", path, pending.AssetID, r.Body)
		}
	}

	for _, f := range []movieFixture{unpublished, pending} {
		wantStatus := "pending"
		if f.AssetID == unpublished.AssetID {
			wantStatus = "ready"
		}
		r := admin.do(http.MethodGet, "/api/v1/titles?q="+url.QueryEscape(f.Name), nil, nil)
		requireStatus(t, http.MethodGet, "/api/v1/titles", r, http.StatusOK)
		if !bytes.Contains(r.Body, []byte(f.Name)) || !bytes.Contains(r.Body, []byte(`"asset_status":"`+wantStatus+`"`)) {
			t.Fatalf("admin projection missing title/state for %q: %s", f.Name, r.Body)
		}
	}

	pendingSeries := s.seriesFixture(t, true, "pending")
	for _, path := range []string{
		"/api/v1/titles?q=" + url.QueryEscape(pendingSeries.Name),
		"/api/v1/titles/" + strconv.Itoa(pendingSeries.TitleID),
		"/api/v1/series/" + strconv.Itoa(pendingSeries.TitleID),
	} {
		r := normal.API.do(http.MethodGet, path, nil, nil)
		requireStatus(t, http.MethodGet, path, r, http.StatusOK)
		if !bytes.Contains(r.Body, []byte(pendingSeries.Name)) {
			t.Fatalf("GET %s hid published pending series %q: %s", path, pendingSeries.Name, r.Body)
		}
		if bytes.Contains(r.Body, []byte(pendingSeries.AssetID)) {
			t.Fatalf("GET %s leaked the pending episode asset id %q: %s", path, pendingSeries.AssetID, r.Body)
		}
	}
	adminSeries := admin.do(http.MethodGet, "/api/v1/titles?q="+url.QueryEscape(pendingSeries.Name), nil, nil)
	requireStatus(t, http.MethodGet, "/api/v1/titles", adminSeries, http.StatusOK)
	if !bytes.Contains(adminSeries.Body, []byte(`"asset_status":"pending"`)) {
		t.Fatalf("admin projection omitted pending series: %s", adminSeries.Body)
	}
}

func TestCatalogDetailIncludesGenresAndCast(t *testing.T) {
	s := newSuite(t)
	u := s.arrangedUser(t, "catalog-metadata")
	var titleID int
	var contentType string
	var movieID *int
	var wantGenres, wantCast []string
	err := s.db.QueryRow(context.Background(), `select t.id_title,t.type::text,m.id_movie,
		array(select g.name from title_genres tg join genres g on g.id_genre=tg.id_genre where tg.id_title=t.id_title order by g.name),
		array(select a.name from title_actors ta join actors a on a.id_actor=ta.id_actor where ta.id_title=t.id_title order by a.name)
		from titles t left join movies m on m.id_title=t.id_title where t.published and exists(select 1 from title_genres where id_title=t.id_title)
		and exists(select 1 from title_actors where id_title=t.id_title)
		and (exists(select 1 from movies m join video_assets va on va.id_movie=m.id_movie and va.status='ready' where m.id_title=t.id_title)
		or exists(select 1 from series s join seasons se on se.id_series=s.id_series join episodes e on e.id_season=se.id_season join video_assets va on va.id_episode=e.id_episode and va.status='ready' where s.id_title=t.id_title))
		order by t.id_title limit 1`).Scan(&titleID, &contentType, &movieID, &wantGenres, &wantCast)
	if err != nil {
		t.Fatalf("find seeded title metadata: %v", err)
	}
	paths := []string{"/api/v1/titles/" + strconv.Itoa(titleID)}
	if contentType == "Movie" && movieID != nil {
		paths = append(paths, "/api/v1/movies/"+strconv.Itoa(*movieID))
	} else {
		paths = append(paths, "/api/v1/series/"+strconv.Itoa(titleID))
	}
	for _, path := range paths {
		r := u.API.do(http.MethodGet, path, nil, nil)
		requireStatus(t, http.MethodGet, path, r, http.StatusOK)
		got := decodeBody[struct {
			Genres []string `json:"genres"`
			Cast   []string `json:"cast"`
		}](t, r)
		if strings.Join(got.Genres, ",") != strings.Join(wantGenres, ",") || strings.Join(got.Cast, ",") != strings.Join(wantCast, ",") {
			t.Fatalf("%s metadata genres=%v cast=%v, want genres=%v cast=%v; body=%s", path, got.Genres, got.Cast, wantGenres, wantCast, r.Body)
		}
	}
}

func TestAdminMetadataAssignments(t *testing.T) {
	s := newSuite(t)
	admin := s.admin(t)
	var genreID, actorID int
	if err := s.db.QueryRow(context.Background(), `select (select id_genre from genres where deleted_at is null order by id_genre limit 1),(select id_actor from actors where deleted_at is null order by id_actor limit 1)`).Scan(&genreID, &actorID); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, collection string
		extra            map[string]any
	}{
		{name: "movie", collection: "movies", extra: map[string]any{"duration": 90}},
		{name: "series", collection: "series", extra: map[string]any{"number_of_seasons": 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := map[string]any{"title": "integration-metadata-" + suffix(), "genre_ids": []int{genreID, genreID}, "actor_ids": []int{actorID}}
			for key, value := range test.extra {
				body[key] = value
			}
			path := "/api/v1/admin/" + test.collection
			createdResponse := admin.do(http.MethodPost, path, body, nil)
			requireStatus(t, http.MethodPost, path, createdResponse, http.StatusCreated)
			created := decodeBody[struct {
				ID          int   `json:"id"`
				TitleID     int   `json:"title_id"`
				GenreIDs    []int `json:"genre_ids"`
				ActorIDs    []int `json:"actor_ids"`
				CategoryIDs []int `json:"category_ids"`
			}](t, createdResponse)
			t.Cleanup(func() { s.cleanupTitle(created.TitleID) })
			if len(created.GenreIDs) != 1 || created.GenreIDs[0] != genreID || len(created.ActorIDs) != 1 || created.ActorIDs[0] != actorID || len(created.CategoryIDs) != 1 {
				t.Fatalf("unexpected create metadata response: %+v", created)
			}
			body["genre_ids"] = []int{}
			delete(body, "category_ids")
			patchPath := path + "/" + strconv.Itoa(created.ID)
			patchedResponse := admin.do(http.MethodPatch, patchPath, body, nil)
			requireStatus(t, http.MethodPatch, patchPath, patchedResponse, http.StatusOK)
			patched := decodeBody[struct {
				GenreIDs    []int `json:"genre_ids"`
				ActorIDs    []int `json:"actor_ids"`
				CategoryIDs []int `json:"category_ids"`
			}](t, patchedResponse)
			if len(patched.GenreIDs) != 0 || len(patched.ActorIDs) != 1 || len(patched.CategoryIDs) != 1 {
				t.Fatalf("replacement/preservation semantics failed: %+v", patched)
			}
		})
	}
}

func TestAdminAuthorizationEveryRoute(t *testing.T) {
	s := newSuite(t)
	normal := s.arrangedUser(t, "rbac")
	anonymous := newAPI(t, s)
	routes := []struct{ method, path string }{
		{"POST", "/api/v1/admin/movies"}, {"PATCH", "/api/v1/admin/movies/1"}, {"DELETE", "/api/v1/admin/movies/1"},
		{"POST", "/api/v1/admin/series"}, {"PATCH", "/api/v1/admin/series/1"}, {"DELETE", "/api/v1/admin/series/1"},
		{"POST", "/api/v1/admin/series/1/seasons"}, {"POST", "/api/v1/admin/seasons/1/episodes"}, {"PATCH", "/api/v1/admin/seasons/1"}, {"DELETE", "/api/v1/admin/seasons/1"},
		{"PATCH", "/api/v1/admin/episodes/1"}, {"DELETE", "/api/v1/admin/episodes/1"}, {"POST", "/api/v1/admin/genres"}, {"PATCH", "/api/v1/admin/genres/1"}, {"DELETE", "/api/v1/admin/genres/1"},
		{"POST", "/api/v1/admin/actors"}, {"PATCH", "/api/v1/admin/actors/1"}, {"DELETE", "/api/v1/admin/actors/1"}, {"POST", "/api/v1/admin/categories"}, {"PATCH", "/api/v1/admin/categories/1"}, {"DELETE", "/api/v1/admin/categories/1"},
		{"POST", "/api/v1/admin/movies/1/video"}, {"POST", "/api/v1/admin/episodes/1/video"}, {"POST", "/api/v1/admin/titles/1/thumbnail"}, {"POST", "/api/v1/admin/episodes/1/thumbnail"}, {"POST", "/api/v1/admin/titles/1/publish"}, {"GET", "/api/v1/admin/assets/" + uuid.NewString()},
	}
	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			requireStatus(t, route.method, route.path, anonymous.do(route.method, route.path, nil, nil), http.StatusUnauthorized)
			requireStatus(t, route.method, route.path, normal.API.do(route.method, route.path, nil, nil), http.StatusForbidden)
		})
	}
}

func TestEpisodeArtworkUploadPatchAndStreaming(t *testing.T) {
	s := newSuite(t)
	admin := s.admin(t)
	var titleID, episodeID, episodeNumber, duration int
	var title, description string
	var previousThumbnail *string
	err := s.db.QueryRow(context.Background(), `select t.id_title,e.id_episode,e.episode_number,e.title,coalesce(e.description,''),coalesce(e.duration,0),e.thumbnail_url
		from titles t join series sr on sr.id_title=t.id_title join seasons se on se.id_series=sr.id_series join episodes e on e.id_season=se.id_season join video_assets va on va.id_episode=e.id_episode and va.status='ready'
		where t.published and t.deleted_at is null and sr.deleted_at is null and se.deleted_at is null and e.deleted_at is null order by t.id_title,se.season_number,e.episode_number limit 1`).Scan(&titleID, &episodeID, &episodeNumber, &title, &description, &duration, &previousThumbnail)
	if err != nil {
		t.Fatalf("find seeded episode: %v", err)
	}
	imageBytes := append([]byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}, bytes.Repeat([]byte{0x5a}, 600)...)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "episode-still.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(imageBytes); err != nil {
		t.Fatal(err)
	}
	if err = mw.Close(); err != nil {
		t.Fatal(err)
	}
	uploadPath := "/api/v1/admin/episodes/" + strconv.Itoa(episodeID) + "/thumbnail"
	upload := admin.do(http.MethodPost, uploadPath, body.Bytes(), map[string]string{"Content-Type": mw.FormDataContentType()})
	requireStatus(t, http.MethodPost, uploadPath, upload, http.StatusOK)
	uploaded := decodeBody[struct {
		Thumbnail string `json:"thumbnail_url"`
	}](t, upload)
	if !strings.HasPrefix(uploaded.Thumbnail, "/api/v1/stream/thumbnails/episode-") {
		t.Fatalf("unexpected thumbnail URL %q", uploaded.Thumbnail)
	}
	t.Cleanup(func() {
		_, _ = s.db.Exec(context.Background(), `update episodes set thumbnail_url=$2 where id_episode=$1`, episodeID, previousThumbnail)
		cleanup := exec.Command("docker", "compose", "exec", "-T", "worker", "rm", "-f", "/media/thumbnails/"+filepath.Base(uploaded.Thumbnail))
		cleanup.Dir = filepath.Join("..", "..")
		_ = cleanup.Run()
	})

	assertDetailThumbnail := func(want string) {
		detailPath := "/api/v1/series/" + strconv.Itoa(titleID)
		detailResponse := admin.do(http.MethodGet, detailPath, nil, nil)
		requireStatus(t, http.MethodGet, detailPath, detailResponse, http.StatusOK)
		detail := decodeBody[struct {
			Seasons []struct {
				Episodes []struct {
					ID        int    `json:"id"`
					Thumbnail string `json:"thumbnail_url"`
				} `json:"episodes"`
			} `json:"seasons"`
		}](t, detailResponse)
		for _, season := range detail.Seasons {
			for _, episode := range season.Episodes {
				if episode.ID == episodeID {
					if episode.Thumbnail != want {
						t.Fatalf("series detail thumbnail=%q want %q", episode.Thumbnail, want)
					}
					return
				}
			}
		}
		t.Fatalf("episode %d absent from series detail", episodeID)
	}
	assertDetailThumbnail(uploaded.Thumbnail)
	served := admin.do(http.MethodGet, uploaded.Thumbnail, nil, nil)
	requireStatus(t, http.MethodGet, uploaded.Thumbnail, served, http.StatusOK)
	if served.Header.Get("Content-Type") != "image/jpeg" || !bytes.Equal(served.Body, imageBytes) {
		t.Fatalf("streamed artwork content-type=%q bytes=%d want image/jpeg bytes=%d", served.Header.Get("Content-Type"), len(served.Body), len(imageBytes))
	}

	patchPath := "/api/v1/admin/episodes/" + strconv.Itoa(episodeID)
	patch := func(thumbnail *string) {
		body := map[string]any{"episode_number": episodeNumber, "title": title, "description": description, "duration": duration}
		if thumbnail != nil {
			body["thumbnail_url"] = *thumbnail
		}
		response := admin.do(http.MethodPatch, patchPath, body, nil)
		requireStatus(t, http.MethodPatch, patchPath, response, http.StatusNoContent)
	}
	patch(nil)
	assertDetailThumbnail(uploaded.Thumbnail)
	empty := ""
	patch(&empty)
	assertDetailThumbnail("")
	patch(&uploaded.Thumbnail)
	assertDetailThumbnail(uploaded.Thumbnail)
}

type plan struct {
	ID    int   `json:"id"`
	Price int64 `json:"price"`
}
type payment struct {
	ID, Reference, Status                   string
	Amount, Subtotal, DiscountAmount, Total int64
}

func (p *payment) UnmarshalJSON(data []byte) error {
	var v struct {
		ID        string  `json:"id"`
		Reference *string `json:"reference"`
		Status    string  `json:"status"`
		Amount    int64   `json:"amount"`
		Subtotal  int64   `json:"subtotal"`
		Discount  int64   `json:"discount_amount"`
		Total     int64   `json:"total"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	p.ID = v.ID
	p.Status = v.Status
	p.Amount = v.Amount
	p.Subtotal = v.Subtotal
	p.DiscountAmount = v.Discount
	p.Total = v.Total
	if v.Reference != nil {
		p.Reference = *v.Reference
	}
	return nil
}

func firstPlan(t *testing.T, a *apiClient) plan {
	r := a.do("GET", "/api/v1/plans", nil, nil)
	requireStatus(t, "GET", "/api/v1/plans", r, 200)
	plans := decodeBody[[]plan](t, r)
	if len(plans) == 0 {
		t.Fatal("no active plans")
	}
	return plans[0]
}

func (s *suite) discount(t *testing.T, kind string, value string, max *int, active bool, starts, expires any) (int, string) {
	t.Helper()
	code := "INT-" + strings.ToUpper(suffix())
	var id int
	err := s.db.QueryRow(context.Background(), `insert into discounts(code,kind,value,max_redemptions,active,starts_at,expires_at) values($1,$2,$3,$4,$5,$6,$7) returning id`, code, kind, value, max, active, starts, expires).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		tx, err := s.db.Begin(ctx)
		if err != nil {
			return
		}
		defer tx.Rollback(ctx)
		_, _ = tx.Exec(ctx, `delete from discount_redemptions where discount_id=$1`, id)
		_, _ = tx.Exec(ctx, `delete from payments where discount_id=$1`, id)
		_, _ = tx.Exec(ctx, `delete from discounts where id=$1`, id)
		_ = tx.Commit(ctx)
	})
	return id, code
}

func TestPaymentsAndDiscounts(t *testing.T) {
	s := newSuite(t)
	u := s.arrangedUser(t, "payments")
	other := s.arrangedUser(t, "payments-idor")
	p := firstPlan(t, u.API)
	t.Run("server authoritative price and no PAN", func(t *testing.T) {
		body := map[string]any{"plan_id": p.ID, "price": 1, "subtotal": 1, "total": 1, "discount_amount": 999999, "card": map[string]string{"number": "4242424242424242", "exp": "12/40", "cvv": "123", "name": "Integration"}}
		r := u.API.do("POST", "/api/v1/payments/card", body, nil)
		requireStatus(t, "POST", "/api/v1/payments/card", r, 201)
		pay := decodeBody[payment](t, r)
		var subtotal, total, discount, stored string
		var last4 *string
		if err := s.db.QueryRow(context.Background(), `select subtotal::text,total::text,discount_amount::text,card_last4,row_to_json(payments)::text from payments where id=$1`, pay.ID).Scan(&subtotal, &total, &discount, &last4, &stored); err != nil {
			t.Fatal(err)
		}
		if subtotal != fmt.Sprintf("%.2f", float64(p.Price)/100) || total != subtotal || discount != "0.00" {
			t.Fatalf("stored authoritative amounts subtotal=%s total=%s discount=%s backend_price=%d", subtotal, total, discount, p.Price)
		}
		if last4 == nil || *last4 != "4242" || strings.Contains(stored, "4242424242424242") || strings.Contains(stored, "\"cvv\"") {
			t.Fatalf("unsafe card persistence: last4=%v row=%s", last4, stored)
		}
	})
	t.Run("validate is preview only", func(t *testing.T) {
		id, code := s.discount(t, "percent", "10.00", nil, true, nil, nil)
		r := u.API.do("POST", "/api/v1/discounts/validate", map[string]any{"plan_id": p.ID, "code": code}, nil)
		requireStatus(t, "POST", "/api/v1/discounts/validate", r, 200)
		var count int
		s.db.QueryRow(context.Background(), `select count(*) from discount_redemptions where discount_id=$1`, id).Scan(&count)
		if count != 0 {
			t.Fatalf("validation created %d redemptions", count)
		}
	})
	t.Run("single use concurrent", func(t *testing.T) {
		one := 1
		id, code := s.discount(t, "fixed", "10.00", &one, true, nil, nil)
		start := make(chan struct{})
		results := make(chan response, 2)
		for _, who := range []*apiClient{u.API, other.API} {
			go func(a *apiClient) {
				<-start
				results <- a.do("POST", "/api/v1/payments/oxxo", map[string]any{"plan_id": p.ID, "code": code}, nil)
			}(who)
		}
		close(start)
		statuses := map[int]int{}
		for i := 0; i < 2; i++ {
			r := <-results
			statuses[r.Status]++
		}
		if statuses[201] != 1 || statuses[409] != 1 {
			t.Fatalf("concurrent discount statuses=%v, want one 201 and one 409", statuses)
		}
		var count, redeemed int
		s.db.QueryRow(context.Background(), `select count(*) from discount_redemptions where discount_id=$1`, id).Scan(&count)
		s.db.QueryRow(context.Background(), `select redemption_count from discounts where id=$1`, id).Scan(&redeemed)
		if count != 1 || redeemed != 1 {
			t.Fatalf("redemption rows=%d counter=%d, want 1/1", count, redeemed)
		}
	})
	t.Run("invalid code errors distinguishable", func(t *testing.T) {
		one := 1
		_, expired := s.discount(t, "fixed", "1.00", nil, true, nil, time.Now().Add(-time.Hour))
		_, inactive := s.discount(t, "fixed", "1.00", nil, false, nil, nil)
		id, exhausted := s.discount(t, "fixed", "1.00", &one, true, nil, nil)
		_, err := s.db.Exec(context.Background(), `update discounts set redemption_count=1 where id=$1`, id)
		if err != nil {
			t.Fatal(err)
		}
		cases := []string{expired, inactive, exhausted, "UNKNOWN-" + suffix()}
		seen := map[string]bool{}
		for _, code := range cases {
			r := u.API.do("POST", "/api/v1/payments/oxxo", map[string]any{"plan_id": p.ID, "code": code}, nil)
			if r.Status < 400 {
				t.Fatalf("code %s unexpectedly accepted: %d %s", code, r.Status, r.Body)
			}
			seen[string(r.Body)] = true
		}
		if len(seen) != len(cases) {
			t.Fatalf("invalid discount errors are not distinguishable: responses=%v", seen)
		}
	})
	t.Run("OXXO lifecycle and IDOR", func(t *testing.T) {
		r := u.API.do("POST", "/api/v1/payments/oxxo", map[string]any{"plan_id": p.ID}, nil)
		requireStatus(t, "POST", "/api/v1/payments/oxxo", r, 201)
		pay := decodeBody[payment](t, r)
		if pay.Reference == "" || pay.Amount != p.Price {
			t.Fatalf("bad OXXO response: %+v plan=%+v", pay, p)
		}
		path := "/api/v1/payments/oxxo/" + pay.Reference + "/simulate-payment"
		requireStatus(t, "POST", path, other.API.do("POST", path, nil, nil), 404)
		requireStatus(t, "POST", path, u.API.do("POST", path, nil, nil), 200)
		var status string
		s.db.QueryRow(context.Background(), `select status from subscriptions where user_id=$1`, u.ID).Scan(&status)
		if status != "active" {
			t.Fatalf("subscription status=%q, want active", status)
		}
	})
	t.Run("out of range plan id", func(t *testing.T) {
		r := u.API.do("POST", "/api/v1/payments/oxxo", `{"plan_id":9223372036854775807}`, nil)
		requireStatus(t, "POST", "/api/v1/payments/oxxo", r, http.StatusBadRequest)
	})
	t.Run("database rejects invalid discount definitions", func(t *testing.T) {
		for _, value := range []string{"-10.00", "150.00"} {
			_, err := s.db.Exec(context.Background(), `insert into discounts(code,kind,value) values($1,'percent',$2)`, "INVALID-"+suffix(), value)
			if err == nil {
				t.Fatalf("database accepted invalid percent discount %s", value)
			}
		}
	})
}

func TestStreaming(t *testing.T) {
	s := newSuite(t)
	u := s.arrangedUser(t, "stream")
	var asset string
	err := s.db.QueryRow(context.Background(), `select va.id::text from video_assets va join movies m on m.id_movie=va.id_movie join titles t on t.id_title=m.id_title where va.status='ready' and va.duration_seconds>100 and t.published and va.qualities<> '[]' order by va.created_at limit 1`).Scan(&asset)
	if err != nil {
		t.Fatalf("find playable seed asset: %v", err)
	}
	masterPath := "/api/v1/stream/" + asset + "/master.m3u8"
	master := u.API.do("GET", masterPath, nil, nil)
	requireStatus(t, "GET", masterPath, master, 200)
	reVariant := regexp.MustCompile(`(?m)^([^#\r\n]+/playlist\.m3u8)\r?$`)
	m := reVariant.FindSubmatch(master.Body)
	if len(m) != 2 {
		t.Fatalf("master has no rendition: %s", master.Body)
	}
	playlistPath := "/api/v1/stream/" + asset + "/" + string(m[1])
	playlist := u.API.do("GET", playlistPath, nil, nil)
	requireStatus(t, "GET", playlistPath, playlist, 200)
	reSegment := regexp.MustCompile(`(?m)^(seg_\d{5}\.ts)\r?$`)
	segments := reSegment.FindAllSubmatch(playlist.Body, -1)
	if len(segments) < 3 {
		t.Fatalf("playlist has %d segments, need a mid-playlist segment: %s", len(segments), playlist.Body)
	}
	seg := segments[len(segments)/2]
	segmentPath := strings.TrimSuffix(playlistPath, "playlist.m3u8") + string(seg[1])
	full := u.API.do("GET", segmentPath, nil, nil)
	requireStatus(t, "GET", segmentPath, full, 200)
	if !strings.Contains(full.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("segment cache header=%q", full.Header.Get("Cache-Control"))
	}
	partial := u.API.do("GET", segmentPath, nil, map[string]string{"Range": "bytes=0-9"})
	requireStatus(t, "GET", segmentPath, partial, 206)
	wantRange := fmt.Sprintf("bytes 0-9/%d", len(full.Body))
	if partial.Header.Get("Content-Range") != wantRange || len(partial.Body) != 10 {
		t.Fatalf("range Content-Range=%q bytes=%d want %q/10", partial.Header.Get("Content-Range"), len(partial.Body), wantRange)
	}
	t.Logf("mid-playlist segment %s (%d of %d): status=206 Content-Range=%s bytes=%d", seg[1], len(segments)/2+1, len(segments), partial.Header.Get("Content-Range"), len(partial.Body))
	hidden := s.movieFixture(t, false, "ready")
	for _, path := range []string{"/api/v1/stream/" + hidden.AssetID + "/master.m3u8", "/api/v1/stream/" + hidden.AssetID + "/144p/playlist.m3u8", "/api/v1/stream/" + hidden.AssetID + "/144p/seg_00000.ts"} {
		requireStatus(t, "GET", path, u.API.do("GET", path, nil, nil), 404)
		requireStatus(t, "GET", path, newAPI(t, s).do("GET", path, nil, nil), 401)
	}
}

func TestWatchProgressFavoritesAndIDOR(t *testing.T) {
	s := newSuite(t)
	a := s.arrangedUser(t, "user-a")
	b := s.arrangedUser(t, "user-b")
	f := s.movieFixture(t, true, "ready")
	progressPath := "/api/v1/progress/movie/" + strconv.Itoa(f.MovieID)
	requireStatus(t, "PUT", progressPath, a.API.do("PUT", progressPath, map[string]int{"current_time_seconds": 37}, nil), 200)
	get := a.API.do("GET", progressPath, nil, nil)
	requireStatus(t, "GET", progressPath, get, 200)
	if !bytes.Contains(get.Body, []byte(`"current_time_seconds":37`)) {
		t.Fatalf("progress round-trip: %s", get.Body)
	}
	requireStatus(t, "GET", progressPath, b.API.do("GET", progressPath, nil, nil), 404)
	requireStatus(t, "PUT", progressPath, b.API.do("PUT", progressPath, map[string]int{"current_time_seconds": 99}, nil), 200)
	var aTime, bTime int
	s.db.QueryRow(context.Background(), `select current_time_seconds from watch_progress where user_id=$1 and id_movie=$2`, a.ID, f.MovieID).Scan(&aTime)
	s.db.QueryRow(context.Background(), `select current_time_seconds from watch_progress where user_id=$1 and id_movie=$2`, b.ID, f.MovieID).Scan(&bTime)
	if aTime != 37 || bTime != 99 {
		t.Fatalf("IDOR progress values A=%d B=%d", aTime, bTime)
	}
	requireStatus(t, "POST", "/api/v1/favorites", a.API.do("POST", "/api/v1/favorites", map[string]int{"title_id": f.TitleID}, nil), 201)
	list := a.API.do("GET", "/api/v1/favorites", nil, nil)
	requireStatus(t, "GET", "/api/v1/favorites", list, 200)
	if !bytes.Contains(list.Body, []byte(f.Name)) {
		t.Fatalf("favorite missing: %s", list.Body)
	}
	deletePath := "/api/v1/favorites/" + strconv.Itoa(f.TitleID)
	requireStatus(t, "DELETE", deletePath, b.API.do("DELETE", deletePath, nil, nil), 404)
	requireStatus(t, "DELETE", deletePath, a.API.do("DELETE", deletePath, nil, nil), 204)
	clamped := a.API.do("PUT", progressPath, map[string]int{"current_time_seconds": 999999}, nil)
	requireStatus(t, "PUT", progressPath, clamped, http.StatusOK)
	if !bytes.Contains(clamped.Body, []byte(`"current_time_seconds":120`)) {
		t.Fatalf("progress was not clamped to duration: %s", clamped.Body)
	}
	hidden := s.movieFixture(t, false, "ready")
	hiddenPath := "/api/v1/progress/movie/" + strconv.Itoa(hidden.MovieID)
	requireStatus(t, "PUT", hiddenPath, a.API.do("PUT", hiddenPath, map[string]int{"current_time_seconds": 30}, nil), http.StatusNotFound)
	requireStatus(t, "PUT", "/api/v1/progress/movie/2147483647", a.API.do("PUT", "/api/v1/progress/movie/2147483647", map[string]int{"current_time_seconds": 30}, nil), http.StatusNotFound)
	requireStatus(t, "PUT", "/api/v1/progress/movie/9223372036854775807", a.API.do("PUT", "/api/v1/progress/movie/9223372036854775807", map[string]int{"current_time_seconds": 30}, nil), http.StatusBadRequest)
}

func TestProfileLimits(t *testing.T) {
	s := newSuite(t)
	u := s.arrangedUser(t, "profiles")
	requireStatus(t, "POST", "/api/v1/profiles", u.API.do("POST", "/api/v1/profiles", map[string]string{"name": strings.Repeat("é", 51)}, nil), http.StatusBadRequest)
	start := make(chan struct{})
	statuses := make(chan int, 6)
	for i := 0; i < 6; i++ {
		go func(i int) {
			<-start
			statuses <- u.API.do("POST", "/api/v1/profiles", map[string]string{"name": fmt.Sprintf("Profile %d", i)}, nil).Status
		}(i)
	}
	close(start)
	counts := map[int]int{}
	for i := 0; i < 6; i++ {
		counts[<-statuses]++
	}
	if counts[http.StatusCreated] != 5 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent profile statuses=%v, want five 201 and one 409", counts)
	}
	if _, err := s.db.Exec(context.Background(), `insert into profiles(user_id,name) values($1,$2)`, u.ID, strings.Repeat("x", 51)); err == nil {
		t.Fatal("database accepted a 51-character profile name")
	}
}

func TestUploadTranscodeReady(t *testing.T) {
	if testing.Short() {
		t.Skip("slow upload/transcode test")
	}
	s := newSuite(t)
	admin := s.admin(t)
	normal := s.arrangedUser(t, "upload")
	create := admin.do("POST", "/api/v1/admin/movies", map[string]any{"title": "integration-upload-" + suffix(), "duration": 1}, nil)
	requireStatus(t, "POST", "/api/v1/admin/movies", create, 201)
	created := decodeBody[struct {
		ID      int `json:"id"`
		TitleID int `json:"title_id"`
	}](t, create)
	t.Cleanup(func() { s.cleanupTitle(created.TitleID) })
	publishPath := "/api/v1/admin/titles/" + strconv.Itoa(created.TitleID) + "/publish"
	requireStatus(t, "POST", publishPath, admin.do("POST", publishPath, map[string]bool{"published": true}, nil), http.StatusOK)
	video := os.Getenv("INTEGRATION_UPLOAD_VIDEO")
	if video == "" {
		video = filepath.Join("..", "..", "seed", "video", "video-short.mp4")
	}
	if _, err := os.Stat(video); os.IsNotExist(err) {
		// Seed video is not committed. Drop a clip into seed/video/ or point
		// INTEGRATION_UPLOAD_VIDEO at one to exercise the transcode pipeline.
		t.Skipf("upload fixture %s not present", video)
	} else if err != nil {
		t.Fatalf("upload fixture: %v", err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", "tiny.mp4")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(video)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(part, file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	mw.Close()
	path := "/api/v1/admin/movies/" + strconv.Itoa(created.ID) + "/video"
	started := time.Now()
	upload := admin.do("POST", path, body.Bytes(), map[string]string{"Content-Type": mw.FormDataContentType()})
	requireStatus(t, "POST", path, upload, 202)
	if time.Since(started) > 10*time.Second {
		t.Fatalf("upload blocked for %s", time.Since(started))
	}
	asset := decodeBody[struct {
		AssetID string `json:"asset_id"`
		Status  string `json:"status"`
	}](t, upload)
	t.Cleanup(func() {
		cleanup := exec.Command("docker", "compose", "exec", "-T", "worker", "rm", "-rf", "/media/hls/"+asset.AssetID, "/media/sources/"+asset.AssetID)
		cleanup.Dir = filepath.Join("..", "..")
		_ = cleanup.Run()
	})
	if asset.Status != "pending" && asset.Status != "processing" {
		t.Fatalf("upload status=%q body=%s", asset.Status, upload.Body)
	}
	playPath := "/api/v1/stream/" + asset.AssetID + "/master.m3u8"
	early := normal.API.do("GET", playPath, nil, nil)
	requireStatus(t, "GET", playPath, early, http.StatusNotFound)
	deadline := time.Now().Add(2 * time.Minute)
	var status string
	var qualities []string
	var sourceHeight int
	for time.Now().Before(deadline) {
		var raw []byte
		err = s.db.QueryRow(context.Background(), `select status::text,qualities,coalesce(source_height,0) from video_assets where id=$1`, asset.AssetID).Scan(&status, &raw, &sourceHeight)
		if err != nil {
			t.Fatal(err)
		}
		_ = json.Unmarshal(raw, &qualities)
		if status == "ready" || status == "failed" {
			break
		}
		time.Sleep(time.Second)
	}
	if status != "ready" {
		var workerErr *string
		s.db.QueryRow(context.Background(), `select error from video_assets where id=$1`, asset.AssetID).Scan(&workerErr)
		t.Fatalf("asset did not become ready: status=%s error=%v", status, workerErr)
	}
	for _, q := range qualities {
		h, _ := strconv.Atoi(strings.TrimSuffix(q, "p"))
		if h > sourceHeight {
			t.Fatalf("rendition %s upscales source height %d", q, sourceHeight)
		}
	}
	master := normal.API.do("GET", playPath, nil, nil)
	requireStatus(t, "GET", playPath, master, 200)
	found := []string{}
	for _, m := range regexp.MustCompile(`(?m)^([^#\r\n]+)/playlist\.m3u8\r?$`).FindAllSubmatch(master.Body, -1) {
		found = append(found, string(m[1]))
	}
	sort.Strings(found)
	sort.Strings(qualities)
	if strings.Join(found, ",") != strings.Join(qualities, ",") {
		t.Fatalf("master renditions=%v database/disk renditions=%v\n%s", found, qualities, master.Body)
	}
	disk := exec.Command("docker", "compose", "exec", "-T", "worker", "ls", "-1", "/media/hls/"+asset.AssetID)
	disk.Dir = filepath.Join("..", "..")
	diskOutput, err := disk.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect generated HLS directory: %v\n%s", err, diskOutput)
	}
	diskRenditions := []string{}
	for _, entry := range strings.Fields(string(diskOutput)) {
		if entry != "master.m3u8" {
			diskRenditions = append(diskRenditions, entry)
		}
	}
	sort.Strings(diskRenditions)
	if strings.Join(diskRenditions, ",") != strings.Join(qualities, ",") {
		t.Fatalf("disk renditions=%v database renditions=%v", diskRenditions, qualities)
	}
	for _, q := range qualities {
		p := "/api/v1/stream/" + asset.AssetID + "/" + q + "/playlist.m3u8"
		requireStatus(t, "GET", p, normal.API.do("GET", p, nil, nil), 200)
	}
}
