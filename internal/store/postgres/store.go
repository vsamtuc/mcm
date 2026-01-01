package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vsamtuc/mcm/pkg/course"
	"github.com/vsamtuc/mcm/pkg/store"
)

// Store persists data inside Postgres.
type Store struct {
	db *sql.DB
}

// New creates a Store backed by the provided database handle.
func New(db *sql.DB) *Store {
	if db == nil {
		panic("postgres store requires a db handle")
	}
	return &Store{db: db}
}

var _ store.Store = (*Store)(nil)

const courseColumns = `id, code, title, term, active, created_at, updated_at`

func (s *Store) ListCourses(ctx context.Context) ([]course.Course, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT %s FROM courses ORDER BY id`, courseColumns))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	courses := make([]course.Course, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		courses = append(courses, c)
		ids = append(ids, c.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(courses) == 0 {
		return courses, nil
	}

	instructors, err := s.fetchInstructors(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range courses {
		courses[i].Instructors = cloneInstructors(instructors[courses[i].ID])
	}
	return courses, nil
}

func (s *Store) GetCourse(ctx context.Context, id int64) (course.Course, error) {
	row := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT %s FROM courses WHERE id = $1`, courseColumns), id)
	c, err := scanCourse(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return course.Course{}, course.ErrNotFound
		}
		return course.Course{}, err
	}
	instructors, err := s.fetchInstructors(ctx, []int64{id})
	if err != nil {
		return course.Course{}, err
	}
	c.Instructors = cloneInstructors(instructors[id])
	return c, nil
}

func (s *Store) CreateCourse(ctx context.Context, input course.CreateCourseInput) (course.Course, error) {
	if err := course.ValidateCreate(input); err != nil {
		return course.Course{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return course.Course{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var created course.Course
	err = tx.QueryRowContext(ctx, fmt.Sprintf(`
        INSERT INTO courses (code, title, term, add_drop_deadline, team_lock_date)
        VALUES ($1, $2, $3, NOW(), NOW())
        RETURNING %s`, courseColumns), input.Code, input.Title, input.Term).
		Scan(&created.ID, &created.Code, &created.Title, &created.Term, &created.Active, &created.CreatedAt, &created.UpdatedAt)
	if err != nil {
		return course.Course{}, mapPgError(err)
	}

	if len(input.Instructors) > 0 {
		if err := insertInstructors(ctx, tx, created.ID, input.Instructors); err != nil {
			return course.Course{}, mapPgError(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return course.Course{}, err
	}

	return s.GetCourse(ctx, created.ID)
}

func (s *Store) UpdateCourse(ctx context.Context, id int64, input course.UpdateCourseInput) (course.Course, error) {
	if err := course.ValidateUpdate(input); err != nil {
		return course.Course{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return course.Course{}, err
	}
	defer func() { _ = tx.Rollback() }()

	assignments := make([]string, 0, 3)
	args := make([]any, 0, 3)
	idx := 1
	if input.Code != nil {
		assignments = append(assignments, fmt.Sprintf("code = $%d", idx))
		args = append(args, *input.Code)
		idx++
	}
	if input.Title != nil {
		assignments = append(assignments, fmt.Sprintf("title = $%d", idx))
		args = append(args, *input.Title)
		idx++
	}
	if input.Term != nil {
		assignments = append(assignments, fmt.Sprintf("term = $%d", idx))
		args = append(args, *input.Term)
		idx++
	}
	rowConfirmed := false
	if len(assignments) > 0 {
		assignments = append(assignments, "updated_at = NOW()")
		args = append(args, id)
		query := fmt.Sprintf("UPDATE courses SET %s WHERE id = $%d", strings.Join(assignments, ", "), idx)
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return course.Course{}, mapPgError(err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return course.Course{}, err
		}
		if affected == 0 {
			return course.Course{}, course.ErrNotFound
		}
		rowConfirmed = true
	}
	if input.Instructors != nil {
		if !rowConfirmed {
			if err := ensureCourseExists(ctx, tx, id); err != nil {
				return course.Course{}, err
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM course_instructors WHERE course_id = $1`, id); err != nil {
			return course.Course{}, err
		}
		if len(*input.Instructors) > 0 {
			if err := insertInstructors(ctx, tx, id, *input.Instructors); err != nil {
				return course.Course{}, mapPgError(err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return course.Course{}, err
	}

	return s.GetCourse(ctx, id)
}

func (s *Store) UpdateCourseActive(ctx context.Context, id int64, active bool) (course.Course, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE courses SET active = $1, updated_at = NOW() WHERE id = $2`, active, id)
	if err != nil {
		return course.Course{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return course.Course{}, err
	}
	if n == 0 {
		return course.Course{}, course.ErrNotFound
	}
	return s.GetCourse(ctx, id)
}

func (s *Store) DeleteCourse(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM courses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return course.ErrNotFound
	}
	return nil
}

func (s *Store) FindProfessorIDBySubject(ctx context.Context, subject string) (int64, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return 0, course.ErrNotFound
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM professors WHERE keycloak_id = $1`, subject).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, course.ErrNotFound
		}
		return 0, err
	}
	return id, nil
}

func (s *Store) ListProfessors(ctx context.Context) ([]course.Professor, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, keycloak_id, username, COALESCE(NULLIF('', ''), username) AS email, created_at FROM professors ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profs []course.Professor
	for rows.Next() {
		var p course.Professor
		if err := rows.Scan(&p.ID, &p.KeycloakID, &p.Name, &p.Email, &p.CreatedAt); err != nil {
			return nil, err
		}
		profs = append(profs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return profs, nil
}

func (s *Store) ProfessorCourses(ctx context.Context, professorID int64) ([]course.Course, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT %s FROM courses WHERE id IN (SELECT course_id FROM course_instructors WHERE professor_id = $1) ORDER BY id`, courseColumns), professorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var courses []course.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		courses = append(courses, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return courses, nil
}

func (s *Store) ListStudents(ctx context.Context) ([]course.Student, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, keycloak_id, university_id, username, created_at FROM students ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var students []course.Student
	for rows.Next() {
		var st course.Student
		if err := rows.Scan(&st.ID, &st.KeycloakID, &st.UniversityID, &st.FullName, &st.CreatedAt); err != nil {
			return nil, err
		}
		students = append(students, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return students, nil
}

func (s *Store) StudentCourses(ctx context.Context, studentID int64) ([]course.Course, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT %s FROM courses WHERE id IN (SELECT course_id FROM course_enrollments WHERE student_id = $1) ORDER BY id`, courseColumns), studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var courses []course.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		courses = append(courses, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return courses, nil
}

func (s *Store) EnrollStudent(ctx context.Context, courseID int64, studentID int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO course_enrollments (course_id, student_id) VALUES ($1, $2)`, courseID, studentID)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

func (s *Store) UnenrollStudent(ctx context.Context, courseID int64, studentID int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM course_enrollments WHERE course_id = $1 AND student_id = $2`, courseID, studentID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return course.ErrNotFound
	}
	return nil
}

func (s *Store) ListTeams(ctx context.Context, courseID int64) ([]course.Team, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, course_id, name, status, created_at FROM teams WHERE course_id = $1 ORDER BY id`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []course.Team
	for rows.Next() {
		var t course.Team
		if err := rows.Scan(&t.ID, &t.CourseID, &t.Name, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return teams, nil
}

func (s *Store) CreateTeam(ctx context.Context, input course.CreateTeamInput) (course.Team, error) {
	if strings.TrimSpace(input.Name) == "" {
		return course.Team{}, fmt.Errorf("team name cannot be empty")
	}
	var t course.Team
	err := s.db.QueryRowContext(ctx, `INSERT INTO teams (course_id, name, status) VALUES ($1, $2, COALESCE(NULLIF($3, ''), 'pending')) RETURNING id, course_id, name, status, created_at`, input.CourseID, input.Name, input.Status).
		Scan(&t.ID, &t.CourseID, &t.Name, &t.Status, &t.CreatedAt)
	if err != nil {
		return course.Team{}, mapPgError(err)
	}
	return t, nil
}

func (s *Store) UpdateTeam(ctx context.Context, teamID int64, input course.UpdateTeamInput) (course.Team, error) {
	assignments := make([]string, 0, 2)
	args := make([]any, 0, 2)
	idx := 1
	if input.Name != nil {
		assignments = append(assignments, fmt.Sprintf("name = $%d", idx))
		args = append(args, *input.Name)
		idx++
	}
	if input.Status != nil {
		assignments = append(assignments, fmt.Sprintf("status = $%d", idx))
		args = append(args, *input.Status)
		idx++
	}
	if len(assignments) == 0 {
		// nothing to change, just fetch existing
		return s.fetchTeam(ctx, teamID)
	}
	args = append(args, teamID)
	query := fmt.Sprintf("UPDATE teams SET %s WHERE id = $%d", strings.Join(assignments, ", "), idx)
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return course.Team{}, mapPgError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return course.Team{}, err
	}
	if n == 0 {
		return course.Team{}, course.ErrNotFound
	}
	return s.fetchTeam(ctx, teamID)
}

func (s *Store) DeleteTeam(ctx context.Context, teamID int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM teams WHERE id = $1`, teamID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return course.ErrNotFound
	}
	return nil
}

func (s *Store) AddTeamMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error) {
	if _, err := s.db.ExecContext(ctx, `INSERT INTO team_memberships (team_id, student_id) VALUES ($1, $2)`, teamID, studentID); err != nil {
		return course.Team{}, mapPgError(err)
	}
	return s.fetchTeam(ctx, teamID)
}

func (s *Store) RemoveTeamMember(ctx context.Context, teamID int64, studentID int64) (course.Team, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM team_memberships WHERE team_id = $1 AND student_id = $2`, teamID, studentID)
	if err != nil {
		return course.Team{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return course.Team{}, err
	}
	if n == 0 {
		return course.Team{}, course.ErrNotFound
	}
	return s.fetchTeam(ctx, teamID)
}

func (s *Store) fetchInstructors(ctx context.Context, courseIDs []int64) (map[int64][]course.Instructor, error) {
	result := make(map[int64][]course.Instructor, len(courseIDs))
	if len(courseIDs) == 0 {
		return result, nil
	}
	placeholders := make([]string, len(courseIDs))
	args := make([]any, len(courseIDs))
	for i, id := range courseIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT course_id, professor_id, role FROM course_instructors WHERE course_id IN (%s) ORDER BY course_id, professor_id`, strings.Join(placeholders, ","))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var courseID int64
		var inst course.Instructor
		if err := rows.Scan(&courseID, &inst.ProfessorID, &inst.Role); err != nil {
			return nil, err
		}
		result[courseID] = append(result[courseID], inst)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func insertInstructors(ctx context.Context, tx *sql.Tx, courseID int64, instructors []course.Instructor) error {
	for _, inst := range instructors {
		role := normalizeRole(inst.Role)
		if _, err := tx.ExecContext(ctx, `INSERT INTO course_instructors (course_id, professor_id, role) VALUES ($1, $2, $3)`, courseID, inst.ProfessorID, role); err != nil {
			return err
		}
	}
	return nil
}

func scanCourse(scanner interface{ Scan(dest ...any) error }) (course.Course, error) {
	var c course.Course
	if err := scanner.Scan(&c.ID, &c.Code, &c.Title, &c.Term, &c.Active, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return course.Course{}, err
	}
	return c, nil
}

func cloneInstructors(in []course.Instructor) []course.Instructor {
	if len(in) == 0 {
		return nil
	}
	out := make([]course.Instructor, len(in))
	copy(out, in)
	return out
}

func normalizeRole(role string) string {
	trimmed := strings.TrimSpace(role)
	if trimmed == "" {
		return "primary"
	}
	return trimmed
}

func ensureCourseExists(ctx context.Context, tx *sql.Tx, id int64) error {
	var existsID int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM courses WHERE id = $1`, id).Scan(&existsID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return course.ErrNotFound
		}
		return err
	}
	return nil
}

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		if pgErr.ConstraintName == "courses_code_key" {
			return fmt.Errorf("course code already exists: %w", err)
		}
		return fmt.Errorf("unique constraint violation: %w", err)
	case "23503":
		return fmt.Errorf("related record missing: %w", err)
	default:
		return err
	}
}

func (s *Store) fetchTeam(ctx context.Context, id int64) (course.Team, error) {
	var t course.Team
	err := s.db.QueryRowContext(ctx, `SELECT id, course_id, name, status, created_at FROM teams WHERE id = $1`, id).
		Scan(&t.ID, &t.CourseID, &t.Name, &t.Status, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return course.Team{}, course.ErrNotFound
		}
		return course.Team{}, err
	}
	return t, nil
}
