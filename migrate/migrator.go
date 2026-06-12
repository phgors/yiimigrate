package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"
)

// Migrator applies, reverts, and inspects registered migrations.
type Migrator struct {
	db         *sql.DB
	dialect    Dialect
	migrations []Migration
	now        func() time.Time

	// MigrationTable is the table used to record applied migrations.
	MigrationTable string
}

// NewMigrator creates a migrator for the provided database and migrations.
func NewMigrator(db *sql.DB, dialect Dialect, migrations []Migration) *Migrator {
	if dialect == nil {
		dialect = MySQLDialect{}
	}
	copied := append([]Migration(nil), migrations...)
	sort.Slice(copied, func(i, j int) bool {
		return copied[i].Name() < copied[j].Name()
	})
	return &Migrator{
		db:             db,
		dialect:        dialect,
		migrations:     copied,
		now:            time.Now,
		MigrationTable: DefaultMigrationTable,
	}
}

// EnsureMigrationTable creates the migration table when it does not exist.
func (m *Migrator) EnsureMigrationTable(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, m.dialect.CreateMigrationTableSQL(m.migrationTable()))
	return err
}

// Applied returns applied migrations ordered by apply time from oldest to newest.
func (m *Migrator) Applied(ctx context.Context) ([]AppliedMigration, error) {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return nil, err
	}
	return m.loadApplied(ctx, false)
}

// Pending returns registered migrations that have not been applied.
func (m *Migrator) Pending(ctx context.Context) ([]Migration, error) {
	applied, err := m.Applied(ctx)
	if err != nil {
		return nil, err
	}
	appliedSet := make(map[string]struct{}, len(applied))
	for _, migration := range applied {
		appliedSet[migration.Version] = struct{}{}
	}

	pending := make([]Migration, 0, len(m.migrations))
	for _, migration := range m.migrations {
		if _, ok := appliedSet[migration.Name()]; !ok {
			pending = append(pending, migration)
		}
	}
	return pending, nil
}

// History returns applied migrations ordered from newest to oldest.
func (m *Migrator) History(ctx context.Context, limit int) ([]AppliedMigration, error) {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return nil, err
	}
	applied, err := m.loadApplied(ctx, true)
	if err != nil {
		return nil, err
	}
	return limitApplied(applied, limit), nil
}

// Up applies pending migrations. A limit of zero or less applies all pending migrations.
func (m *Migrator) Up(ctx context.Context, limit int) ([]string, error) {
	pending, err := m.Pending(ctx)
	if err != nil {
		return nil, err
	}
	pending = limitMigrations(pending, limit)

	applied := make([]string, 0, len(pending))
	for _, migration := range pending {
		if err := m.apply(ctx, migration); err != nil {
			return applied, err
		}
		applied = append(applied, migration.Name())
	}
	return applied, nil
}

// Down reverts applied migrations from newest to oldest. A limit of zero or less reverts all applied migrations.
func (m *Migrator) Down(ctx context.Context, limit int) ([]string, error) {
	history, err := m.History(ctx, limit)
	if err != nil {
		return nil, err
	}
	registered := m.registeredByName()

	reverted := make([]string, 0, len(history))
	for _, applied := range history {
		migration, ok := registered[applied.Version]
		if !ok {
			return reverted, fmt.Errorf("%w: %s", ErrMigrationNotFound, applied.Version)
		}
		if err := m.revert(ctx, migration); err != nil {
			return reverted, err
		}
		reverted = append(reverted, migration.Name())
	}
	return reverted, nil
}

func (m *Migrator) loadApplied(ctx context.Context, descending bool) ([]AppliedMigration, error) {
	rows, err := m.db.QueryContext(ctx, m.dialect.SelectAppliedMigrationsSQL(m.migrationTable(), descending))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var applied []AppliedMigration
	for rows.Next() {
		var migration AppliedMigration
		if err := rows.Scan(&migration.Version, &migration.ApplyTime); err != nil {
			return nil, err
		}
		applied = append(applied, migration)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func (m *Migrator) apply(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	migrationContext := NewMigrationContext(tx, m.dialect)
	if err := migration.Up(ctx, migrationContext); err != nil {
		return rollbackWithCause(tx, err)
	}
	if _, err := tx.ExecContext(ctx, m.dialect.InsertMigrationSQL(m.migrationTable()), migration.Name(), m.now().Unix()); err != nil {
		return rollbackWithCause(tx, err)
	}
	return tx.Commit()
}

func (m *Migrator) revert(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	migrationContext := NewMigrationContext(tx, m.dialect)
	if err := migration.Down(ctx, migrationContext); err != nil {
		return rollbackWithCause(tx, err)
	}
	if _, err := tx.ExecContext(ctx, m.dialect.DeleteMigrationSQL(m.migrationTable()), migration.Name()); err != nil {
		return rollbackWithCause(tx, err)
	}
	return tx.Commit()
}

func (m *Migrator) registeredByName() map[string]Migration {
	registered := make(map[string]Migration, len(m.migrations))
	for _, migration := range m.migrations {
		registered[migration.Name()] = migration
	}
	return registered
}

func (m *Migrator) migrationTable() string {
	if m.MigrationTable == "" {
		return DefaultMigrationTable
	}
	return m.MigrationTable
}

func rollbackWithCause(tx *sql.Tx, cause error) error {
	if err := tx.Rollback(); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

func limitMigrations(migrations []Migration, limit int) []Migration {
	if limit <= 0 || limit >= len(migrations) {
		return migrations
	}
	return migrations[:limit]
}

func limitApplied(applied []AppliedMigration, limit int) []AppliedMigration {
	if limit <= 0 || limit >= len(applied) {
		return applied
	}
	return applied[:limit]
}
