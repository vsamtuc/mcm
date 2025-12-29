package application

import (
	"context"

	"github.com/vsamtuc/mcm/pkg/course"
)

// Service exposes the application use cases to the transport layer.
type Service interface {
	ListCourses(ctx context.Context) ([]course.Course, error)
	GetCourse(ctx context.Context, id int64) (course.Course, error)
	CreateCourse(ctx context.Context, input course.CreateCourseInput) (course.Course, error)
	UpdateCourse(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error)
	DeleteCourse(ctx context.Context, id int64) error
	AddCourseInstructor(ctx context.Context, courseID int64, instructor course.Instructor) (course.Course, error)
	RemoveCourseInstructor(ctx context.Context, courseID int64, professorID int64) (course.Course, error)
}
