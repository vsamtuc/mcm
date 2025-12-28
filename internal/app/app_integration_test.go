package app

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	testpg "github.com/vsamtuc/mcm/internal/testsupport/postgres"
)

func TestEnsureSchemaValidatesVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping postgres integration test in short mode")
	}

	ctx := context.Background()
	pg := testpg.Start(ctx, t)

	t.Setenv("DATABASE_URL", pg.ConnURI())

	t.Run("matching version passes", func(t *testing.T) {
		prepSchema(t, pg.ConnURI(), expectedSchemaVersion, false)
		app := New(discardLogger())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		db := openTestDB(t, pg.ConnURI())
		if err := app.ensureSchema(ctx, db); err != nil {
			t.Fatalf("ensureSchema returned error: %v", err)
		}
	})

	t.Run("mismatched version fails", func(t *testing.T) {
		prepSchema(t, pg.ConnURI(), expectedSchemaVersion+1, false)
		app := New(discardLogger())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		db := openTestDB(t, pg.ConnURI())
		if err := app.ensureSchema(ctx, db); err == nil || !strings.Contains(err.Error(), "schema version mismatch") {
			t.Fatalf("expected mismatch error, got %v", err)
		}
	})
}

func prepSchema(t *testing.T, dsn string, version int64, dirty bool) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	exec := func(query string, args ...any) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec %q: %v", query, err)
		}
	}

	exec(`DROP TABLE IF EXISTS schema_migrations`)
	exec(`CREATE TABLE schema_migrations (version BIGINT NOT NULL, dirty BOOLEAN NOT NULL DEFAULT false)`)
	exec(`INSERT INTO schema_migrations(version, dirty) VALUES ($1, $2)`, version, dirty)
}

func openTestDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
