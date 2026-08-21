package authctx

import (
	"context"
	"net/http"
	"strings"
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
	for key := range r.Header {
		normalized := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
		if normalized == "x-user-id" || normalized == "x-user-email" || normalized == "x-user-role" {
			delete(r.Header, key)
		}
	}
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
