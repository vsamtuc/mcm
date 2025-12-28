package http

import (
	"net/http/httptest"
	"testing"

	"github.com/vsamtuc/mcm/pkg/auth"
)

func TestUserDisplayName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		user auth.User
		want string
	}{
		{name: "username", user: auth.User{Username: "mcm-admin", Email: "admin@mcm.local"}, want: "mcm-admin"},
		{name: "email fallback", user: auth.User{Email: "student@example.com"}, want: "student@example.com"},
		{name: "subject fallback", user: auth.User{Subject: "123"}, want: "123"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := userDisplayName(tc.user); got != tc.want {
				t.Fatalf("userDisplayName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestUserInitials(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		user auth.User
		want string
	}{
		{name: "two words", user: auth.User{Username: "Ada Lovelace"}, want: "AL"},
		{name: "email", user: auth.User{Email: "user@example.com"}, want: "UC"},
		{name: "subject fallback", user: auth.User{Subject: "id"}, want: "ID"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := userInitials(tc.user); got != tc.want {
				t.Fatalf("userInitials() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAbsoluteURL(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "mcm.example")
	got := absoluteURL(r, "auth/callback")
	want := "https://mcm.example/auth/callback"
	if got != want {
		t.Fatalf("absoluteURL() = %q, want %q", got, want)
	}
}
