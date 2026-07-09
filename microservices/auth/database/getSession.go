package database

import "context"

// GetSession asks Supabase Auth who the bearer of the access token is. Unlike
// the local JWT check in the middleware, this round-trips to Supabase, so it
// also catches sessions that were revoked before the token expired.
func GetSession(ctx context.Context, accessToken string) (*User, error) {
	var user User
	if err := gotrueRequest(ctx, "GET", "/user", accessToken, nil, &user); err != nil {
		return nil, err
	}

	return &user, nil
}
