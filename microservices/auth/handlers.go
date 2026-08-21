package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fernandovmedina/netflix-clone/microservices/shared/jsonx"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "container": hostname})
}

type signupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *application) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if req.Email == "" || len(req.Password) < 8 {
		app.error(w, http.StatusBadRequest, "email and a password of at least 8 characters are required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		app.internalError(w, err)
		return
	}
	session, err := app.repo.createUser(r.Context(), req.Name, req.Email, string(hash), r.UserAgent(), r.RemoteAddr)
	if err != nil {
		if errors.Is(err, errEmailExists) {
			app.error(w, http.StatusConflict, "email is already registered")
			return
		}
		app.internalError(w, err)
		return
	}
	app.setCookies(w, session)
	app.write(w, http.StatusCreated, map[string]any{"user": session.User})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *application) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		app.error(w, http.StatusBadRequest, "email and password are required")
		return
	}
	user, err := app.repo.findByEmail(r.Context(), req.Email)
	if err == pgx.ErrNoRows {
		_ = bcrypt.CompareHashAndPassword(app.dummyHash, []byte(req.Password))
		app.error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		app.internalError(w, err)
		return
	}
	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		app.error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	session, err := app.repo.login(r.Context(), user, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		app.internalError(w, err)
		return
	}
	app.setCookies(w, session)
	app.write(w, http.StatusOK, map[string]any{"user": session.User})
}

func (app *application) handleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		app.error(w, http.StatusUnauthorized, "missing refresh token")
		return
	}
	session, err := app.repo.rotate(r.Context(), cookie.Value, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		if errors.Is(err, errInvalidRefresh) || errors.Is(err, errRefreshReuse) {
			app.clearCookies(w)
			app.error(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		app.internalError(w, err)
		return
	}
	app.setCookies(w, session)
	app.write(w, http.StatusOK, map[string]any{"user": session.User})
}

func (app *application) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("refresh_token")
	if cookie != nil {
		if err := app.repo.logout(r.Context(), cookie.Value); err != nil {
			app.internalError(w, err)
			return
		}
	}
	app.clearCookies(w)
	w.WriteHeader(http.StatusNoContent)
}
func (app *application) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r)
	user, err := app.repo.userByID(r.Context(), claims.Subject)
	if err == pgx.ErrNoRows {
		app.error(w, http.StatusUnauthorized, "user no longer exists")
		return
	}
	if err != nil {
		app.internalError(w, err)
		return
	}
	user.Email = claims.Email
	user.Role = claims.Role
	app.write(w, http.StatusOK, user)
}

func (app *application) setCookies(w http.ResponseWriter, t sessionTokens) {
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: t.Access, HttpOnly: true, Secure: app.cookieSecure, SameSite: http.SameSiteLaxMode, Path: "/", MaxAge: int(app.tokens.accessTTL.Seconds())})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: t.Refresh, HttpOnly: true, Secure: app.cookieSecure, SameSite: http.SameSiteLaxMode, Path: "/api/v1/auth", MaxAge: int(app.repo.refreshTTL.Seconds())})
}
func (app *application) clearCookies(w http.ResponseWriter) {
	past := time.Unix(1, 0)
	for _, c := range []*http.Cookie{{Name: "access_token", Path: "/"}, {Name: "refresh_token", Path: "/api/v1/auth"}} {
		c.Value = ""
		c.HttpOnly = true
		c.Secure = app.cookieSecure
		c.SameSite = http.SameSiteLaxMode
		c.MaxAge = -1
		c.Expires = past
		http.SetCookie(w, c)
	}
}
func (app *application) write(w http.ResponseWriter, status int, value any) {
	jsonx.Write(w, status, value)
}
func (app *application) error(w http.ResponseWriter, status int, message string) {
	app.write(w, status, map[string]string{"error": message})
}
func (app *application) internalError(w http.ResponseWriter, err error) {
	log.Printf("[%s] auth error: %v", hostname, err)
	app.error(w, http.StatusInternalServerError, "internal server error")
}
