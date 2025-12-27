package auth

import "context"

// User represents an authenticated identity extracted from a JWT.
type User struct {
	Subject  string   `json:"sub"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

type ctxKey struct{}

// WithUser stores the authenticated user in the context.
func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, ctxKey{}, user)
}

// UserFrom retrieves the authenticated user from the context.
func UserFrom(ctx context.Context) (User, bool) {
	if ctx == nil {
		return User{}, false
	}
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}
