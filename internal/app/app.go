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
}

func New(logger *slog.Logger) *App {
	return &App{log: logger}
}

func (a *App) Start(ctx context.Context) error {
	a.log.Info("starting app")
	a.schemaReady.Store(false)
	if err := a.ensureSchema(ctx); err != nil {
		return err
	}
	a.schemaReady.Store(true)
	// init dependencies, DB connections, etc.
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	a.log.Info("stopping app", "timeout", "5s")
	// graceful shutdown
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ensureSchema connects to the database, inspects schema_migrations, and verifies the
// deployed version matches the executable's expectation.
func (a *App) ensureSchema(parent context.Context) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL env var is not set")
	}

	ctx, cancel := context.WithTimeout(parent, schemaCheckTimeout)
	defer cancel()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database connection: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

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
