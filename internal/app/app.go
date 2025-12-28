package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/vsamtuc/mcm/internal/course/memory"
	pgstore "github.com/vsamtuc/mcm/internal/course/postgres"
	"github.com/vsamtuc/mcm/internal/course/service"
	"github.com/vsamtuc/mcm/pkg/course"
)

const (
	// expectedSchemaVersion is the schema version this executable expects the database to be at.
	expectedSchemaVersion int64 = 1

	// schemaCheckTimeout is the maximum time to wait for the database schema verification.
	schemaCheckTimeout = 5 * time.Second
)

type App struct {
	log         *slog.Logger
	schemaReady atomic.Bool
	db          *sql.DB
	courseStore course.Store
	courseSvc   course.Service
}

func New(logger *slog.Logger) *App {
	store := memory.NewStore()
	return &App{log: logger, courseStore: store, courseSvc: service.New(store)}
}

func (a *App) Start(ctx context.Context) error {
	a.log.Info("starting app")
	a.schemaReady.Store(false)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL env var is not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database connection: %w", err)
	}
	if err := a.ensureSchema(ctx, db); err != nil {
		_ = db.Close()
		return err
	}
	a.db = db
	store := pgstore.New(db)
	a.courseStore = store
	a.courseSvc = service.New(store)
	a.schemaReady.Store(true)
	// init dependencies, DB connections, etc.
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	a.log.Info("stopping app", "timeout", "5s")
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.log.Error("close database", "err", err)
		}
		a.db = nil
	}
	// graceful shutdown
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ensureSchema inspects schema_migrations and verifies the deployed version matches expectations.
func (a *App) ensureSchema(parent context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("database handle is nil")
	}
	ctx, cancel := context.WithTimeout(parent, schemaCheckTimeout)
	defer cancel()

	var version int64
	var dirty bool
	row := db.QueryRowContext(ctx, `
		SELECT version, dirty
		FROM schema_migrations
		ORDER BY version DESC
		LIMIT 1`)
	switch err := row.Scan(&version, &dirty); {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("schema_migrations table is empty; expected version %d", expectedSchemaVersion)
	case err != nil:
		return fmt.Errorf("read schema version: %w", err)
	}

	if dirty {
		return errors.New("database schema is marked dirty; rerun migrations to clear it")
	}

	if version != expectedSchemaVersion {
		return fmt.Errorf("schema version mismatch: db=%d expected=%d", version, expectedSchemaVersion)
	}

	a.log.Info("schema verified", "version", version)
	return nil
}

// SchemaReady returns true once ensureSchema completes successfully.
func (a *App) SchemaReady() bool {
	return a.schemaReady.Load()
}

// Courses exposes the backing course store for handlers.
func (a *App) Courses() course.Service {
	return a.courseSvc
}
