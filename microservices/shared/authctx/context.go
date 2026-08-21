package authctx

import (
	"context"
	"net/http"
)

const (
	UserIDHeader = "X-User-Id"
	EmailHeader  = "X-User-Email"
	RoleHeader   = "X-User-Role"
)

type User struct {
	ID    string
	Email string
	Role  string
}

type contextKey struct{}

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

func FromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(contextKey{}).(User)
	return user, ok
}

func Strip(r *http.Request) {
	r.Header.Del(UserIDHeader)
	r.Header.Del(EmailHeader)
	r.Header.Del(RoleHeader)
}

func Inject(r *http.Request, user User) {
	Strip(r)
	r.Header.Set(UserIDHeader, user.ID)
	r.Header.Set(EmailHeader, user.Email)
	r.Header.Set(RoleHeader, user.Role)
}

func FromHeaders(r *http.Request) User {
	return User{ID: r.Header.Get(UserIDHeader), Email: r.Header.Get(EmailHeader), Role: r.Header.Get(RoleHeader)}
}
