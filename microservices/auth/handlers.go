package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/fernandovmedina/netflix-clone/microservices/auth/database"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeAuthError relays Supabase Auth failures (wrong password, duplicate
// email, ...) with their original status; anything else is a 502.
func writeAuthError(w http.ResponseWriter, err error) {
	var authErr *database.AuthError
	if errors.As(err, &authErr) {
		writeError(w, authErr.Status, authErr.Message)
		return
	}
	log.Printf("[%s] supabase auth request failed: %v", hostname, err)
	writeError(w, http.StatusBadGateway, "could not reach Supabase Auth")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "container": hostname})
}

type signupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleSignup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	session, err := database.Signup(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	session, err := database.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// handleUser returns the basic information of the logged-in user so the
// frontend can confirm the session is still alive. It re-checks the token
// against Supabase, which also catches sessions revoked before expiry.
func handleUser(w http.ResponseWriter, r *http.Request) {
	token := tokenFrom(r)

	user, err := database.GetSession(r.Context(), token)
	if err != nil {
		var authErr *database.AuthError
		if errors.As(err, &authErr) && authErr.Status == http.StatusUnauthorized {
			writeError(w, http.StatusUnauthorized, "invalid or expired session: the user is not logged in")
			return
		}
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"name":  user.Name(),
		"email": user.Email,
		"token": token,
	})
}
