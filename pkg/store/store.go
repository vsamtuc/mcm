package store

import (
	"context"

	"github.com/vsamtuc/mcm/pkg/course"
)

// Store abstracts persistence for all domain entities.
type Store interface {
	ListCourses(ctx context.Context) ([]course.Course, error)
	GetCourse(ctx context.Context, id int64) (course.Course, error)
	CreateCourse(ctx context.Context, input course.CreateCourseInput) (course.Course, error)
	UpdateCourse(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error)
	DeleteCourse(ctx context.Context, id int64) error
	UpdateCourseActive(ctx context.Context, id int64, active bool) (course.Course, error)

	ListProfessors(ctx context.Context) ([]course.Professor, error)
	ProfessorCourses(ctx context.Context, professorID int64) ([]course.Course, error)

	// FindProfessorIDBySubject returns the professor ID associated with 
	// the given subject (keycloak ID).
	FindProfessorIDBySubject(ctx context.Context, subject string) (int64, error)

	ListStudents(ctx context.Context) ([]course.Student, error)
	StudentCourses(ctx context.Context, studentID int64) ([]course.Course, error)
	EnrollStudent(ctx context.Context, courseID int64, studentID int64) error
	UnenrollStudent(ctx context.Context, courseID int64, studentID int64) error

	ListTeams(ctx context.Context, courseID int64) ([]course.Team, error)
	CreateTeam(ctx context.Context, input course.CreateTeamInput) (course.Team, error)
	UpdateTeam(ctx context.Context, teamID int64, input course.UpdateTeamInput) (course.Team, error)
	DeleteTeam(ctx context.Context, teamID int64) error
	AddTeamMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error)
	RemoveTeamMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error)
}
