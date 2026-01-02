package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	appservice "github.com/vsamtuc/mcm/internal/service"
	memorystore "github.com/vsamtuc/mcm/internal/store/memory"
	pgstore "github.com/vsamtuc/mcm/internal/store/postgres"
	"github.com/vsamtuc/mcm/pkg/application"
	"github.com/vsamtuc/mcm/pkg/store"
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
	persistence store.Store
	service     application.Service
	devel       bool
}

func New(logger *slog.Logger) *App {
	st := memorystore.New()
	a := &App{log: logger, persistence: st, service: appservice.New(st), devel: develModeEnabled()}
	if a.devel {
		a.schemaReady.Store(true)
	}
	return a
}

func (a *App) Start(ctx context.Context) error {
	a.log.Info("starting app", "devel_mode", a.devel)
	if a.devel {
		return nil
	}
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
	st := pgstore.New(db)
	a.persistence = st
	a.service = appservice.New(st)
	a.schemaReady.Store(true)
	// init dependencies, DB connections, etc.
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	a.log.Info("stopping app", "timeout", "5s")
	if a.devel {
		return nil
	}
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

// Inspects schema_migrations and verifies the
// deployed version matches expectations. The expected version is defined
// by expectedSchemaVersion constant.
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

// Service exposes the application service for HTTP handlers.
func (a *App) Service() application.Service {
	return a.service
}

// DevelMode reports whether the application is running with development shortcuts.
func (a *App) DevelMode() bool {
	return a.devel
}

func develModeEnabled() bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("MCM_DEVEL_MODE")))
	return val == "1" || val == "true" || val == "yes" || val == "on"
}
