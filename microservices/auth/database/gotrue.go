package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var gotrueClient = &http.Client{Timeout: 10 * time.Second}

// User is the subset of the Supabase Auth user record the frontend needs.
type User struct {
	ID           string         `json:"id"`
	Email        string         `json:"email"`
	UserMetadata map[string]any `json:"user_metadata"`
}

// Name returns the display name stored in the user metadata at signup.
func (u *User) Name() string {
	if name, ok := u.UserMetadata["name"].(string); ok {
		return name
	}
	return ""
}

// Session mirrors the Supabase Auth session payload so the Next.js frontend
// receives the exact shape supabase-js produces.
type Session struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

// AuthError carries the status and message returned by Supabase Auth so
// handlers can relay them (e.g. "Invalid login credentials" with a 400).
type AuthError struct {
	Status  int
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("supabase auth: %s (status %d)", e.Message, e.Status)
}

// gotrueRequest calls the Supabase Auth REST API. A non-empty token is sent
// as the user's Bearer token, otherwise the anon key is used.
func gotrueRequest(ctx context.Context, method, path, token string, body any, out any) error {
	baseURL := os.Getenv("SUPABASE_URL")
	anonKey := os.Getenv("SUPABASE_ANON_KEY")
	if baseURL == "" || anonKey == "" {
		return fmt.Errorf("SUPABASE_URL and SUPABASE_ANON_KEY must be set")
	}

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+"/auth/v1"+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", anonKey)
	req.Header.Set("Content-Type", "application/json")
	if token == "" {
		token = anonKey
	}
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := gotrueClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		return &AuthError{Status: res.StatusCode, Message: gotrueErrorMessage(raw)}
	}

	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// gotrueErrorMessage extracts a human-readable message from the different
// error shapes GoTrue returns.
func gotrueErrorMessage(raw []byte) string {
	var payload struct {
		Msg              string `json:"msg"`
		Message          string `json:"message"`
		ErrorDescription string `json:"error_description"`
		Err              string `json:"error"`
	}
	if json.Unmarshal(raw, &payload) == nil {
		for _, msg := range []string{payload.Msg, payload.Message, payload.ErrorDescription, payload.Err} {
			if msg != "" {
				return msg
			}
		}
	}
	return string(raw)
}
