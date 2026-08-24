package main

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	maxProfilesPerUser  = 5
	maxProfileNameRunes = 50
)

type favorite struct {
	TitleID     int       `json:"title_id"`
	Title       string    `json:"title"`
	ContentType string    `json:"content_type"`
	Thumbnail   string    `json:"thumbnail_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (app *application) listFavorites(w http.ResponseWriter, r *http.Request) {
	rows, err := app.pool.Query(r.Context(), `select f.id_title,t.title,t.type::text,coalesce(t.thumbnail_url,''),f.created_at
		from favorites f join titles t on t.id_title=f.id_title where f.user_id=$1 and t.deleted_at is null order by f.created_at desc`, userID(r))
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	items := []favorite{}
	for rows.Next() {
		var item favorite
		if err = rows.Scan(&item.TitleID, &item.Title, &item.ContentType, &item.Thumbnail, &item.CreatedAt); err != nil {
			serverError(w, err)
			return
		}
		items = append(items, item)
	}
	jsonx.Write(w, http.StatusOK, items)
}

func (app *application) addFavorite(w http.ResponseWriter, r *http.Request) {
	var in struct {
		TitleID int `json:"title_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !validDatabaseID(in.TitleID) {
		jsonx.Error(w, http.StatusBadRequest, "valid title_id is required")
		return
	}
	var created time.Time
	err := app.pool.QueryRow(r.Context(), `insert into favorites(user_id,id_title) values($1,$2)
		on conflict(user_id,id_title) do update set user_id=excluded.user_id returning created_at`, userID(r), in.TitleID).Scan(&created)
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, http.StatusCreated, map[string]any{"title_id": in.TitleID, "created_at": created})
}

func (app *application) deleteFavorite(w http.ResponseWriter, r *http.Request) {
	id, ok := pathPositiveInt(w, r, "id")
	if !ok {
		return
	}
	tag, err := app.pool.Exec(r.Context(), `delete from favorites where user_id=$1 and id_title=$2`, userID(r), id)
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, http.StatusNotFound, "favorite not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type profile struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Avatar    *string   `json:"avatar"`
	IsKids    bool      `json:"is_kids"`
	CreatedAt time.Time `json:"created_at"`
}

type profileInput struct {
	Name   *string `json:"name"`
	Avatar *string `json:"avatar"`
	IsKids *bool   `json:"is_kids"`
}

func scanProfile(row pgx.Row) (profile, error) {
	var item profile
	err := row.Scan(&item.ID, &item.Name, &item.Avatar, &item.IsKids, &item.CreatedAt)
	return item, err
}

func (app *application) listProfiles(w http.ResponseWriter, r *http.Request) {
	rows, err := app.pool.Query(r.Context(), `select id,name,avatar,is_kids,created_at from profiles where user_id=$1 order by created_at,id`, userID(r))
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	items := []profile{}
	for rows.Next() {
		var item profile
		if err = rows.Scan(&item.ID, &item.Name, &item.Avatar, &item.IsKids, &item.CreatedAt); err != nil {
			serverError(w, err)
			return
		}
		items = append(items, item)
	}
	jsonx.Write(w, http.StatusOK, items)
}

func (app *application) createProfile(w http.ResponseWriter, r *http.Request) {
	var in profileInput
	if !decode(w, r, &in) {
		return
	}
	if in.Name == nil {
		jsonx.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	name := strings.TrimSpace(*in.Name)
	if name == "" {
		jsonx.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if utf8.RuneCountInString(name) > maxProfileNameRunes {
		jsonx.Error(w, http.StatusBadRequest, "profile name must be 50 characters or fewer")
		return
	}
	kids := false
	if in.IsKids != nil {
		kids = *in.IsKids
	}
	tx, err := app.pool.Begin(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var lockedUser uuid.UUID
	if err = tx.QueryRow(r.Context(), `select id from users where id=$1 for update`, userID(r)).Scan(&lockedUser); err != nil {
		serverError(w, err)
		return
	}
	var count int
	if err = tx.QueryRow(r.Context(), `select count(*) from profiles where user_id=$1`, lockedUser).Scan(&count); err != nil {
		serverError(w, err)
		return
	}
	if count >= maxProfilesPerUser {
		jsonx.Error(w, http.StatusConflict, "profile limit reached")
		return
	}
	item, err := scanProfile(tx.QueryRow(r.Context(), `insert into profiles(user_id,name,avatar,is_kids) values($1,$2,$3,$4)
		returning id,name,avatar,is_kids,created_at`, userID(r), name, cleanOptional(in.Avatar), kids))
	if err != nil {
		serverError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, http.StatusCreated, item)
}

func (app *application) getProfile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonx.Error(w, http.StatusBadRequest, "invalid profile id")
		return
	}
	item, err := scanProfile(app.pool.QueryRow(r.Context(), `select id,name,avatar,is_kids,created_at from profiles where id=$1 and user_id=$2`, id, userID(r)))
	if err == pgx.ErrNoRows {
		jsonx.Error(w, http.StatusNotFound, "profile not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, http.StatusOK, item)
}

func (app *application) patchProfile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonx.Error(w, http.StatusBadRequest, "invalid profile id")
		return
	}
	var in profileInput
	if !decode(w, r, &in) {
		return
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			jsonx.Error(w, http.StatusBadRequest, "name must not be empty")
			return
		}
		if utf8.RuneCountInString(name) > maxProfileNameRunes {
			jsonx.Error(w, http.StatusBadRequest, "profile name must be 50 characters or fewer")
			return
		}
	}
	item, err := scanProfile(app.pool.QueryRow(r.Context(), `update profiles set
		name=case when $3::boolean then $4 else name end,
		avatar=case when $5::boolean then $6 else avatar end,
		is_kids=case when $7::boolean then $8 else is_kids end
		where id=$1 and user_id=$2 returning id,name,avatar,is_kids,created_at`,
		id, userID(r), in.Name != nil, optionalString(in.Name), in.Avatar != nil, cleanOptional(in.Avatar), in.IsKids != nil, optionalBool(in.IsKids)))
	if err == pgx.ErrNoRows {
		jsonx.Error(w, http.StatusNotFound, "profile not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	jsonx.Write(w, http.StatusOK, item)
}

func (app *application) deleteProfile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		jsonx.Error(w, http.StatusBadRequest, "invalid profile id")
		return
	}
	tag, err := app.pool.Exec(r.Context(), `delete from profiles where id=$1 and user_id=$2`, id, userID(r))
	if err != nil {
		serverError(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		jsonx.Error(w, http.StatusNotFound, "profile not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func cleanOptional(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func optionalBool(value *bool) bool { return value != nil && *value }
