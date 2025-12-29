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

	// FindProfessorIDBySubject returns the professor ID associated with 
	// the given subject (keycloak ID).
	FindProfessorIDBySubject(ctx context.Context, subject string) (int64, error)
}
