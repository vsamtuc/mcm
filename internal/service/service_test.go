package service

import (
	"context"
	"errors"
	"testing"

	memorystore "github.com/vsamtuc/mcm/internal/store/memory"
	"github.com/vsamtuc/mcm/pkg/auth"
	"github.com/vsamtuc/mcm/pkg/course"
	"github.com/vsamtuc/mcm/pkg/store"
)

func TestServiceCreateRequiresAuthentication(t *testing.T) {
	st := &stubStore{}
	svc := New(st)
	if _, err := svc.CreateCourse(context.Background(), course.CreateCourseInput{Code: "CS101", Title: "Intro", Term: "Fall"}); !errors.Is(err, course.ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
	if st.createCalled {
		t.Fatalf("store.CreateCourse should not be called when unauthorized")
	}
}

func TestServiceCreateRequiresPrivilegedRole(t *testing.T) {
	st := &stubStore{}
	svc := New(st)
	ctx := auth.WithUser(context.Background(), auth.User{Roles: []string{"user"}})
	if _, err := svc.CreateCourse(ctx, course.CreateCourseInput{Code: "CS101", Title: "Intro", Term: "Fall"}); !errors.Is(err, course.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if st.createCalled {
		t.Fatalf("store.CreateCourse should not be called when forbidden")
	}
}

func TestServiceCreateAllowsAdmin(t *testing.T) {
	st := &stubStore{}
	svc := New(st)
	ctx := auth.WithUser(context.Background(), auth.User{Roles: []string{"admin"}})
	created, err := svc.CreateCourse(ctx, course.CreateCourseInput{Code: "CS101", Title: "Intro", Term: "Fall"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.createCalled {
		t.Fatalf("expected store.CreateCourse to be called")
	}
	if created.Code != "CS101" || created.ID != 1 {
		t.Fatalf("unexpected course returned: %+v", created)
	}
}

func TestAddCourseInstructorByAdmin(t *testing.T) {
	st := memorystore.New()
	svc := New(st)
	ctx := auth.WithUser(context.Background(), auth.User{Roles: []string{"admin"}})
	created, err := svc.CreateCourse(ctx, course.CreateCourseInput{Code: "CS101", Title: "Intro", Term: "Fall"})
	if err != nil {
		t.Fatalf("CreateCourse() error = %v", err)
	}
	updated, err := svc.AddCourseInstructor(ctx, created.ID, course.Instructor{ProfessorID: 10, Role: "coordinator"})
	if err != nil {
		t.Fatalf("AddCourseInstructor() error = %v", err)
	}
	if len(updated.Instructors) != 1 || updated.Instructors[0].ProfessorID != 10 {
		t.Fatalf("expected instructor to be added, got %+v", updated.Instructors)
	}
}

func TestAddCourseInstructorRequiresPrimaryInstructor(t *testing.T) {
	st := memorystore.New()
	st.SeedProfessor("primary-subject", 42)
	svc := New(st)
	adminCtx := auth.WithUser(context.Background(), auth.User{Roles: []string{"admin"}})
	created, err := svc.CreateCourse(adminCtx, course.CreateCourseInput{
		Code:        "CS201",
		Title:       "Data Structures",
		Term:        "Spring",
		Instructors: []course.Instructor{{ProfessorID: 42, Role: "primary"}},
	})
	if err != nil {
		t.Fatalf("CreateCourse() error = %v", err)
	}
	primaryCtx := auth.WithUser(context.Background(), auth.User{Roles: []string{"professor"}, Subject: "primary-subject"})
	updated, err := svc.AddCourseInstructor(primaryCtx, created.ID, course.Instructor{ProfessorID: 43, Role: "assistant"})
	if err != nil {
		t.Fatalf("AddCourseInstructor() error = %v", err)
	}
	if len(updated.Instructors) != 2 {
		t.Fatalf("expected two instructors, got %d", len(updated.Instructors))
	}
}

func TestAddCourseInstructorRejectsNonPrimaryInstructor(t *testing.T) {
	st := memorystore.New()
	st.SeedProfessor("assistant-subject", 51)
	svc := New(st)
	adminCtx := auth.WithUser(context.Background(), auth.User{Roles: []string{"admin"}})
	created, err := svc.CreateCourse(adminCtx, course.CreateCourseInput{
		Code:        "CS301",
		Title:       "Algorithms",
		Term:        "Fall",
		Instructors: []course.Instructor{{ProfessorID: 51, Role: "assistant"}},
	})
	if err != nil {
		t.Fatalf("CreateCourse() error = %v", err)
	}
	assistantCtx := auth.WithUser(context.Background(), auth.User{Roles: []string{"professor"}, Subject: "assistant-subject"})
	if _, err := svc.AddCourseInstructor(assistantCtx, created.ID, course.Instructor{ProfessorID: 52, Role: "assistant"}); !errors.Is(err, course.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRemoveCourseInstructorByPrimary(t *testing.T) {
	st := memorystore.New()
	st.SeedProfessor("primary-subject", 61)
	svc := New(st)
	adminCtx := auth.WithUser(context.Background(), auth.User{Roles: []string{"admin"}})
	created, err := svc.CreateCourse(adminCtx, course.CreateCourseInput{
		Code:  "CS401",
		Title: "Operating Systems",
		Term:  "Winter",
		Instructors: []course.Instructor{
			{ProfessorID: 61, Role: "primary"},
			{ProfessorID: 62, Role: "assistant"},
		},
	})
	if err != nil {
		t.Fatalf("CreateCourse() error = %v", err)
	}
	primaryCtx := auth.WithUser(context.Background(), auth.User{Roles: []string{"professor"}, Subject: "primary-subject"})
	updated, err := svc.RemoveCourseInstructor(primaryCtx, created.ID, 62)
	if err != nil {
		t.Fatalf("RemoveCourseInstructor() error = %v", err)
	}
	if len(updated.Instructors) != 1 || updated.Instructors[0].ProfessorID != 61 {
		t.Fatalf("expected assistant to be removed, got %+v", updated.Instructors)
	}
}

type stubStore struct {
	createCalled bool
}

var _ store.Store = (*stubStore)(nil)

func (s *stubStore) ListCourses(ctx context.Context) ([]course.Course, error) {
	return nil, nil
}

func (s *stubStore) GetCourse(ctx context.Context, id int64) (course.Course, error) {
	return course.Course{}, course.ErrNotFound
}

func (s *stubStore) CreateCourse(ctx context.Context, input course.CreateCourseInput) (course.Course, error) {
	s.createCalled = true
	return course.Course{ID: 1, Code: input.Code, Title: input.Title, Term: input.Term}, nil
}

func (s *stubStore) UpdateCourse(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error) {
	return course.Course{}, nil
}

func (s *stubStore) DeleteCourse(ctx context.Context, id int64) error {
	return nil
}

func (s *stubStore) FindProfessorIDBySubject(ctx context.Context, subject string) (int64, error) {
	return 0, course.ErrNotFound
}
