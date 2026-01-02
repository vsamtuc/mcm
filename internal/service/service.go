package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vsamtuc/mcm/pkg/application"
	"github.com/vsamtuc/mcm/pkg/auth"
	"github.com/vsamtuc/mcm/pkg/course"
	"github.com/vsamtuc/mcm/pkg/store"
)

// Service orchestrates application use cases on top of the persistence layer.
type Service struct {
	store store.Store
}

// New creates a Service backed by the provided store implementation.
func New(store store.Store) *Service {
	if store == nil {
		panic("store is nil")
	}
	return &Service{store: store}
}

// Return error if current user is not an admin.
func requireAdmin(ctx context.Context) error {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return course.ErrUnauthenticated
	}
	if !user.HasRole("admin") {
		return course.ErrForbidden
	}
	return nil
}

// Return a list of all courses. Any authenticated user can call this.
func (s *Service) ListCourses(ctx context.Context) ([]course.Course, error) {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, course.ErrUnauthenticated
	}
	_ = user // currently not used, but may be in future for filtering
	return s.store.ListCourses(ctx)
}

// Return a list of all Professors. Requires admin role.
func (s *Service) ListProfessors(ctx context.Context) ([]course.Professor, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	return s.store.ListProfessors(ctx)
}

// Return the courses instructed by the given professor. Requires admin role, or
// professor themselves.
func (s *Service) ProfessorCourses(ctx context.Context, professorID int64) ([]course.Course, error) {
	if err := requreAdminOrProfessorSelf(ctx, s.store, professorID); err != nil {
		return nil, err
	}
	return s.store.ProfessorCourses(ctx, professorID)
}

// GetCourse retrieves a course by its ID. Any authenticated user can call this.
func (s *Service) GetCourse(ctx context.Context, id int64) (course.Course, error) {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return course.Course{}, course.ErrUnauthenticated
	}
	_ = user // currently not used, but may be in future for filtering
	return s.store.GetCourse(ctx, id)
}

// CreateCourse creates a new course. Requires admin or professor role.
func (s *Service) CreateCourse(ctx context.Context, input course.CreateCourseInput) (course.Course, error) {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return course.Course{}, course.ErrUnauthenticated
	}
	if !user.HasAnyRole("admin", "professor") {
		return course.Course{}, course.ErrForbidden
	}
	return s.store.CreateCourse(ctx, input)
}

func (s *Service) UpdateCourse(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error) {
	return s.store.UpdateCourse(ctx, id, input)
}

// DeleteCourse deletes a course by its ID. Requires admin.
func (s *Service) DeleteCourse(ctx context.Context, id int64) error {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return course.ErrUnauthenticated
	}
	if !user.HasRole("admin") {
		return course.ErrForbidden
	}
	// For later: check for resources; TODO: cascade delete?
	return s.store.DeleteCourse(ctx, id)
}

func (s *Service) SetCourseActive(ctx context.Context, id int64) (course.Course, error) {
	if err := s.requireInstructorOrAdmin(ctx); err != nil {
		return course.Course{}, err
	}
	return s.store.UpdateCourseActive(ctx, id, true)
}

func (s *Service) SetCourseInactive(ctx context.Context, id int64) (course.Course, error) {
	if err := s.requireInstructorOrAdmin(ctx); err != nil {
		return course.Course{}, err
	}
	return s.store.UpdateCourseActive(ctx, id, false)
}

func (s *Service) AddCourseInstructor(ctx context.Context, courseID int64, instructor course.Instructor) (course.Course, error) {
	if instructor.ProfessorID <= 0 {
		return course.Course{}, fmt.Errorf("professor_id must be positive")
	}
	normalized := course.Instructor{ProfessorID: instructor.ProfessorID, Role: normalizeInstructorRole(instructor.Role)}
	c, err := s.store.GetCourse(ctx, courseID)
	if err != nil {
		return course.Course{}, err
	}
	if err := s.ensureInstructorAccess(ctx, c); err != nil {
		return course.Course{}, err
	}
	for _, inst := range c.Instructors {
		if inst.ProfessorID == normalized.ProfessorID {
			return course.Course{}, fmt.Errorf("instructor already assigned")
		}
	}
	updated := append(cloneInstructors(c.Instructors), normalized)
	return s.store.UpdateCourse(ctx, courseID, course.UpdateCourseInput{Instructors: &updated})
}

func (s *Service) RemoveCourseInstructor(ctx context.Context, courseID int64, professorID int64) (course.Course, error) {
	if professorID <= 0 {
		return course.Course{}, fmt.Errorf("professor_id must be positive")
	}
	c, err := s.store.GetCourse(ctx, courseID)
	if err != nil {
		return course.Course{}, err
	}
	if err := s.ensureInstructorAccess(ctx, c); err != nil {
		return course.Course{}, err
	}
	filtered, removed := removeInstructorByID(c.Instructors, professorID)
	if !removed {
		return course.Course{}, fmt.Errorf("instructor not assigned")
	}
	return s.store.UpdateCourse(ctx, courseID, course.UpdateCourseInput{Instructors: &filtered})
}

func (s *Service) ensureInstructorAccess(ctx context.Context, cr course.Course) error {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return course.ErrUnauthenticated
	}
	if user.HasRole("admin") {
		return nil
	}
	if strings.TrimSpace(user.Subject) == "" {
		return course.ErrForbidden
	}
	profID, err := s.store.FindProfessorIDBySubject(ctx, user.Subject)
	if err != nil {
		if errors.Is(err, course.ErrNotFound) {
			return course.ErrForbidden
		}
		return err
	}
	if !isPrimaryInstructor(cr.Instructors, profID) {
		return course.ErrForbidden
	}
	return nil
}

func (s *Service) requireInstructorOrAdmin(ctx context.Context) error {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return course.ErrUnauthenticated
	}
	if user.HasAnyRole("admin", "professor") {
		return nil
	}
	return course.ErrForbidden
}

func isPrimaryInstructor(instructors []course.Instructor, professorID int64) bool {
	for _, inst := range instructors {
		if inst.ProfessorID == professorID && strings.EqualFold(strings.TrimSpace(inst.Role), "primary") {
			return true
		}
	}
	return false
}

func cloneInstructors(in []course.Instructor) []course.Instructor {
	if len(in) == 0 {
		return nil
	}
	out := make([]course.Instructor, len(in))
	copy(out, in)
	return out
}

func requreAdminOrProfessorSelf(ctx context.Context, st store.Store, professorID int64) error {
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return course.ErrUnauthenticated
	}
	if !user.HasRole("admin") {
		if strings.TrimSpace(user.Subject) == "" {
			return course.ErrForbidden
		}
		profID, err := st.FindProfessorIDBySubject(ctx, user.Subject)
		if err != nil {
			if errors.Is(err, course.ErrNotFound) {
				return course.ErrForbidden
			}
			return err
		}
		if profID != professorID {
			return course.ErrForbidden
		}
	}
	return nil
}

func removeInstructorByID(in []course.Instructor, professorID int64) ([]course.Instructor, bool) {
	if len(in) == 0 {
		return nil, false
	}
	result := make([]course.Instructor, 0, len(in))
	removed := false
	for _, inst := range in {
		if inst.ProfessorID == professorID {
			removed = true
			continue
		}
		result = append(result, inst)
	}
	return result, removed
}

func normalizeInstructorRole(role string) string {
	trimmed := strings.TrimSpace(role)
	if trimmed == "" {
		return "primary"
	}
	return trimmed
}

func (s *Service) ListStudents(ctx context.Context) ([]course.Student, error) {
	return s.store.ListStudents(ctx)
}

func (s *Service) StudentCourses(ctx context.Context, studentID int64) ([]course.Course, error) {
	return s.store.StudentCourses(ctx, studentID)
}

func (s *Service) EnrollStudentInCourse(ctx context.Context, courseID int64, studentID int64) (course.Course, error) {
	if err := s.requireInstructorOrAdmin(ctx); err != nil {
		return course.Course{}, err
	}
	if err := s.store.EnrollStudent(ctx, courseID, studentID); err != nil {
		return course.Course{}, err
	}
	return s.store.GetCourse(ctx, courseID)
}

func (s *Service) UnenrollStudentFromCourse(ctx context.Context, courseID int64, studentID int64) (course.Course, error) {
	if err := s.requireInstructorOrAdmin(ctx); err != nil {
		return course.Course{}, err
	}
	if err := s.store.UnenrollStudent(ctx, courseID, studentID); err != nil {
		return course.Course{}, err
	}
	return s.store.GetCourse(ctx, courseID)
}

func (s *Service) ListTeams(ctx context.Context, courseID int64) ([]course.Team, error) {
	return s.store.ListTeams(ctx, courseID)
}

func (s *Service) CreateTeam(ctx context.Context, input course.CreateTeamInput) (course.Team, error) {
	if err := s.requireInstructorOrAdmin(ctx); err != nil {
		return course.Team{}, err
	}
	return s.store.CreateTeam(ctx, input)
}

func (s *Service) UpdateTeam(ctx context.Context, teamID int64, input course.UpdateTeamInput) (course.Team, error) {
	if err := s.requireInstructorOrAdmin(ctx); err != nil {
		return course.Team{}, err
	}
	return s.store.UpdateTeam(ctx, teamID, input)
}

func (s *Service) DeleteTeam(ctx context.Context, teamID int64) error {
	if err := s.requireInstructorOrAdmin(ctx); err != nil {
		return err
	}
	return s.store.DeleteTeam(ctx, teamID)
}

func (s *Service) TeamAddMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error) {
	if err := s.requireInstructorOrAdmin(ctx); err != nil {
		return course.Team{}, err
	}
	return s.store.AddTeamMember(ctx, teamID, studentID)
}

func (s *Service) TeamRemoveMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error) {
	if err := s.requireInstructorOrAdmin(ctx); err != nil {
		return course.Team{}, err
	}
	return s.store.RemoveTeamMember(ctx, teamID, studentID)
}

var _ application.Service = (*Service)(nil)
