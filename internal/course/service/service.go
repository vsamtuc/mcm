package service

import (
	"context"

	"github.com/vsamtuc/mcm/pkg/course"
)

// Service provides business operations on top of the course store.
type Service struct {
	store course.Store
}

// New creates a Service backed by the provided store.
func New(store course.Store) *Service {
	if store == nil {
		panic("course store is nil")
	}
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context) ([]course.Course, error) {
	return s.store.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (course.Course, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, input course.CreateCourseInput) (course.Course, error) {
	return s.store.Create(ctx, input)
}

func (s *Service) Update(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error) {
	return s.store.Update(ctx, id, input)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

var _ course.Service = (*Service)(nil)
