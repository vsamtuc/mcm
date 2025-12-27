package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/vsamtuc/mcm/pkg/course"
)

// Store is an in-memory implementation of course.Store for local development.
type Store struct {
	mu      sync.RWMutex
	nextID  int64
	courses map[int64]course.Course
}

// NewStore creates an empty course store.
func NewStore() *Store {
	return &Store{nextID: 1, courses: make(map[int64]course.Course)}
}

func (s *Store) List(ctx context.Context) ([]course.Course, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]course.Course, 0, len(s.courses))
	for _, c := range s.courses {
		items = append(items, c)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *Store) Get(ctx context.Context, id int64) (course.Course, error) {
	if err := ctx.Err(); err != nil {
		return course.Course{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.courses[id]
	if !ok {
		return course.Course{}, course.ErrNotFound
	}
	return c, nil
}

func (s *Store) Create(ctx context.Context, input course.CreateCourseInput) (course.Course, error) {
	if err := course.ValidateCreate(input); err != nil {
		return course.Course{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	now := time.Now()
	c := course.Course{
		ID:          id,
		Code:        input.Code,
		Title:       input.Title,
		Term:        input.Term,
		Instructors: append([]course.Instructor(nil), input.Instructors...),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.courses[id] = c
	return c, nil
}

func (s *Store) Update(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error) {
	if err := course.ValidateUpdate(input); err != nil {
		return course.Course{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.courses[id]
	if !ok {
		return course.Course{}, course.ErrNotFound
	}
	if input.Code != nil {
		c.Code = *input.Code
	}
	if input.Title != nil {
		c.Title = *input.Title
	}
	if input.Term != nil {
		c.Term = *input.Term
	}
	if input.Instructors != nil {
		c.Instructors = append([]course.Instructor(nil), (*input.Instructors)...)
	}
	c.UpdatedAt = time.Now()
	s.courses[id] = c
	return c, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.courses[id]; !ok {
		return course.ErrNotFound
	}
	delete(s.courses, id)
	return nil
}
