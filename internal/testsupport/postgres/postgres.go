package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// Instance wraps a running postgres container and its connection string.
type Instance struct {
	Container  *pg.PostgresContainer
	ConnString string
}

// Start launches a Postgres container for tests and registers cleanup with t.Cleanup.
func Start(ctx context.Context, t *testing.T) *Instance {
	t.Helper()

	containerCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

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

	inst := &Instance{Container: pgContainer, ConnString: conn}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminate postgres container: %v", err)
		}
	})

	return inst
}

// ConnURI returns a database URL suitable for database/sql with pgx driver.
func (i *Instance) ConnURI() string {
	return i.ConnString
}
