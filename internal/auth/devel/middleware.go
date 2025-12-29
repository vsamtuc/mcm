package devel

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/vsamtuc/mcm/pkg/auth"
)

// Config controls development auth middleware behavior.
type Config struct {
	CookieName  string
	DefaultUser auth.User
	SkipPaths   []string
}

// Middleware injects a developer-configured user into the request context.
type Middleware struct {
	cookieName  string
	defaultUser auth.User
	skipPaths   map[string]struct{}
}

// New constructs the middleware using the provided configuration.
func New(cfg Config) *Middleware {
	cookie := strings.TrimSpace(cfg.CookieName)
	if cookie == "" {
		cookie = "mcm_dev_user"
	}
	defUser := cfg.DefaultUser
	if len(defUser.Roles) == 0 {
		defUser.Roles = []string{"admin"}
	}
	if defUser.Subject == "" {
		defUser.Subject = "dev-admin"
	}
	if defUser.Username == "" {
		defUser.Username = "Dev Admin"
	}
	if defUser.Email == "" {
		defUser.Email = "dev-admin@example.com"
	}
	skip := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		skip[p] = struct{}{}
	}
	return &Middleware{cookieName: cookie, defaultUser: defUser, skipPaths: skip}
}

// Wrap adds the dev user to the request context. Header X-MCM-Dev-User overrides the cookie.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.shouldSkip(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		user := m.defaultUser
		if parsed, ok := m.parseUser(r); ok {
			user = parsed
		}
		ctx := auth.WithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) parseUser(r *http.Request) (auth.User, bool) {
	if r == nil {
		return auth.User{}, false
	}
	if header := strings.TrimSpace(r.Header.Get("X-MCM-Dev-User")); header != "" {
		if u, err := decodeUser(header, m.defaultUser); err == nil {
			return u, true
		}
	}
	if m.cookieName != "" {
		if c, err := r.Cookie(m.cookieName); err == nil {
			if u, err := decodeUser(c.Value, m.defaultUser); err == nil {
				return u, true
			}
		}
	}
	return auth.User{}, false
}

func (m *Middleware) shouldSkip(path string) bool {
	if path == "" {
		return false
	}
	_, ok := m.skipPaths[path]
	return ok
}

// EncodeUser serializes the user into a query-string representation suitable for cookies or headers.
func EncodeUser(u auth.User) string {
	values := url.Values{}
	if u.Subject != "" {
		values.Set("sub", u.Subject)
	}
	if u.Username != "" {
		values.Set("username", u.Username)
	}
	if u.Email != "" {
		values.Set("email", u.Email)
	}
	if len(u.Roles) > 0 {
		values.Set("roles", strings.Join(u.Roles, ","))
	}
	return values.Encode()
}

func decodeUser(raw string, fallback auth.User) (auth.User, error) {
	vals, err := url.ParseQuery(raw)
	if err != nil {
		return fallback, err
	}
	u := fallback
	if v := first(vals, "sub", "subject"); v != "" {
		u.Subject = v
	}
	if v := first(vals, "username", "user"); v != "" {
		u.Username = v
	}
	if v := first(vals, "email"); v != "" {
		u.Email = v
	}
	if roles := vals.Get("roles"); roles != "" {
		u.Roles = splitRoles(roles)
	}
	if len(u.Roles) == 0 {
		u.Roles = fallback.Roles
	}
	return u, nil
}

func first(vals url.Values, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(vals.Get(key)); v != "" {
			return v
		}
	}
	return ""
}

func splitRoles(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
