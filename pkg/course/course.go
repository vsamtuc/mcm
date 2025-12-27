package course

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Course represents a course offering students can enroll in.
type Course struct {
	ID          int64        `json:"id"`
	Code        string       `json:"code"`
	Title       string       `json:"title"`
	Term        string       `json:"term"`
	Instructors []Instructor `json:"instructors"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Instructor links a professor to a course and captures their role.
type Instructor struct {
	ProfessorID int64  `json:"professor_id"`
	Role        string `json:"role"`
}

// CreateCourseInput captures the fields required to create a course.
type CreateCourseInput struct {
	Code        string       `json:"code"`
	Title       string       `json:"title"`
	Term        string       `json:"term"`
	Instructors []Instructor `json:"instructors"`
}

// UpdateCourseInput contains fields that can be patched on a course.
type UpdateCourseInput struct {
	Code        *string       `json:"code"`
	Title       *string       `json:"title"`
	Term        *string       `json:"term"`
	Instructors *[]Instructor `json:"instructors"`
}

// Store manages persistence for courses.
type Store interface {
	List(ctx context.Context) ([]Course, error)
	Get(ctx context.Context, id int64) (Course, error)
	Create(ctx context.Context, input CreateCourseInput) (Course, error)
	Update(ctx context.Context, id int64, input UpdateCourseInput) (Course, error)
	Delete(ctx context.Context, id int64) error
}

var (
	errEmptyCode  = errors.New("course code cannot be empty")
	errEmptyTitle = errors.New("course title cannot be empty")
	errEmptyTerm  = errors.New("course term cannot be empty")
	// ErrNotFound indicates the requested course does not exist.
	ErrNotFound = errors.New("course not found")
)

// ValidateCreate ensures the input has minimally acceptable data.
func ValidateCreate(input CreateCourseInput) error {
	if strings.TrimSpace(input.Code) == "" {
		return errEmptyCode
	}
	if strings.TrimSpace(input.Title) == "" {
		return errEmptyTitle
	}
	if strings.TrimSpace(input.Term) == "" {
		return errEmptyTerm
	}
	return nil
}

// ValidateUpdate ensures update payloads contain at least one field.
func ValidateUpdate(input UpdateCourseInput) error {
	if input.Code == nil && input.Title == nil && input.Term == nil && input.Instructors == nil {
		return errors.New("no fields provided for update")
	}
	if input.Code != nil && strings.TrimSpace(*input.Code) == "" {
		return errEmptyCode
	}
	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		return errEmptyTitle
	}
	if input.Term != nil && strings.TrimSpace(*input.Term) == "" {
		return errEmptyTerm
	}
	return nil
}
