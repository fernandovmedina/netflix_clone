package database

import "context"

// Signup registers a new user with Supabase Auth, storing the display name in
// the user metadata. When email confirmation is enabled the returned session
// has no access token until the user confirms their address.
func Signup(ctx context.Context, name, email, password string) (*Session, error) {
	body := map[string]any{
		"email":    email,
		"password": password,
		"data": map[string]any{
			"name": name,
		},
	}

	// GoTrue answers with a full session when auto-confirm is on, but with a
	// bare user object while email confirmation is pending — accept both.
	var res struct {
		Session
		ID           string         `json:"id"`
		Email        string         `json:"email"`
		UserMetadata map[string]any `json:"user_metadata"`
	}
	if err := gotrueRequest(ctx, "POST", "/signup", "", body, &res); err != nil {
		return nil, err
	}

	session := res.Session
	if session.User.ID == "" {
		session.User = User{ID: res.ID, Email: res.Email, UserMetadata: res.UserMetadata}
	}

	return &session, nil
}
