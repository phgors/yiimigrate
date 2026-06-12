package migrate

import (
	"context"
	"time"
)

// MigrationContext exposes database helpers to a running migration.
type MigrationContext struct {
	db      DBTX
	dialect Dialect
	dryRun  bool
	logger  Logger
}

// NewMigrationContext creates a migration context for the provided executor.
func NewMigrationContext(db DBTX, dialect Dialect) *MigrationContext {
	if dialect == nil {
		dialect = MySQLDialect{}
	}
	return &MigrationContext{db: db, dialect: dialect, logger: discardLogger{}}
}

// Execute runs a SQL statement with context-aware database execution.
func (m *MigrationContext) Execute(ctx context.Context, query string, args ...any) error {
	return m.exec(ctx, SQLStatement{Query: query, Args: args})
}

// Dialect returns the SQL dialect used by the migration.
func (m *MigrationContext) Dialect() Dialect {
	return m.dialect
}

// Schema creates a chainable schema plan.
func (m *MigrationContext) Schema() *SchemaPlan {
	return &SchemaPlan{ctx: m}
}

// SetDryRun toggles dry-run mode for schema execution.
func (m *MigrationContext) SetDryRun(dryRun bool) {
	m.dryRun = dryRun
}

// SetLogger sets the context logger.
func (m *MigrationContext) SetLogger(logger Logger) {
	if logger == nil {
		logger = discardLogger{}
	}
	m.logger = logger
}

func (m *MigrationContext) exec(ctx context.Context, statement SQLStatement) error {
	start := time.Now()
	if m.logger != nil {
		m.logger.Printf("    > %s", statement.Query)
	}
	if m.dryRun {
		return nil
	}
	_, err := m.db.ExecContext(ctx, statement.Query, statement.Args...)
	if m.logger != nil {
		m.logger.Printf("      %.3fs", time.Since(start).Seconds())
	}
	return err
}
