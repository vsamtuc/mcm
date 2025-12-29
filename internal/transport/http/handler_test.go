package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appservice "github.com/vsamtuc/mcm/internal/service"
	memorystore "github.com/vsamtuc/mcm/internal/store/memory"
	"github.com/vsamtuc/mcm/pkg/auth"
	"github.com/vsamtuc/mcm/pkg/course"
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

func TestIndexRedirectsWhenUnauthenticated(t *testing.T) {
	t.Parallel()
	mux, err := NewMux(func() bool { return true }, appservice.New(memorystore.New()), AuthConfig{DevMode: true})
	if err != nil {
		t.Fatalf("NewMux: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != LoginPath {
		t.Fatalf("redirect Location = %q, want %q", loc, LoginPath)
	}
}

func TestCoursesRedirectsWhenUnauthenticated(t *testing.T) {
	t.Parallel()
	mux, err := NewMux(func() bool { return true }, appservice.New(memorystore.New()), AuthConfig{DevMode: true})
	if err != nil {
		t.Fatalf("NewMux: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/courses", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != LoginPath {
		t.Fatalf("redirect Location = %q, want %q", loc, LoginPath)
	}
}

func TestCoursesPageRendersWithUser(t *testing.T) {
	t.Parallel()
	mux, err := NewMux(func() bool { return true }, appservice.New(memorystore.New()), AuthConfig{DevMode: true})
	if err != nil {
		t.Fatalf("NewMux: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/courses", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{Subject: "dev", Roles: []string{"admin"}}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Create course") {
		t.Fatalf("response missing form content")
	}
}

func TestCoursesRedirectsStudentToEnroll(t *testing.T) {
	t.Parallel()
	mux, err := NewMux(func() bool { return true }, appservice.New(memorystore.New()), AuthConfig{DevMode: true})
	if err != nil {
		t.Fatalf("NewMux: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/courses", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{Subject: "dev-student", Roles: []string{"student"}}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "/enroll" {
		t.Fatalf("redirect Location = %q, want %q", loc, "/enroll")
	}
}

func TestEnrollPageRendersForStudent(t *testing.T) {
	t.Parallel()
	mux, err := NewMux(func() bool { return true }, appservice.New(memorystore.New()), AuthConfig{DevMode: true})
	if err != nil {
		t.Fatalf("NewMux: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/enroll", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{Subject: "dev-student", Roles: []string{"student"}}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Enroll in courses") {
		t.Fatalf("response missing enrollment content")
	}
}

func TestCoursesTrailingSlashRedirects(t *testing.T) {
	t.Parallel()
	mux, err := NewMux(func() bool { return true }, appservice.New(memorystore.New()), AuthConfig{DevMode: true})
	if err != nil {
		t.Fatalf("NewMux: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/courses/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if loc := rec.Header().Get("Location"); loc != "/courses" {
		t.Fatalf("redirect Location = %q, want %q", loc, "/courses")
	}
}

func TestCourseHandlerCreateCourseRequiresAuth(t *testing.T) {
	t.Parallel()
	h, _ := newTestCourseHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/courses", strings.NewReader(`{"code":"CS101","title":"Intro","term":"Fall"}`))
	rec := httptest.NewRecorder()
	h.createCourse(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestCourseHandlerCreateCourseRequiresPrivilegedRole(t *testing.T) {
	t.Parallel()
	h, _ := newTestCourseHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/courses", strings.NewReader(`{"code":"CS101","title":"Intro","term":"Fall"}`))
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{Roles: []string{"user"}}))
	rec := httptest.NewRecorder()
	h.createCourse(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCourseHandlerCreateCourseAdminSuccess(t *testing.T) {
	t.Parallel()
	h, _ := newTestCourseHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/courses", strings.NewReader(`{"code":"CS101","title":"Intro","term":"Fall"}`))
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{Roles: []string{"ADMIN"}}))
	rec := httptest.NewRecorder()
	h.createCourse(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	var created course.Course
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == 0 || created.Code != "CS101" {
		t.Fatalf("unexpected course payload: %+v", created)
	}
}

func TestCourseHandlerAddInstructorAdmin(t *testing.T) {
	t.Parallel()
	h, store := newTestCourseHandler()
	ctx := context.Background()
	c, err := store.CreateCourse(ctx, course.CreateCourseInput{Code: "CS100", Title: "Title", Term: "Fall"})
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/courses/%d/instructors", c.ID), strings.NewReader(`{"professor_id":5,"role":"assistant"}`))
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{Roles: []string{"admin"}}))
	rec := httptest.NewRecorder()
	h.route(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var updated course.Course
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(updated.Instructors) != 1 || updated.Instructors[0].ProfessorID != 5 {
		t.Fatalf("expected instructor added, got %+v", updated.Instructors)
	}
}

func TestCourseHandlerRemoveInstructorRequiresPrimary(t *testing.T) {
	t.Parallel()
	h, store := newTestCourseHandler()
	ctx := context.Background()
	store.SeedProfessor("primary", 10)
	c, err := store.CreateCourse(ctx, course.CreateCourseInput{
		Code:        "CS200",
		Title:       "Title",
		Term:        "Fall",
		Instructors: []course.Instructor{{ProfessorID: 10, Role: "primary"}, {ProfessorID: 11, Role: "assistant"}},
	})
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/courses/%d/instructors/11", c.ID), nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{Subject: "primary", Roles: []string{"professor"}}))
	rec := httptest.NewRecorder()
	h.route(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var updated course.Course
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(updated.Instructors) != 1 || updated.Instructors[0].ProfessorID != 10 {
		t.Fatalf("expected assistant removed, got %+v", updated.Instructors)
	}
}

func newTestCourseHandler() (*courseHandler, *memorystore.Store) {
	store := memorystore.New()
	svc := appservice.New(store)
	return &courseHandler{service: svc}, store
}
