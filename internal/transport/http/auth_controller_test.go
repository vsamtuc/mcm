package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDevelAuthControllerLoginSetsCookie(t *testing.T) {
	t.Parallel()
	ctrl := newDevelAuthController(AuthConfig{SessionCookie: "dev-cookie"})
	mux := http.NewServeMux()
	ctrl.register(mux)

	req := httptest.NewRequest(http.MethodGet, LoginPath+"?sub=prof-1&roles=professor", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	var raw string
	for _, c := range rec.Result().Cookies() {
		if c.Name == "dev-cookie" {
			raw = c.Value
		}
	}
	if raw == "" {
		t.Fatalf("expected dev cookie to be set")
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse cookie: %v", err)
	}
	if vals.Get("sub") != "prof-1" || vals.Get("roles") != "professor" {
		t.Fatalf("unexpected cookie payload: %v", vals)
	}
}

func TestDevelAuthControllerPersonaPreset(t *testing.T) {
	t.Parallel()
	ctrl := newDevelAuthController(AuthConfig{SessionCookie: "dev-cookie"})
	mux := http.NewServeMux()
	ctrl.register(mux)

	req := httptest.NewRequest(http.MethodGet, LoginPath+"?persona=student", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	var raw string
	for _, c := range rec.Result().Cookies() {
		if c.Name == "dev-cookie" {
			raw = c.Value
		}
	}
	if raw == "" {
		t.Fatalf("expected dev cookie to be set")
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse cookie: %v", err)
	}
	if vals.Get("sub") != "dev-student" {
		t.Fatalf("expected student subject, got %q", vals.Get("sub"))
	}
	if roles := vals.Get("roles"); roles != "student" {
		t.Fatalf("expected student role, got %q", roles)
	}
}
