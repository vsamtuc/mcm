package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/vsamtuc/mcm/pkg/course"
	"github.com/vsamtuc/mcm/pkg/store"
)

// Store is an in-memory implementation of store.Store for local development.
type Store struct {
	mu         sync.RWMutex
	nextID     int64
	courses    map[int64]course.Course
	professors map[string]int64
}

// New creates an empty store instance.
func New() *Store {
	return &Store{nextID: 1, courses: make(map[int64]course.Course), professors: make(map[string]int64)}
}

func (s *Store) ListCourses(ctx context.Context) ([]course.Course, error) {
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

func (s *Store) GetCourse(ctx context.Context, id int64) (course.Course, error) {
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

func (s *Store) CreateCourse(ctx context.Context, input course.CreateCourseInput) (course.Course, error) {
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

func (s *Store) UpdateCourse(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error) {
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

func (s *Store) DeleteCourse(ctx context.Context, id int64) error {
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

func (s *Store) FindProfessorIDBySubject(ctx context.Context, subject string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.professors[subject]
	if !ok {
		return 0, course.ErrNotFound
	}
	return id, nil
}

// SeedProfessor registers a professor mapping for development and tests.
func (s *Store) SeedProfessor(subject string, id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.professors[subject] = id
}

var _ store.Store = (*Store)(nil)
