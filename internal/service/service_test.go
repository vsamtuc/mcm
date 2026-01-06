package service

import (
	"context"
	"errors"
	"testing"

	memorystore "github.com/vsamtuc/mcm/internal/store/memory"
	"github.com/vsamtuc/mcm/pkg/auth"
	"github.com/vsamtuc/mcm/pkg/course"
	"github.com/vsamtuc/mcm/pkg/resource"
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

func (s *stubStore) UpdateCourseActive(ctx context.Context, id int64, active bool) (course.Course, error) {
	return course.Course{}, nil
}

func (s *stubStore) ListProfessors(ctx context.Context) ([]course.Professor, error) {
	return nil, nil
}

func (s *stubStore) ProfessorCourses(ctx context.Context, professorID int64) ([]course.Course, error) {
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

func (s *stubStore) ListStudents(ctx context.Context) ([]course.Student, error) {
	return nil, nil
}

func (s *stubStore) StudentCourses(ctx context.Context, studentID int64) ([]course.Course, error) {
	return nil, nil
}

func (s *stubStore) EnrollStudent(ctx context.Context, courseID int64, studentID int64) error {
	return nil
}

func (s *stubStore) UnenrollStudent(ctx context.Context, courseID int64, studentID int64) error {
	return nil
}

func (s *stubStore) ListTeams(ctx context.Context, courseID int64) ([]course.Team, error) {
	return nil, nil
}

func (s *stubStore) CreateTeam(ctx context.Context, input course.CreateTeamInput) (course.Team, error) {
	return course.Team{}, nil
}

func (s *stubStore) UpdateTeam(ctx context.Context, teamID int64, input course.UpdateTeamInput) (course.Team, error) {
	return course.Team{}, nil
}

func (s *stubStore) DeleteTeam(ctx context.Context, teamID int64) error {
	return nil
}

func (s *stubStore) AddTeamMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error) {
	return course.Team{}, nil
}

func (s *stubStore) RemoveTeamMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error) {
	return course.Team{}, nil
}

func (s *stubStore) CreateResourceClass(ctx context.Context, rc resource.ResourceClass) (resource.ResourceClass, error) {
	return rc, nil
}

func (s *stubStore) GetResourceClass(ctx context.Context, id int64) (resource.ResourceClass, error) {
	return resource.ResourceClass{}, nil
}

func (s *stubStore) ListResourceClasses(ctx context.Context) ([]resource.ResourceClass, error) {
	return nil, nil
}

func (s *stubStore) UpdateResourceClass(ctx context.Context, rc resource.ResourceClass) error {
	return nil
}

func (s *stubStore) DeleteResourceClass(ctx context.Context, id int64) error {
	return nil
}

func (s *stubStore) CreateResourceSet(ctx context.Context, rs resource.ResourceSet) (resource.ResourceSet, error) {
	return rs, nil
}

func (s *stubStore) GetResourceSet(ctx context.Context, id int64) (resource.ResourceSet, error) {
	return resource.ResourceSet{}, nil
}

func (s *stubStore) ListResourceSetsByCourse(ctx context.Context, courseID int64) ([]resource.ResourceSet, error) {
	return nil, nil
}

func (s *stubStore) ListResourceSetsByClass(ctx context.Context, resourceClassID int64) ([]resource.ResourceSet, error) {
	return nil, nil
}

func (s *stubStore) UpdateResourceSet(ctx context.Context, rs resource.ResourceSet) error {
	return nil
}

func (s *stubStore) DeleteResourceSet(ctx context.Context, id int64) error {
	return nil
}

func (s *stubStore) CreateResource(ctx context.Context, r resource.Resource) (resource.Resource, error) {
	return r, nil
}

func (s *stubStore) GetResource(ctx context.Context, id int64) (resource.Resource, error) {
	return resource.Resource{}, nil
}

func (s *stubStore) ListResourcesBySet(ctx context.Context, resourceSetID int64) ([]resource.Resource, error) {
	return nil, nil
}

func (s *stubStore) UpdateResource(ctx context.Context, r resource.Resource) error {
	return nil
}

func (s *stubStore) DeleteResource(ctx context.Context, id int64) error {
	return nil
}
