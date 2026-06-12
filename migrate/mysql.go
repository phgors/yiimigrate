package migrate

import "strings"

// MySQLDialect builds SQL for MySQL databases.
type MySQLDialect struct{}

// QuoteTable quotes a MySQL table name with backticks.
func (MySQLDialect) QuoteTable(name string) string {
	return quoteDottedIdentifier(name)
}

// QuoteColumn quotes a MySQL column name with backticks.
func (MySQLDialect) QuoteColumn(name string) string {
	return quoteDottedIdentifier(name)
}

// Placeholder returns a MySQL parameter placeholder.
func (MySQLDialect) Placeholder(index int) string {
	return "?"
}

// CreateMigrationTableSQL returns SQL that creates the migration table when missing.
func (d MySQLDialect) CreateMigrationTableSQL(table string) string {
	return "CREATE TABLE IF NOT EXISTS " + d.QuoteTable(table) +
		" (" + d.QuoteColumn("version") + " varchar(180) NOT NULL PRIMARY KEY, " +
		d.QuoteColumn("apply_time") + " int NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"
}

// SelectAppliedMigrationsSQL returns SQL that lists applied migrations.
func (d MySQLDialect) SelectAppliedMigrationsSQL(table string, descending bool) string {
	direction := "ASC"
	if descending {
		direction = "DESC"
	}
	return "SELECT " + d.QuoteColumn("version") + ", " + d.QuoteColumn("apply_time") +
		" FROM " + d.QuoteTable(table) +
		" ORDER BY " + d.QuoteColumn("apply_time") + " " + direction +
		", " + d.QuoteColumn("version") + " " + direction
}

// InsertMigrationSQL returns SQL that records an applied migration.
func (d MySQLDialect) InsertMigrationSQL(table string) string {
	return "INSERT INTO " + d.QuoteTable(table) +
		" (" + d.QuoteColumn("version") + ", " + d.QuoteColumn("apply_time") +
		") VALUES (" + d.Placeholder(1) + ", " + d.Placeholder(2) + ")"
}

// DeleteMigrationSQL returns SQL that removes an applied migration record.
func (d MySQLDialect) DeleteMigrationSQL(table string) string {
	return "DELETE FROM " + d.QuoteTable(table) +
		" WHERE " + d.QuoteColumn("version") + " = " + d.Placeholder(1)
}

func quoteDottedIdentifier(name string) string {
	if name == "" {
		return "``"
	}
	parts := strings.Split(name, ".")
	for i, part := range parts {
		parts[i] = quoteIdentifier(part)
	}
	return strings.Join(parts, ".")
}

func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}
