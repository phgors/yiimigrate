package migrate

// DefaultMigrationTable is the default table used to record applied migrations.
const DefaultMigrationTable = "migration"

// Dialect builds SQL for a supported database engine.
type Dialect interface {
	// QuoteTable quotes a table identifier.
	QuoteTable(name string) string
	// QuoteColumn quotes a column identifier.
	QuoteColumn(name string) string
	// Placeholder returns the parameter placeholder for a one-based argument index.
	Placeholder(index int) string

	// CreateMigrationTableSQL returns SQL that creates the migration table.
	CreateMigrationTableSQL(table string) string
	// SelectAppliedMigrationsSQL returns SQL that lists applied migrations.
	SelectAppliedMigrationsSQL(table string, descending bool) string
	// InsertMigrationSQL returns SQL that records an applied migration.
	InsertMigrationSQL(table string) string
	// DeleteMigrationSQL returns SQL that removes an applied migration record.
	DeleteMigrationSQL(table string) string
}
