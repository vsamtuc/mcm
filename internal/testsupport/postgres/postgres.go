package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Instance wraps a running postgres container and its connection string.
type Instance struct {
	Container  *pg.PostgresContainer
	ConnString string
	cancel     context.CancelFunc
}

// Start launches a Postgres container for tests and registers cleanup with t.Cleanup.
func Start(ctx context.Context, t *testing.T) *Instance {
	t.Helper()

	// Let Ryuk run by default
	// t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	containerCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)

	pgContainer, err := pg.RunContainer(containerCtx,
		testcontainers.WithImage("postgres:16-alpine"),
		pg.WithDatabase("mcm_test"),
		pg.WithUsername("mcm"),
		pg.WithPassword("mcm"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	conn, err := pgContainer.ConnectionString(containerCtx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(context.Background())
		t.Fatalf("fetch connection string: %v", err)
	}

	if err := waitForReady(containerCtx, conn); err != nil {
		_ = pgContainer.Terminate(context.Background())
		t.Fatalf("postgres never became ready: %v", err)
	}

	inst := &Instance{Container: pgContainer, ConnString: conn, cancel: cancel}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
		inst.cancel()
	})

	return inst
}

// ConnURI returns a database URL suitable for database/sql with pgx driver.
func (i *Instance) ConnURI() string {
	return i.ConnString
}

func waitForReady(ctx context.Context, dsn string) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		db, err := sql.Open("pgx", dsn)
		if err == nil {
			pingErr := db.PingContext(ctx)
			_ = db.Close()
			cancel()
			if pingErr == nil {
				return nil
			}
		}
		if err != nil {
			cancel()
		}
		if time.Now().After(deadline) {
			if err != nil {
				return err
			}
			return errors.New("ping timeout")
		}
		time.Sleep(500 * time.Millisecond)
	}
}
