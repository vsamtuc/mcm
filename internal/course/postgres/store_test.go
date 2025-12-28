package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	testpg "github.com/vsamtuc/mcm/internal/testsupport/postgres"
	"github.com/vsamtuc/mcm/pkg/course"
)

func TestStoreCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping postgres integration test in short mode")
	}

	ctx := context.Background()
	pg := testpg.Start(ctx, t)
	db := openDB(t, pg.ConnURI())
	applyMigrations(t, db)

	profID := seedProfessor(t, db)
	store := New(db)

	created, err := store.Create(ctx, course.CreateCourseInput{
		Code:  "CSC101",
		Title: "Distributed Systems",
		Term:  "Fall 2025",
		Instructors: []course.Instructor{
			{ProfessorID: profID, Role: "lead"},
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected created ID to be set")
	}
	if created.Code != "CSC101" || created.Title != "Distributed Systems" {
		t.Fatalf("unexpected course: %+v", created)
	}
	if len(created.Instructors) != 1 || created.Instructors[0].ProfessorID != profID {
		t.Fatalf("expected instructor to be persisted, got %+v", created.Instructors)
	}

	fetched, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.ID != created.ID || fetched.Code != created.Code {
		t.Fatalf("fetched course mismatch: %+v vs %+v", fetched, created)
	}

	listed, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 course, got %d", len(listed))
	}

	newTitle := "Advanced Distributed Systems"
	newTerm := "Winter 2026"
	newCode := "CSC201"
	newInstructors := []course.Instructor{{ProfessorID: profID, Role: "coordinator"}}
	updated, err := store.Update(ctx, created.ID, course.UpdateCourseInput{
		Title:       &newTitle,
		Term:        &newTerm,
		Code:        &newCode,
		Instructors: &newInstructors,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != newTitle || updated.Term != newTerm || updated.Code != newCode {
		t.Fatalf("update did not apply fields: %+v", updated)
	}
	if len(updated.Instructors) != 1 || updated.Instructors[0].Role != "coordinator" {
		t.Fatalf("update did not replace instructors: %+v", updated.Instructors)
	}
	if updated.UpdatedAt.Before(created.UpdatedAt) {
		t.Fatalf("expected updated_at to advance: created=%v updated=%v", created.UpdatedAt, updated.UpdatedAt)
	}

	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, course.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func openDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func applyMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	files := []string{
		"0001_init_schema.up.sql",
	}
	for _, file := range files {
		path := filepath.Join("..", "..", "..", "db", "migrations", file)
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := db.Exec(string(contents)); err != nil {
			t.Fatalf("apply migration %s: %v", file, err)
		}
	}
}

func seedProfessor(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(
		`INSERT INTO professors (keycloak_id, full_name, email) VALUES ('00000000-0000-0000-0000-000000000001', 'Ada Lovelace', 'ada@example.com') RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert professor: %v", err)
	}
	return id
}
