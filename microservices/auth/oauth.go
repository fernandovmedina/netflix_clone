package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func randomURLToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (app *application) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		app.error(w, http.StatusServiceUnavailable, "Google login is not configured")
		return
	}
	state, err := randomURLToken()
	if err != nil {
		app.internalError(w, err)
		return
	}
	verifier, err := randomURLToken()
	if err != nil {
		app.internalError(w, err)
		return
	}
	redirectTo := getenv("FRONTEND_URL", "http://localhost:3000")
	if _, err = app.pool.Exec(r.Context(), `insert into oauth_states(state,code_verifier,redirect_to,expires_at) values($1,$2,$3,now()+interval '10 minutes')`, state, verifier, redirectTo); err != nil {
		app.internalError(w, err)
		return
	}
	sum := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id": {clientID}, "redirect_uri": {getenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback")},
		"response_type": {"code"}, "scope": {"openid email profile"}, "state": {state},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"}, "access_type": {"offline"}, "prompt": {"select_account"},
	}
	http.Redirect(w, r, "https://accounts.google.com/o/oauth2/v2/auth?"+query.Encode(), http.StatusFound)
}

func (app *application) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	state, code := r.URL.Query().Get("state"), r.URL.Query().Get("code")
	if state == "" || code == "" {
		app.error(w, http.StatusBadRequest, "missing OAuth state or code")
		return
	}
	var verifier, redirectTo string
	err := app.pool.QueryRow(r.Context(), `update oauth_states set consumed_at=now() where state=$1 and consumed_at is null and expires_at>now() returning code_verifier,coalesce(redirect_to,$2)`, state, getenv("FRONTEND_URL", "http://localhost:3000")).Scan(&verifier, &redirectTo)
	if err != nil {
		app.error(w, http.StatusBadRequest, "invalid or expired OAuth state")
		return
	}
	form := url.Values{"code": {code}, "client_id": {os.Getenv("GOOGLE_CLIENT_ID")}, "client_secret": {os.Getenv("GOOGLE_CLIENT_SECRET")}, "redirect_uri": {getenv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback")}, "grant_type": {"authorization_code"}, "code_verifier": {verifier}}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		app.internalError(w, err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := app.httpClient.Do(req)
	if err != nil {
		app.internalError(w, err)
		return
	}
	defer res.Body.Close()
	var token struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err = json.NewDecoder(res.Body).Decode(&token); err != nil || res.StatusCode != http.StatusOK || token.AccessToken == "" {
		app.internalError(w, fmt.Errorf("Google token exchange failed: status %d error %s", res.StatusCode, token.Error))
		return
	}
	userReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		app.internalError(w, err)
		return
	}
	userReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	userRes, err := app.httpClient.Do(userReq)
	if err != nil {
		app.internalError(w, err)
		return
	}
	defer userRes.Body.Close()
	var profile struct {
		Sub      string `json:"sub"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Verified bool   `json:"email_verified"`
	}
	if err = json.NewDecoder(userRes.Body).Decode(&profile); err != nil || userRes.StatusCode != http.StatusOK || profile.Sub == "" || profile.Email == "" {
		app.internalError(w, fmt.Errorf("Google userinfo failed: status %d", userRes.StatusCode))
		return
	}
	var user User
	err = app.pool.QueryRow(r.Context(), `insert into users(email,name,google_sub,email_verified) values($1,$2,$3,$4) on conflict(email) do update set name=case when users.name='' then excluded.name else users.name end,google_sub=coalesce(users.google_sub,excluded.google_sub),email_verified=users.email_verified or excluded.email_verified,updated_at=now() returning id::text,name,email::text,role::text`, strings.ToLower(profile.Email), profile.Name, profile.Sub, profile.Verified).Scan(&user.ID, &user.Name, &user.Email, &user.Role)
	if err != nil {
		app.internalError(w, err)
		return
	}
	session, err := app.repo.login(r.Context(), user, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		app.internalError(w, err)
		return
	}
	app.setCookies(w, session)
	http.Redirect(w, r, redirectTo, http.StatusFound)
}
