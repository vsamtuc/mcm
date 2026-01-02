package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vsamtuc/mcm/pkg/course"
	"github.com/vsamtuc/mcm/pkg/store"
)

// Store is an in-memory implementation of store.Store for local development.
type Store struct {
	mu                sync.RWMutex
	nextCourseID      int64
	nextProfessorID   int64
	nextStudentID     int64
	nextTeamID        int64
	courses           map[int64]course.Course
	professorSubjects map[string]int64
	professors        map[int64]course.Professor
	students          map[int64]course.Student
	enrollments       map[int64]map[int64]struct{}
	teams             map[int64]course.Team
	teamMembers       map[int64]map[int64]struct{}
}

// New creates an empty memory store instance.
func New() *Store {
	return &Store{
		nextCourseID:      1,
		nextProfessorID:   1,
		nextStudentID:     1,
		nextTeamID:        1,
		courses:           make(map[int64]course.Course),
		professorSubjects: make(map[string]int64),
		professors:        make(map[int64]course.Professor),
		students:          make(map[int64]course.Student),
		enrollments:       make(map[int64]map[int64]struct{}),
		teams:             make(map[int64]course.Team),
		teamMembers:       make(map[int64]map[int64]struct{}),
	}
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
	id := s.nextCourseID
	s.nextCourseID++
	now := time.Now()
	c := course.Course{
		ID:          id,
		Code:        input.Code,
		Title:       input.Title,
		Term:        input.Term,
		Active:      true,
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

func (s *Store) UpdateCourseActive(ctx context.Context, id int64, active bool) (course.Course, error) {
	if err := ctx.Err(); err != nil {
		return course.Course{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.courses[id]
	if !ok {
		return course.Course{}, course.ErrNotFound
	}
	c.Active = active
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
	id, ok := s.professorSubjects[subject]
	if !ok {
		return 0, course.ErrNotFound
	}
	return id, nil
}

// SeedProfessor registers a professor mapping for development and tests.
func (s *Store) SeedProfessor(subject string, id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.professorSubjects[subject] = id
	s.professors[id] = course.Professor{ID: id, KeycloakID: subject}
}

func (s *Store) SeedStudent(id int64, keycloak string, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.students[id] = course.Student{ID: id, KeycloakID: keycloak, FullName: name}
}

func (s *Store) ListProfessors(ctx context.Context) ([]course.Professor, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]course.Professor, 0, len(s.professors))
	for _, p := range s.professors {
		items = append(items, p)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *Store) ProfessorCourses(ctx context.Context, professorID int64) ([]course.Course, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []course.Course
	for _, c := range s.courses {
		for _, inst := range c.Instructors {
			if inst.ProfessorID == professorID {
				result = append(result, c)
				break
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (s *Store) ListStudents(ctx context.Context) ([]course.Student, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]course.Student, 0, len(s.students))
	for _, st := range s.students {
		items = append(items, st)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *Store) StudentCourses(ctx context.Context, studentID int64) ([]course.Course, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	courses := make([]course.Course, 0)
	for cid, members := range s.enrollments {
		if _, ok := members[studentID]; ok {
			if c, exists := s.courses[cid]; exists {
				courses = append(courses, c)
			}
		}
	}
	sort.Slice(courses, func(i, j int) bool { return courses[i].ID < courses[j].ID })
	return courses, nil
}

func (s *Store) EnrollStudent(ctx context.Context, courseID int64, studentID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.courses[courseID]; !ok {
		return course.ErrNotFound
	}
	if _, ok := s.students[studentID]; !ok {
		return course.ErrNotFound
	}
	members := s.enrollments[courseID]
	if members == nil {
		members = make(map[int64]struct{})
		s.enrollments[courseID] = members
	}
	if _, exists := members[studentID]; exists {
		return fmt.Errorf("student already enrolled")
	}
	members[studentID] = struct{}{}
	return nil
}

func (s *Store) UnenrollStudent(ctx context.Context, courseID int64, studentID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.enrollments[courseID]
	if !ok {
		return course.ErrNotFound
	}
	if _, exists := members[studentID]; !exists {
		return course.ErrNotFound
	}
	delete(members, studentID)
	return nil
}

func (s *Store) ListTeams(ctx context.Context, courseID int64) ([]course.Team, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	teams := make([]course.Team, 0)
	for _, t := range s.teams {
		if t.CourseID == courseID {
			teams = append(teams, t)
		}
	}
	sort.Slice(teams, func(i, j int) bool { return teams[i].ID < teams[j].ID })
	return teams, nil
}

func (s *Store) CreateTeam(ctx context.Context, input course.CreateTeamInput) (course.Team, error) {
	if strings.TrimSpace(input.Name) == "" {
		return course.Team{}, fmt.Errorf("team name cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return course.Team{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.courses[input.CourseID]; !ok {
		return course.Team{}, course.ErrNotFound
	}
	for _, t := range s.teams {
		if t.CourseID == input.CourseID && strings.EqualFold(strings.TrimSpace(t.Name), strings.TrimSpace(input.Name)) {
			return course.Team{}, fmt.Errorf("team name already exists")
		}
	}
	id := s.nextTeamID
	s.nextTeamID++
	team := course.Team{ID: id, CourseID: input.CourseID, Name: input.Name, Status: input.Status}
	s.teams[id] = team
	return team, nil
}

func (s *Store) UpdateTeam(ctx context.Context, teamID int64, input course.UpdateTeamInput) (course.Team, error) {
	if err := ctx.Err(); err != nil {
		return course.Team{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	team, ok := s.teams[teamID]
	if !ok {
		return course.Team{}, course.ErrNotFound
	}
	if input.Name != nil {
		team.Name = *input.Name
	}
	if input.Status != nil {
		team.Status = *input.Status
	}
	s.teams[teamID] = team
	return team, nil
}

func (s *Store) DeleteTeam(ctx context.Context, teamID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.teams[teamID]; !ok {
		return course.ErrNotFound
	}
	delete(s.teams, teamID)
	delete(s.teamMembers, teamID)
	return nil
}

func (s *Store) AddTeamMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error) {
	if err := ctx.Err(); err != nil {
		return course.Team{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	team, ok := s.teams[teamID]
	if !ok {
		return course.Team{}, course.ErrNotFound
	}
	if _, ok := s.students[studentID]; !ok {
		return course.Team{}, course.ErrNotFound
	}
	members := s.teamMembers[teamID]
	if members == nil {
		members = make(map[int64]struct{})
		s.teamMembers[teamID] = members
	}
	if _, exists := members[studentID]; exists {
		return course.Team{}, fmt.Errorf("student already in team")
	}
	members[studentID] = struct{}{}
	return team, nil
}

func (s *Store) RemoveTeamMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error) {
	if err := ctx.Err(); err != nil {
		return course.Team{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	team, ok := s.teams[teamID]
	if !ok {
		return course.Team{}, course.ErrNotFound
	}
	members := s.teamMembers[teamID]
	if members == nil {
		return course.Team{}, course.ErrNotFound
	}
	if _, exists := members[studentID]; !exists {
		return course.Team{}, course.ErrNotFound
	}
	delete(members, studentID)
	return team, nil
}

var _ store.Store = (*Store)(nil)
