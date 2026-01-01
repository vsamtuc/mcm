package application

import (
	"context"

	"github.com/vsamtuc/mcm/pkg/course"
)

// Service exposes the application use cases to the transport layer.
type Service interface {

	// Manage users
	ListProfessors(ctx context.Context) ([]course.Professor, error)
	ProfessorCourses(ctx context.Context, professorID int64) ([]course.Course, error)

	// Course management
	ListCourses(ctx context.Context) ([]course.Course, error)
	GetCourse(ctx context.Context, id int64) (course.Course, error)
	SetCourseActive(ctx context.Context, id int64) (course.Course, error)
	SetCourseInactive(ctx context.Context, id int64) (course.Course, error)
	CreateCourse(ctx context.Context, input course.CreateCourseInput) (course.Course, error)
	UpdateCourse(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error)
	DeleteCourse(ctx context.Context, id int64) error
	
	// Course instructor management
	AddCourseInstructor(ctx context.Context, courseID int64, instructor course.Instructor) (course.Course, error)
	RemoveCourseInstructor(ctx context.Context, courseID int64, professorID int64) (course.Course, error)

	// Students
	ListStudents(ctx context.Context) ([]course.Student, error)
	StudentCourses(ctx context.Context, studentID int64) ([]course.Course, error)
	EnrollStudentInCourse(ctx context.Context, courseID int64, studentID int64) (course.Course, error)
	UnenrollStudentFromCourse(ctx context.Context, courseID int64, studentID int64) (course.Course, error)

	// Teams
	ListTeams(ctx context.Context, courseID int64) ([]course.Team, error)
	CreateTeam(ctx context.Context, input course.CreateTeamInput) (course.Team, error)
	UpdateTeam(ctx context.Context, teamID int64, input course.UpdateTeamInput) (course.Team, error)
	DeleteTeam(ctx context.Context, teamID int64) error

	TeamAddMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error)
	TeamRemoveMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error)
}
