package jwt

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenFromRequest(t *testing.T) {
	t.Parallel()
	mw := &Middleware{cookieName: "mcm_session"}

	t.Run("authorization header", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest("GET", "http://example.org", nil)
		req.Header.Set("Authorization", "Bearer abc.def")
		token, err := mw.tokenFromRequest(req)
		if err != nil {
			t.Fatalf("tokenFromRequest() error = %v", err)
		}
		if token != "abc.def" {
			t.Fatalf("tokenFromRequest() = %q, want %q", token, "abc.def")
		}
	})

	t.Run("session cookie", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest("GET", "http://example.org", nil)
		req.AddCookie(&http.Cookie{Name: "mcm_session", Value: "cookie-token"})
		token, err := mw.tokenFromRequest(req)
		if err != nil {
			t.Fatalf("tokenFromRequest() error = %v", err)
		}
		if token != "cookie-token" {
			t.Fatalf("tokenFromRequest() = %q, want %q", token, "cookie-token")
		}
	})

	t.Run("missing token", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest("GET", "http://example.org", nil)
		if _, err := mw.tokenFromRequest(req); err == nil {
			t.Fatalf("expected error for missing token")
		}
	})
}
