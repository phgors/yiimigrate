package migrate

import (
	"fmt"
	"sort"
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

// QuoteIndexColumn quotes a MySQL index column unless it is an expression.
func (d MySQLDialect) QuoteIndexColumn(name string) string {
	column, direction := splitIndexDirection(strings.TrimSpace(name))
	if column == "" {
		return name
	}
	quoted := d.quoteIndexColumnCore(column)
	if direction != "" {
		return quoted + " " + direction
	}
	return quoted
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

// TableExistsSQL returns SQL and args for checking MySQL table existence.
func (MySQLDialect) TableExistsSQL(table string) (string, []any) {
	return "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", []any{table}
}

// ColumnExistsSQL returns SQL and args for checking MySQL column existence.
func (MySQLDialect) ColumnExistsSQL(table, column string) (string, []any) {
	return "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?", []any{table, column}
}

// IndexExistsSQL returns SQL and args for checking MySQL index existence.
func (MySQLDialect) IndexExistsSQL(table, index string) (string, []any) {
	return "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?", []any{table, index}
}

// ForeignKeyExistsSQL returns SQL and args for checking MySQL foreign key existence.
func (MySQLDialect) ForeignKeyExistsSQL(table, name string) (string, []any) {
	return "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = ? AND constraint_type = 'FOREIGN KEY'", []any{table, name}
}

// ConstraintExistsSQL returns SQL and args for checking MySQL constraint existence.
func (MySQLDialect) ConstraintExistsSQL(table, name string) (string, []any) {
	return "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = DATABASE() AND table_name = ? AND constraint_name = ?", []any{table, name}
}

// BuildRowExistsSQL returns SQL for checking row existence.
func (d MySQLDialect) BuildRowExistsSQL(table string, condition string) string {
	sql := "SELECT EXISTS(SELECT 1 FROM " + d.QuoteTable(table)
	if condition != "" {
		sql += " WHERE " + condition
	}
	return sql + ")"
}

// BuildCountRowsSQL returns SQL for counting rows.
func (d MySQLDialect) BuildCountRowsSQL(table string, condition string) string {
	sql := "SELECT COUNT(*) FROM " + d.QuoteTable(table)
	if condition != "" {
		sql += " WHERE " + condition
	}
	return sql
}

// CreateTable returns SQL that creates a MySQL table.
func (d MySQLDialect) CreateTable(table string, columns *ColumnList, options string) string {
	definitions := make([]string, 0, len(columns.Items()))
	for _, column := range columns.Items() {
		definitions = append(definitions, d.QuoteColumn(column.Name)+" "+d.columnSQL(column.Column))
	}

	sql := "CREATE TABLE " + d.QuoteTable(table) + " (" + strings.Join(definitions, ", ") + ")"
	if options != "" {
		sql += " " + options
	}
	return sql
}

// DropTable returns SQL that drops a MySQL table.
func (d MySQLDialect) DropTable(table string) string {
	return "DROP TABLE " + d.QuoteTable(table)
}

// RenameTable returns SQL that renames a MySQL table.
func (d MySQLDialect) RenameTable(oldName, newName string) string {
	return "RENAME TABLE " + d.QuoteTable(oldName) + " TO " + d.QuoteTable(newName)
}

// TruncateTable returns SQL that truncates a MySQL table.
func (d MySQLDialect) TruncateTable(table string) string {
	return "TRUNCATE TABLE " + d.QuoteTable(table)
}

// AddColumn returns SQL that adds a MySQL column.
func (d MySQLDialect) AddColumn(table, column string, builder *ColumnBuilder) string {
	return "ALTER TABLE " + d.QuoteTable(table) + " ADD COLUMN " + d.QuoteColumn(column) + " " + d.columnSQL(builder)
}

// AlterColumn returns SQL that modifies a MySQL column.
func (d MySQLDialect) AlterColumn(table, column string, builder *ColumnBuilder) string {
	return "ALTER TABLE " + d.QuoteTable(table) + " MODIFY COLUMN " + d.QuoteColumn(column) + " " + d.columnSQL(builder)
}

// DropColumn returns SQL that drops a MySQL column.
func (d MySQLDialect) DropColumn(table, column string) string {
	return "ALTER TABLE " + d.QuoteTable(table) + " DROP COLUMN " + d.QuoteColumn(column)
}

// RenameColumn returns SQL that renames a MySQL column.
func (d MySQLDialect) RenameColumn(table, oldName, newName string) string {
	return "ALTER TABLE " + d.QuoteTable(table) + " RENAME COLUMN " + d.QuoteColumn(oldName) + " TO " + d.QuoteColumn(newName)
}

// CreateIndex returns SQL that creates a MySQL index.
func (d MySQLDialect) CreateIndex(name, table string, columns []string, unique bool) string {
	prefix := "CREATE INDEX "
	if unique {
		prefix = "CREATE UNIQUE INDEX "
	}
	return prefix + d.QuoteColumn(name) + " ON " + d.QuoteTable(table) + " (" + d.joinIndexColumns(columns) + ")"
}

// DropIndex returns SQL that drops a MySQL index.
func (d MySQLDialect) DropIndex(name, table string) string {
	return "DROP INDEX " + d.QuoteColumn(name) + " ON " + d.QuoteTable(table)
}

// AddPrimaryKey returns SQL that adds a MySQL primary key constraint.
func (d MySQLDialect) AddPrimaryKey(name, table string, columns []string) string {
	return "ALTER TABLE " + d.QuoteTable(table) + " ADD CONSTRAINT " + d.QuoteColumn(name) + " PRIMARY KEY (" + d.joinColumns(columns) + ")"
}

// DropPrimaryKey returns SQL that drops a MySQL primary key.
func (d MySQLDialect) DropPrimaryKey(name, table string) string {
	return "ALTER TABLE " + d.QuoteTable(table) + " DROP PRIMARY KEY"
}

// AddForeignKey returns SQL that adds a MySQL foreign key constraint.
func (d MySQLDialect) AddForeignKey(name string, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) string {
	sql := "ALTER TABLE " + d.QuoteTable(table) + " ADD CONSTRAINT " + d.QuoteColumn(name) +
		" FOREIGN KEY (" + d.joinColumns(columns) + ") REFERENCES " + d.QuoteTable(refTable) +
		" (" + d.joinColumns(refColumns) + ")"
	if onDelete != "" {
		sql += " ON DELETE " + onDelete.sql()
	}
	if onUpdate != "" {
		sql += " ON UPDATE " + onUpdate.sql()
	}
	return sql
}

// DropForeignKey returns SQL that drops a MySQL foreign key.
func (d MySQLDialect) DropForeignKey(name, table string) string {
	return "ALTER TABLE " + d.QuoteTable(table) + " DROP FOREIGN KEY " + d.QuoteColumn(name)
}

// AddCommentOnColumn returns SQL that adds a MySQL column comment.
func (d MySQLDialect) AddCommentOnColumn(table, column, comment string) string {
	return "ALTER TABLE " + d.QuoteTable(table) + " MODIFY COLUMN " + d.QuoteColumn(column) + " COMMENT " + quoteSQLString(comment)
}

// DropCommentFromColumn returns SQL that drops a MySQL column comment.
func (d MySQLDialect) DropCommentFromColumn(table, column string) string {
	return d.AddCommentOnColumn(table, column, "")
}

// AddCommentOnTable returns SQL that adds a MySQL table comment.
func (d MySQLDialect) AddCommentOnTable(table, comment string) string {
	return "ALTER TABLE " + d.QuoteTable(table) + " COMMENT = " + quoteSQLString(comment)
}

// DropCommentFromTable returns SQL that drops a MySQL table comment.
func (d MySQLDialect) DropCommentFromTable(table string) string {
	return d.AddCommentOnTable(table, "")
}

// Insert returns SQL and args for a MySQL INSERT statement.
func (d MySQLDialect) Insert(table string, row Row) (string, []any) {
	columns := sortedRowColumns(row)
	quotedColumns := make([]string, 0, len(columns))
	values := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, d.QuoteColumn(column))
		values = appendSQLValue(values, &args, row[column], len(args)+1, d)
	}
	return "INSERT INTO " + d.QuoteTable(table) + " (" + strings.Join(quotedColumns, ", ") + ") VALUES (" + strings.Join(values, ", ") + ")", args
}

// BatchInsert returns SQL and args for a MySQL multi-row INSERT statement.
func (d MySQLDialect) BatchInsert(table string, columns []string, rows [][]any) (string, []any) {
	quotedColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, d.QuoteColumn(column))
	}
	args := make([]any, 0, len(columns)*len(rows))
	valueGroups := make([]string, 0, len(rows))
	for _, row := range rows {
		values := make([]string, 0, len(row))
		for _, value := range row {
			values = appendSQLValue(values, &args, value, len(args)+1, d)
		}
		valueGroups = append(valueGroups, "("+strings.Join(values, ", ")+")")
	}
	return "INSERT INTO " + d.QuoteTable(table) + " (" + strings.Join(quotedColumns, ", ") + ") VALUES " + strings.Join(valueGroups, ", "), args
}

// Update returns SQL and args for a MySQL UPDATE statement.
func (d MySQLDialect) Update(table string, row Row, condition string, conditionArgs ...any) (string, []any) {
	columns := sortedRowColumns(row)
	set := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns)+len(conditionArgs))
	for _, column := range columns {
		values := appendSQLValue(nil, &args, row[column], len(args)+1, d)
		set = append(set, d.QuoteColumn(column)+" = "+values[0])
	}
	args = append(args, conditionArgs...)
	sql := "UPDATE " + d.QuoteTable(table) + " SET " + strings.Join(set, ", ")
	if condition != "" {
		sql += " WHERE " + condition
	}
	return sql, args
}

// Delete returns SQL and args for a MySQL DELETE statement.
func (d MySQLDialect) Delete(table string, condition string, args ...any) (string, []any) {
	sql := "DELETE FROM " + d.QuoteTable(table)
	if condition != "" {
		sql += " WHERE " + condition
	}
	return sql, args
}

// AcquireLockSQL returns SQL and args for acquiring a MySQL advisory lock.
func (MySQLDialect) AcquireLockSQL(lockName string, timeoutSeconds int) (string, []any) {
	return "SELECT GET_LOCK(?, ?)", []any{lockName, timeoutSeconds}
}

// ReleaseLockSQL returns SQL and args for releasing a MySQL advisory lock.
func (MySQLDialect) ReleaseLockSQL(lockName string) (string, []any) {
	return "SELECT RELEASE_LOCK(?)", []any{lockName}
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

func sortedRowColumns(row Row) []string {
	columns := make([]string, 0, len(row))
	for column := range row {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	return columns
}

func appendSQLValue(values []string, args *[]any, value any, index int, dialect Dialect) []string {
	if expr, ok := value.(Expression); ok {
		return append(values, string(expr))
	}
	values = append(values, dialect.Placeholder(index))
	*args = append(*args, value)
	return values
}

func (d MySQLDialect) joinColumns(columns []string) string {
	quoted := make([]string, 0, len(columns))
	for _, column := range columns {
		quoted = append(quoted, d.QuoteColumn(column))
	}
	return strings.Join(quoted, ", ")
}

func (d MySQLDialect) joinIndexColumns(columns []string) string {
	quoted := make([]string, 0, len(columns))
	for _, column := range columns {
		quoted = append(quoted, d.QuoteIndexColumn(column))
	}
	return strings.Join(quoted, ", ")
}

func (d MySQLDialect) quoteIndexColumnCore(column string) string {
	if strings.HasPrefix(column, "`") {
		return column
	}
	if base, length, ok := splitPrefixLength(column); ok {
		return d.QuoteColumn(base) + "(" + length + ")"
	}
	if strings.ContainsAny(column, "()") {
		return column
	}
	return d.QuoteColumn(column)
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

func splitIndexDirection(column string) (string, string) {
	upper := strings.ToUpper(column)
	for _, direction := range []string{" ASC", " DESC"} {
		if strings.HasSuffix(upper, direction) {
			return strings.TrimSpace(column[:len(column)-len(direction)]), strings.TrimSpace(direction)
		}
	}
	return column, ""
}

func splitPrefixLength(column string) (string, string, bool) {
	if !strings.HasSuffix(column, ")") {
		return "", "", false
	}
	open := strings.LastIndex(column, "(")
	if open <= 0 {
		return "", "", false
	}
	length := column[open+1 : len(column)-1]
	if length == "" {
		return "", "", false
	}
	for _, r := range length {
		if r < '0' || r > '9' {
			return "", "", false
		}
	}
	return column[:open], length, true
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
