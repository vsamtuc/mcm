package course

import "context"

// Service captures business operations for courses, decoupled from persistence.
type Service interface {
	List(ctx context.Context) ([]Course, error)
	Get(ctx context.Context, id int64) (Course, error)
	Create(ctx context.Context, input CreateCourseInput) (Course, error)
	Update(ctx context.Context, id int64, input UpdateCourseInput) (Course, error)
	Delete(ctx context.Context, id int64) error
}
