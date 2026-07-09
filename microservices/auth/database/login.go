package database

import "context"

// Login exchanges email/password credentials for a Supabase Auth session
// (password grant). The returned access token is the JWT every subsequent
// request must carry.
func Login(ctx context.Context, email, password string) (*Session, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}

	var session Session
	if err := gotrueRequest(ctx, "POST", "/token?grant_type=password", "", body, &session); err != nil {
		return nil, err
	}

	return &session, nil
}
