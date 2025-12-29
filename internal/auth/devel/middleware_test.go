package devel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vsamtuc/mcm/pkg/auth"
)

func TestDevMiddlewareDefaultUser(t *testing.T) {
	t.Parallel()
	mw := New(Config{})
	var got auth.User
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFrom(r.Context())
		if !ok {
			t.Fatalf("expected user in context")
		}
		got = user
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if got.Subject == "" || !got.HasRole("admin") {
		t.Fatalf("unexpected default user: %+v", got)
	}
}

func TestDevMiddlewareCookieOverride(t *testing.T) {
	t.Parallel()
	mw := New(Config{CookieName: "cookie"})
	want := auth.User{Subject: "prof-1", Username: "Prof", Email: "prof@example.com", Roles: []string{"professor"}}
	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	req.AddCookie(&http.Cookie{Name: "cookie", Value: EncodeUser(want)})

	var got auth.User
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFrom(r.Context())
		if !ok {
			t.Fatalf("expected user in context")
		}
		got = user
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if got.Subject != want.Subject || !got.HasRole("professor") || got.Username != want.Username {
		t.Fatalf("expected override user, got %+v", got)
	}
}

func TestDevMiddlewareHeaderPrecedence(t *testing.T) {
	t.Parallel()
	mw := New(Config{CookieName: "cookie"})
	req := httptest.NewRequest(http.MethodGet, "/api/courses", nil)
	req.AddCookie(&http.Cookie{Name: "cookie", Value: EncodeUser(auth.User{Subject: "cookie-user", Roles: []string{"student"}})})
	req.Header.Set("X-MCM-Dev-User", EncodeUser(auth.User{Subject: "header-user", Roles: []string{"admin"}}))

	var got auth.User
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.UserFrom(r.Context())
		if !ok {
			t.Fatalf("expected user in context")
		}
		got = user
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if got.Subject != "header-user" || !got.HasRole("admin") {
		t.Fatalf("expected header user, got %+v", got)
	}
}
