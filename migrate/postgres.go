package migrate

import (
	"fmt"
	"strings"
)

// PostgreSQLDialect generates SQL for PostgreSQL.
type PostgreSQLDialect struct{}

// Name returns the dialect name.
func (d PostgreSQLDialect) Name() string { return "postgres" }

// QuoteTable quotes a table name with double quotes.
func (d PostgreSQLDialect) QuoteTable(name string) string { return quoteName(name, `"`) }

// QuoteColumn quotes a column name with double quotes.
func (d PostgreSQLDialect) QuoteColumn(name string) string { return quoteName(name, `"`) }

// QuoteIndexColumn quotes an index column unless it is an expression.
func (d PostgreSQLDialect) QuoteIndexColumn(name string) string { return quoteIndexColumn(d, name) }

// Placeholder returns a PostgreSQL positional placeholder.
func (d PostgreSQLDialect) Placeholder(index int) string {
	return "$" + fmt.Sprintf("%d", index)
}

// BuildColumn builds one PostgreSQL column definition.
func (d PostgreSQLDialect) BuildColumn(c *ColumnBuilder) (string, error) {
	if c == nil {
		return "", fmt.Errorf("migrate: column builder is nil")
	}
	if c.unsigned {
		return "", unsupported(d.Name(), "UNSIGNED")
	}
	if c.charset != "" {
		return "", unsupported(d.Name(), "CHARACTER SET")
	}
	if c.collation != "" {
		return "", unsupported(d.Name(), "COLLATE")
	}
	if c.after != "" {
		return "", unsupported(d.Name(), "AFTER")
	}
	if c.first {
		return "", unsupported(d.Name(), "FIRST")
	}

	serial := pgSerialType(c)
	parts := []string{serial}
	if serial == "" {
		parts = []string{pgType(c)}
		if c.nullSet {
			if c.nullable {
				parts = append(parts, "NULL")
			} else {
				parts = append(parts, "NOT NULL")
			}
		}
	} else {
		if c.primaryKey {
			parts = append(parts, "PRIMARY KEY")
		}
	}
	if serial == "" && c.defaultExpr != "" {
		parts = append(parts, "DEFAULT", c.defaultExpr)
	}
	if serial == "" && c.defaultSet {
		parts = append(parts, "DEFAULT", sqlLiteral(c.defaultValue))
	}
	if c.check != "" {
		parts = append(parts, "CHECK ("+c.check+")")
	}
	if c.unique {
		parts = append(parts, "UNIQUE")
	}
	if c.generatedAs != "" {
		parts = append(parts, "GENERATED ALWAYS AS ("+c.generatedAs+")")
		if c.generatedKind != "" {
			parts = append(parts, c.generatedKind)
		}
	}
	if c.appendSQL != "" {
		parts = append(parts, c.appendSQL)
	}
	return strings.Join(parts, " "), nil
}

func pgSerialType(c *ColumnBuilder) string {
	if c.primaryKey && c.autoIncrement {
		switch c.typeName {
		case "integer":
			return "SERIAL"
		case "bigInteger":
			return "BIGSERIAL"
		}
	}
	return ""
}

func pgType(c *ColumnBuilder) string {
	switch c.typeName {
	case "tinyInteger":
		return "SMALLINT"
	case "smallInteger":
		return "SMALLINT"
	case "integer":
		return "INTEGER"
	case "bigInteger":
		return "BIGINT"
	case "string":
		return sizedType("VARCHAR", c.size, 255)
	case "char":
		return sizedType("CHAR", c.size)
	case "text":
		return "TEXT"
	case "tinyText":
		return "TEXT"
	case "mediumText":
		return "TEXT"
	case "longText":
		return "TEXT"
	case "binary":
		return "BYTEA"
	case "tinyBlob":
		return "BYTEA"
	case "mediumBlob":
		return "BYTEA"
	case "longBlob":
		return "BYTEA"
	case "boolean":
		return "BOOLEAN"
	case "float":
		return "REAL"
	case "double":
		return "DOUBLE PRECISION"
	case "decimal":
		return sizedType("DECIMAL", c.size)
	case "money":
		return "MONEY"
	case "date":
		return "DATE"
	case "dateTime":
		return precisionType("TIMESTAMP", c.size)
	case "time":
		return precisionType("TIME", c.size)
	case "timestamp":
		return precisionType("TIMESTAMP", c.size)
	case "json":
		return "JSONB"
	case "uuid":
		return "UUID"
	case "enum":
		return "TEXT"
	case "set":
		return "TEXT"
	default:
		return strings.ToUpper(c.typeName)
	}
}

// TableExistsSQL returns SQL checking for a table.
func (d PostgreSQLDialect) TableExistsSQL(table string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1",
		Args:  []any{table},
	}
}

// ColumnExistsSQL returns SQL checking for a column.
func (d PostgreSQLDialect) ColumnExistsSQL(table, column string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2",
		Args:  []any{table, column},
	}
}

// IndexExistsSQL returns SQL checking for an index.
func (d PostgreSQLDialect) IndexExistsSQL(table, index string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1 AND indexname = $2",
		Args:  []any{table, index},
	}
}

// ForeignKeyExistsSQL returns SQL checking for a foreign key.
func (d PostgreSQLDialect) ForeignKeyExistsSQL(table, name string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = current_schema() AND table_name = $1 AND constraint_name = $2 AND constraint_type = 'FOREIGN KEY'",
		Args:  []any{table, name},
	}
}

// ConstraintExistsSQL returns SQL checking for a constraint.
func (d PostgreSQLDialect) ConstraintExistsSQL(table, name string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = current_schema() AND table_name = $1 AND constraint_name = $2",
		Args:  []any{table, name},
	}
}

// BuildRowExistsSQL returns SQL for checking if a row exists.
func (d PostgreSQLDialect) BuildRowExistsSQL(table string, condition string) string {
	query := "SELECT EXISTS(SELECT 1 FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return query + ")"
}

// BuildCountRowsSQL returns SQL for counting rows.
func (d PostgreSQLDialect) BuildCountRowsSQL(table string, condition string) string {
	query := "SELECT COUNT(*) FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return query
}

// CreateTable builds CREATE TABLE SQL.
func (d PostgreSQLDialect) CreateTable(table string, columns *ColumnList, options string) (string, error) {
	defs := make([]string, 0, len(columns.Items()))
	for _, item := range columns.Items() {
		columnSQL, err := d.BuildColumn(item.Column)
		if err != nil {
			return "", err
		}
		defs = append(defs, d.QuoteColumn(item.Name)+" "+columnSQL)
	}
	return fmt.Sprintf("CREATE TABLE %s (%s)%s", d.QuoteTable(table), strings.Join(defs, ", "), joinOptions(options)), nil
}

// DropTable builds DROP TABLE SQL.
func (d PostgreSQLDialect) DropTable(table string) (string, error) {
	return "DROP TABLE IF EXISTS " + d.QuoteTable(table), nil
}

// RenameTable builds ALTER TABLE RENAME SQL.
func (d PostgreSQLDialect) RenameTable(oldName, newName string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s RENAME TO %s", d.QuoteTable(oldName), d.QuoteTable(newName)), nil
}

// TruncateTable builds TRUNCATE TABLE SQL.
func (d PostgreSQLDialect) TruncateTable(table string) (string, error) {
	return "TRUNCATE TABLE " + d.QuoteTable(table), nil
}

// AddColumn builds ADD COLUMN SQL.
func (d PostgreSQLDialect) AddColumn(table, column string, builder *ColumnBuilder) (string, error) {
	columnSQL, err := d.BuildColumn(builder)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", d.QuoteTable(table), d.QuoteColumn(column), columnSQL), nil
}

// AlterColumn builds ALTER COLUMN TYPE SQL.
func (d PostgreSQLDialect) AlterColumn(table, column string, builder *ColumnBuilder) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", d.QuoteTable(table), d.QuoteColumn(column), pgType(builder)), nil
}

// DropColumn builds DROP COLUMN SQL.
func (d PostgreSQLDialect) DropColumn(table, column string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", d.QuoteTable(table), d.QuoteColumn(column)), nil
}

// RenameColumn builds RENAME COLUMN SQL.
func (d PostgreSQLDialect) RenameColumn(table, oldName, newName string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", d.QuoteTable(table), d.QuoteColumn(oldName), d.QuoteColumn(newName)), nil
}

// CreateIndex builds CREATE INDEX SQL.
func (d PostgreSQLDialect) CreateIndex(name, table string, columns []string, unique bool) (string, error) {
	prefix := "CREATE INDEX"
	if unique {
		prefix = "CREATE UNIQUE INDEX"
	}
	return fmt.Sprintf("%s %s ON %s (%s)", prefix, d.QuoteTable(name), d.QuoteTable(table), columnList(d, columns)), nil
}

// DropIndex builds DROP INDEX SQL.
func (d PostgreSQLDialect) DropIndex(name, table string) (string, error) {
	return "DROP INDEX IF EXISTS " + d.QuoteTable(name), nil
}

// AddPrimaryKey builds ADD PRIMARY KEY SQL.
func (d PostgreSQLDialect) AddPrimaryKey(name, table string, columns []string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s)", d.QuoteTable(table), d.QuoteTable(name), columnList(d, columns)), nil
}

// DropPrimaryKey builds DROP CONSTRAINT SQL for a primary key.
func (d PostgreSQLDialect) DropPrimaryKey(name, table string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", d.QuoteTable(table), d.QuoteTable(name)), nil
}

// AddForeignKey builds ADD FOREIGN KEY SQL.
func (d PostgreSQLDialect) AddForeignKey(name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) (string, error) {
	query := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)", d.QuoteTable(table), d.QuoteTable(name), columnList(d, columns), d.QuoteTable(refTable), columnList(d, refColumns))
	if onDelete != "" {
		query += " ON DELETE " + string(onDelete)
	}
	if onUpdate != "" {
		query += " ON UPDATE " + string(onUpdate)
	}
	return query, nil
}

// DropForeignKey builds DROP CONSTRAINT SQL for a foreign key.
func (d PostgreSQLDialect) DropForeignKey(name, table string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", d.QuoteTable(table), d.QuoteTable(name)), nil
}

// AddCommentOnColumn builds COMMENT ON COLUMN SQL.
func (d PostgreSQLDialect) AddCommentOnColumn(table, column, comment string) (string, error) {
	return fmt.Sprintf("COMMENT ON COLUMN %s.%s IS %s", d.QuoteTable(table), d.QuoteColumn(column), sqlLiteral(comment)), nil
}

// DropCommentFromColumn builds SQL for removing a column comment.
func (d PostgreSQLDialect) DropCommentFromColumn(table, column string) (string, error) {
	return fmt.Sprintf("COMMENT ON COLUMN %s.%s IS NULL", d.QuoteTable(table), d.QuoteColumn(column)), nil
}

// AddCommentOnTable builds COMMENT ON TABLE SQL.
func (d PostgreSQLDialect) AddCommentOnTable(table, comment string) (string, error) {
	return fmt.Sprintf("COMMENT ON TABLE %s IS %s", d.QuoteTable(table), sqlLiteral(comment)), nil
}

// DropCommentFromTable builds SQL for removing a table comment.
func (d PostgreSQLDialect) DropCommentFromTable(table string) (string, error) {
	return fmt.Sprintf("COMMENT ON TABLE %s IS NULL", d.QuoteTable(table)), nil
}

// Insert builds INSERT SQL.
func (d PostgreSQLDialect) Insert(table string, row Row) (SQLStatement, error) {
	return buildInsert(d, table, row)
}

// BatchInsert builds multi-row INSERT SQL.
func (d PostgreSQLDialect) BatchInsert(table string, columns []string, rows [][]any) (SQLStatement, error) {
	return buildBatchInsert(d, table, columns, rows)
}

// Update builds UPDATE SQL.
func (d PostgreSQLDialect) Update(table string, row Row, condition string, args ...any) (SQLStatement, error) {
	return buildUpdate(d, table, row, condition, args...)
}

// Delete builds DELETE SQL.
func (d PostgreSQLDialect) Delete(table string, condition string, args ...any) (SQLStatement, error) {
	return buildDelete(d, table, condition, args...)
}

// AcquireLockSQL builds SQL for acquiring an advisory lock.
func (d PostgreSQLDialect) AcquireLockSQL(lockName string, timeoutSeconds int) (SQLStatement, error) {
	return SQLStatement{
		Query: "SELECT pg_advisory_lock(hashtext($1))",
		Args:  []any{lockName},
	}, nil
}

// ReleaseLockSQL builds SQL for releasing an advisory lock.
func (d PostgreSQLDialect) ReleaseLockSQL(lockName string) (SQLStatement, error) {
	return SQLStatement{
		Query: "SELECT pg_advisory_unlock(hashtext($1))",
		Args:  []any{lockName},
	}, nil
}
