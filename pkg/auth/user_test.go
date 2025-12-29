package auth

import "testing"

func TestUserHasRole(t *testing.T) {
	user := User{Roles: []string{"admin", "Professor"}}
	if !user.HasRole("ADMIN") {
		t.Fatalf("expected user to have role ADMIN")
	}
	if !user.HasRole(" professor ") {
		t.Fatalf("expected user to have role professor with whitespace")
	}
	if user.HasRole("student") {
		t.Fatalf("did not expect user to have role student")
	}
}

func TestUserHasAnyRole(t *testing.T) {
	user := User{Roles: []string{"user"}}
	if user.HasAnyRole() {
		t.Fatalf("empty role list should be false")
	}
	if user.HasAnyRole("admin", "professor") {
		t.Fatalf("user should not have admin or professor roles")
	}
	if !user.HasAnyRole("viewer", "user") {
		t.Fatalf("expected true when at least one role matches")
	}
}
