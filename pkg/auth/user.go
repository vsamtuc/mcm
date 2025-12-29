package auth

import (
	"context"
	"strings"
)

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

// HasRole reports whether the user carries the provided role (case-insensitive).
func (u User) HasRole(role string) bool {
	target := normalizeRole(role)
	if target == "" {
		return false
	}
	for _, r := range u.Roles {
		if normalizeRole(r) == target {
			return true
		}
	}
	return false
}

// HasAnyRole reports whether the user has at least one of the provided roles.
func (u User) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if u.HasRole(role) {
			return true
		}
	}
	return false
}

func normalizeRole(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return ""
	}
	return strings.ToLower(role)
}
