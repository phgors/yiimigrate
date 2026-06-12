package migrate

import (
	"fmt"
	"strings"
)

// MySQLDialect generates SQL for MySQL.
type MySQLDialect struct{}

// Name returns the dialect name.
func (d MySQLDialect) Name() string { return "mysql" }

// QuoteTable quotes a table name with backticks.
func (d MySQLDialect) QuoteTable(name string) string { return quoteName(name, "`") }

// QuoteColumn quotes a column name with backticks.
func (d MySQLDialect) QuoteColumn(name string) string { return quoteName(name, "`") }

// QuoteIndexColumn quotes an index column unless it is an expression.
func (d MySQLDialect) QuoteIndexColumn(name string) string { return quoteIndexColumn(d, name) }

// Placeholder returns a MySQL placeholder.
func (d MySQLDialect) Placeholder(index int) string { return "?" }

// BuildColumn builds one MySQL column definition.
func (d MySQLDialect) BuildColumn(c *ColumnBuilder) (string, error) {
	if c == nil {
		return "", fmt.Errorf("migrate: column builder is nil")
	}
	parts := []string{mysqlType(c)}
	if c.unsigned && supportsMySQLUnsigned(c.typeName) {
		parts = append(parts, "UNSIGNED")
	}
	if c.nullSet {
		if c.nullable {
			parts = append(parts, "NULL")
		} else {
			parts = append(parts, "NOT NULL")
		}
	}
	if c.autoIncrement {
		parts = append(parts, "AUTO_INCREMENT")
	}
	if c.primaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if c.unique {
		parts = append(parts, "UNIQUE")
	}
	if c.defaultExpr != "" {
		parts = append(parts, "DEFAULT", c.defaultExpr)
	}
	if c.defaultSet {
		parts = append(parts, "DEFAULT", sqlLiteral(c.defaultValue))
	}
	if c.check != "" {
		parts = append(parts, "CHECK ("+c.check+")")
	}
	if c.generatedAs != "" {
		parts = append(parts, "GENERATED ALWAYS AS ("+c.generatedAs+")")
		if c.generatedKind != "" {
			parts = append(parts, c.generatedKind)
		}
	}
	if c.charset != "" {
		parts = append(parts, "CHARACTER SET", c.charset)
	}
	if c.collation != "" {
		parts = append(parts, "COLLATE", c.collation)
	}
	if c.comment != "" {
		parts = append(parts, "COMMENT", sqlLiteral(c.comment))
	}
	if c.first {
		parts = append(parts, "FIRST")
	}
	if c.after != "" {
		parts = append(parts, "AFTER", d.QuoteColumn(c.after))
	}
	if c.appendSQL != "" {
		parts = append(parts, c.appendSQL)
	}
	return strings.Join(parts, " "), nil
}

func mysqlType(c *ColumnBuilder) string {
	switch c.typeName {
	case "tinyInteger":
		return sizedType("TINYINT", c.size)
	case "smallInteger":
		return sizedType("SMALLINT", c.size)
	case "integer":
		return sizedType("INT", c.size)
	case "bigInteger":
		return sizedType("BIGINT", c.size)
	case "string":
		return sizedType("VARCHAR", c.size, 255)
	case "char":
		return sizedType("CHAR", c.size)
	case "text":
		return "TEXT"
	case "tinyText":
		return "TINYTEXT"
	case "mediumText":
		return "MEDIUMTEXT"
	case "longText":
		return "LONGTEXT"
	case "binary":
		return sizedType("VARBINARY", c.size)
	case "tinyBlob":
		return "TINYBLOB"
	case "mediumBlob":
		return "MEDIUMBLOB"
	case "longBlob":
		return "LONGBLOB"
	case "boolean":
		return "TINYINT(1)"
	case "float":
		return sizedType("FLOAT", c.size)
	case "double":
		return sizedType("DOUBLE", c.size)
	case "decimal", "money":
		return sizedType("DECIMAL", c.size)
	case "date":
		return "DATE"
	case "dateTime":
		return precisionType("DATETIME", c.size)
	case "time":
		return precisionType("TIME", c.size)
	case "timestamp":
		return precisionType("TIMESTAMP", c.size)
	case "json":
		return "JSON"
	case "uuid":
		return "CHAR(36)"
	case "enum":
		return "ENUM(" + enumValues(c.values) + ")"
	case "set":
		return "SET(" + enumValues(c.values) + ")"
	default:
		return strings.ToUpper(c.typeName)
	}
}

func precisionType(name string, size []int) string {
	if len(size) == 0 {
		return name
	}
	return sizedType(name, size)
}

func supportsMySQLUnsigned(typeName string) bool {
	switch typeName {
	case "tinyInteger", "smallInteger", "integer", "bigInteger", "float", "double", "decimal", "money":
		return true
	default:
		return false
	}
}

// TableExistsSQL returns SQL checking for a table.
func (d MySQLDialect) TableExistsSQL(table string) SQLStatement {
	return SQLStatement{Query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", Args: []any{table}}
}

// ColumnExistsSQL returns SQL checking for a column.
func (d MySQLDialect) ColumnExistsSQL(table, column string) SQLStatement {
	return SQLStatement{Query: "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?", Args: []any{table, column}}
}

// IndexExistsSQL returns SQL checking for an index.
func (d MySQLDialect) IndexExistsSQL(table, index string) SQLStatement {
	return SQLStatement{Query: "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?", Args: []any{table, index}}
}

// ForeignKeyExistsSQL returns SQL checking for a foreign key.
func (d MySQLDialect) ForeignKeyExistsSQL(table, name string) SQLStatement {
	return SQLStatement{Query: "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = ? AND constraint_type = 'FOREIGN KEY'", Args: []any{table, name}}
}

// ConstraintExistsSQL returns SQL checking for a constraint.
func (d MySQLDialect) ConstraintExistsSQL(table, name string) SQLStatement {
	return SQLStatement{Query: "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = ?", Args: []any{table, name}}
}

// BuildRowExistsSQL returns SQL for checking if a row exists.
func (d MySQLDialect) BuildRowExistsSQL(table string, condition string) string {
	query := "SELECT EXISTS(SELECT 1 FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return query + ")"
}

// BuildCountRowsSQL returns SQL for counting rows.
func (d MySQLDialect) BuildCountRowsSQL(table string, condition string) string {
	query := "SELECT COUNT(*) FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return query
}

// CreateTable builds CREATE TABLE SQL.
func (d MySQLDialect) CreateTable(table string, columns *ColumnList, options string) (string, error) {
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
func (d MySQLDialect) DropTable(table string) (string, error) {
	return "DROP TABLE " + d.QuoteTable(table), nil
}

// RenameTable builds RENAME TABLE SQL.
func (d MySQLDialect) RenameTable(oldName, newName string) (string, error) {
	return fmt.Sprintf("RENAME TABLE %s TO %s", d.QuoteTable(oldName), d.QuoteTable(newName)), nil
}

// TruncateTable builds TRUNCATE TABLE SQL.
func (d MySQLDialect) TruncateTable(table string) (string, error) {
	return "TRUNCATE TABLE " + d.QuoteTable(table), nil
}

// AddColumn builds ADD COLUMN SQL.
func (d MySQLDialect) AddColumn(table, column string, builder *ColumnBuilder) (string, error) {
	columnSQL, err := d.BuildColumn(builder)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", d.QuoteTable(table), d.QuoteColumn(column), columnSQL), nil
}

// AlterColumn builds ALTER COLUMN SQL.
func (d MySQLDialect) AlterColumn(table, column string, builder *ColumnBuilder) (string, error) {
	columnSQL, err := d.BuildColumn(builder)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s", d.QuoteTable(table), d.QuoteColumn(column), columnSQL), nil
}

// DropColumn builds DROP COLUMN SQL.
func (d MySQLDialect) DropColumn(table, column string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", d.QuoteTable(table), d.QuoteColumn(column)), nil
}

// RenameColumn builds RENAME COLUMN SQL.
func (d MySQLDialect) RenameColumn(table, oldName, newName string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", d.QuoteTable(table), d.QuoteColumn(oldName), d.QuoteColumn(newName)), nil
}

// CreateIndex builds CREATE INDEX SQL.
func (d MySQLDialect) CreateIndex(name, table string, columns []string, unique bool) (string, error) {
	prefix := "CREATE INDEX"
	if unique {
		prefix = "CREATE UNIQUE INDEX"
	}
	return fmt.Sprintf("%s %s ON %s (%s)", prefix, d.QuoteTable(name), d.QuoteTable(table), columnList(d, columns)), nil
}

// DropIndex builds DROP INDEX SQL.
func (d MySQLDialect) DropIndex(name, table string) (string, error) {
	return fmt.Sprintf("DROP INDEX %s ON %s", d.QuoteTable(name), d.QuoteTable(table)), nil
}

// AddPrimaryKey builds ADD PRIMARY KEY SQL.
func (d MySQLDialect) AddPrimaryKey(name, table string, columns []string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s)", d.QuoteTable(table), d.QuoteTable(name), columnList(d, columns)), nil
}

// DropPrimaryKey builds DROP PRIMARY KEY SQL.
func (d MySQLDialect) DropPrimaryKey(name, table string) (string, error) {
	return "ALTER TABLE " + d.QuoteTable(table) + " DROP PRIMARY KEY", nil
}

// AddForeignKey builds ADD FOREIGN KEY SQL.
func (d MySQLDialect) AddForeignKey(name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) (string, error) {
	query := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)", d.QuoteTable(table), d.QuoteTable(name), columnList(d, columns), d.QuoteTable(refTable), columnList(d, refColumns))
	if onDelete != "" {
		query += " ON DELETE " + string(onDelete)
	}
	if onUpdate != "" {
		query += " ON UPDATE " + string(onUpdate)
	}
	return query, nil
}

// DropForeignKey builds DROP FOREIGN KEY SQL.
func (d MySQLDialect) DropForeignKey(name, table string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP FOREIGN KEY %s", d.QuoteTable(table), d.QuoteTable(name)), nil
}

// AddCommentOnColumn builds column comment SQL.
func (d MySQLDialect) AddCommentOnColumn(table, column, comment string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s COMMENT %s", d.QuoteTable(table), d.QuoteColumn(column), sqlLiteral(comment)), nil
}

// DropCommentFromColumn builds SQL for dropping a column comment.
func (d MySQLDialect) DropCommentFromColumn(table, column string) (string, error) {
	return d.AddCommentOnColumn(table, column, "")
}

// AddCommentOnTable builds table comment SQL.
func (d MySQLDialect) AddCommentOnTable(table, comment string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s COMMENT = %s", d.QuoteTable(table), sqlLiteral(comment)), nil
}

// DropCommentFromTable builds SQL for dropping a table comment.
func (d MySQLDialect) DropCommentFromTable(table string) (string, error) {
	return d.AddCommentOnTable(table, "")
}

// Insert builds INSERT SQL.
func (d MySQLDialect) Insert(table string, row Row) (SQLStatement, error) {
	return buildInsert(d, table, row)
}

// BatchInsert builds multi-row INSERT SQL.
func (d MySQLDialect) BatchInsert(table string, columns []string, rows [][]any) (SQLStatement, error) {
	return buildBatchInsert(d, table, columns, rows)
}

// Update builds UPDATE SQL.
func (d MySQLDialect) Update(table string, row Row, condition string, args ...any) (SQLStatement, error) {
	return buildUpdate(d, table, row, condition, args...)
}

// Delete builds DELETE SQL.
func (d MySQLDialect) Delete(table string, condition string, args ...any) (SQLStatement, error) {
	return buildDelete(d, table, condition, args...)
}

// AcquireLockSQL builds SQL for acquiring an advisory migration lock.
func (d MySQLDialect) AcquireLockSQL(lockName string, timeoutSeconds int) (SQLStatement, error) {
	return SQLStatement{Query: "SELECT GET_LOCK(?, ?)", Args: []any{lockName, timeoutSeconds}}, nil
}

// ReleaseLockSQL builds SQL for releasing an advisory migration lock.
func (d MySQLDialect) ReleaseLockSQL(lockName string) (SQLStatement, error) {
	return SQLStatement{Query: "SELECT RELEASE_LOCK(?)", Args: []any{lockName}}, nil
}
