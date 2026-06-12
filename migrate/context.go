package migrate

import "context"

// MigrationContext exposes database helpers to a running migration.
type MigrationContext struct {
	db      DBTX
	dialect Dialect
}

// NewMigrationContext creates a migration context for the provided executor.
func NewMigrationContext(db DBTX, dialect Dialect) *MigrationContext {
	return &MigrationContext{db: db, dialect: dialect}
}

// Execute runs a SQL statement with context-aware database execution.
func (m *MigrationContext) Execute(ctx context.Context, query string, args ...any) error {
	_, err := m.db.ExecContext(ctx, query, args...)
	return err
}

// Dialect returns the SQL dialect used by the migration.
func (m *MigrationContext) Dialect() Dialect {
	return m.dialect
}
