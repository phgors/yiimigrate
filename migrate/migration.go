package migrate

import (
	"context"
	"database/sql"
	"errors"
)

// ErrIrreversibleMigration is returned by migrations that cannot be reverted.
var ErrIrreversibleMigration = errors.New("irreversible migration")

// ErrMigrationNotFound is returned when an applied migration has no registered implementation.
var ErrMigrationNotFound = errors.New("migration not found")

// ErrMigrationLockTimeout is returned when the migration lock cannot be acquired.
var ErrMigrationLockTimeout = errors.New("migration lock timeout")

// Migration is implemented by concrete database migrations.
type Migration interface {
	// Name returns the unique migration version name.
	Name() string
	// Up applies the migration.
	Up(ctx context.Context, m *MigrationContext) error
	// Down reverts the migration.
	Down(ctx context.Context, m *MigrationContext) error
}

// DBTX is the shared interface implemented by *sql.DB and *sql.Tx.
type DBTX interface {
	// ExecContext executes a query that does not return rows.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	// QueryContext executes a query that returns rows.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRowContext executes a query expected to return at most one row.
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// AppliedMigration describes a migration recorded in the migration table.
type AppliedMigration struct {
	// Version is the migration version name.
	Version string
	// ApplyTime is the Unix timestamp recorded when the migration was applied.
	ApplyTime int64
}
