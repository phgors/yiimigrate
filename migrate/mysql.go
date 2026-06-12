package migrate

import (
	"fmt"
	"strings"
)

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

func (d MySQLDialect) columnSQL(builder *ColumnBuilder) string {
	if builder == nil {
		return ""
	}

	parts := []string{builder.sqlType}
	if builder.unsigned {
		parts = append(parts, "UNSIGNED")
	}
	if builder.charset != "" {
		parts = append(parts, "CHARACTER SET "+builder.charset)
	}
	if builder.collation != "" {
		parts = append(parts, "COLLATE "+builder.collation)
	}
	if builder.generatedAs != "" {
		parts = append(parts, "GENERATED ALWAYS AS ("+builder.generatedAs+")")
		if builder.generatedStorage != "" {
			parts = append(parts, builder.generatedStorage)
		}
	}
	if builder.nullable != nil {
		if *builder.nullable {
			parts = append(parts, "NULL")
		} else {
			parts = append(parts, "NOT NULL")
		}
	}
	if builder.autoIncrement {
		parts = append(parts, "AUTO_INCREMENT")
	}
	if builder.primaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if builder.unique {
		parts = append(parts, "UNIQUE")
	}
	if builder.hasDefaultValue {
		parts = append(parts, "DEFAULT "+formatSQLLiteral(builder.defaultValue))
	}
	if builder.defaultExpression != "" {
		parts = append(parts, "DEFAULT "+builder.defaultExpression)
	}
	if builder.comment != "" {
		parts = append(parts, "COMMENT "+quoteSQLString(builder.comment))
	}
	if builder.check != "" {
		parts = append(parts, "CHECK ("+builder.check+")")
	}
	parts = append(parts, builder.appendSQL...)
	if builder.first {
		parts = append(parts, "FIRST")
	} else if builder.after != "" {
		parts = append(parts, "AFTER "+d.QuoteColumn(builder.after))
	}
	return strings.Join(parts, " ")
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

func formatSQLLiteral(value any) string {
	switch v := value.(type) {
	case nil:
		return "NULL"
	case string:
		return quoteSQLString(v)
	case []byte:
		return quoteSQLString(string(v))
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprint(v)
	}
}

func quoteSQLString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
