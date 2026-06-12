package migrate

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Dialect generates SQL for a specific database engine.
type Dialect interface {
	Name() string
	QuoteTable(name string) string
	QuoteColumn(name string) string
	QuoteIndexColumn(name string) string
	Placeholder(index int) string
	BuildColumn(column *ColumnBuilder) (string, error)
	TableExistsSQL(table string) SQLStatement
	ColumnExistsSQL(table, column string) SQLStatement
	IndexExistsSQL(table, index string) SQLStatement
	ForeignKeyExistsSQL(table, name string) SQLStatement
	ConstraintExistsSQL(table, name string) SQLStatement
	BuildRowExistsSQL(table string, condition string) string
	BuildCountRowsSQL(table string, condition string) string
	CreateTable(table string, columns *ColumnList, options string) (string, error)
	DropTable(table string) (string, error)
	RenameTable(oldName, newName string) (string, error)
	TruncateTable(table string) (string, error)
	AddColumn(table, column string, builder *ColumnBuilder) (string, error)
	AlterColumn(table, column string, builder *ColumnBuilder) (string, error)
	DropColumn(table, column string) (string, error)
	RenameColumn(table, oldName, newName string) (string, error)
	CreateIndex(name, table string, columns []string, unique bool) (string, error)
	DropIndex(name, table string) (string, error)
	AddPrimaryKey(name, table string, columns []string) (string, error)
	DropPrimaryKey(name, table string) (string, error)
	AddForeignKey(name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) (string, error)
	DropForeignKey(name, table string) (string, error)
	AddCommentOnColumn(table, column, comment string) (string, error)
	DropCommentFromColumn(table, column string) (string, error)
	AddCommentOnTable(table, comment string) (string, error)
	DropCommentFromTable(table string) (string, error)
	Insert(table string, row Row) (SQLStatement, error)
	BatchInsert(table string, columns []string, rows [][]any) (SQLStatement, error)
	Update(table string, row Row, condition string, args ...any) (SQLStatement, error)
	Delete(table string, condition string, args ...any) (SQLStatement, error)
	AcquireLockSQL(lockName string, timeoutSeconds int) (SQLStatement, error)
	ReleaseLockSQL(lockName string) (SQLStatement, error)
}

// ForeignKeyAction is an ON DELETE or ON UPDATE action.
type ForeignKeyAction string

const (
	// Cascade maps to CASCADE.
	Cascade ForeignKeyAction = "CASCADE"
	// Restrict maps to RESTRICT.
	Restrict ForeignKeyAction = "RESTRICT"
	// SetNull maps to SET NULL.
	SetNull ForeignKeyAction = "SET NULL"
	// NoAction maps to NO ACTION.
	NoAction ForeignKeyAction = "NO ACTION"
)

func quoteName(name, quote string) string {
	if name == "*" {
		return name
	}
	parts := strings.Split(name, ".")
	for i, part := range parts {
		parts[i] = quote + strings.ReplaceAll(part, quote, quote+quote) + quote
	}
	return strings.Join(parts, ".")
}

func quoteIndexColumn(d Dialect, name string) string {
	if strings.ContainsAny(name, "() ") || strings.Contains(name, "->") {
		return name
	}
	return d.QuoteColumn(name)
}

func columnList(d Dialect, columns []string) string {
	quoted := make([]string, 0, len(columns))
	for _, column := range columns {
		quoted = append(quoted, d.QuoteIndexColumn(column))
	}
	return strings.Join(quoted, ", ")
}

func joinOptions(options string) string {
	options = strings.TrimSpace(options)
	if options == "" {
		return ""
	}
	return " " + options
}

func sqlLiteral(v any) string {
	switch value := v.(type) {
	case nil:
		return "NULL"
	case Expression:
		return string(value)
	case string:
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	case bool:
		if value {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprint(value)
	}
}

func sizedType(name string, size []int, defaults ...int) string {
	if len(size) == 0 {
		if len(defaults) == 0 {
			return name
		}
		return name + "(" + strconv.Itoa(defaults[0]) + ")"
	}
	if len(size) == 1 {
		return name + "(" + strconv.Itoa(size[0]) + ")"
	}
	return name + "(" + strconv.Itoa(size[0]) + ", " + strconv.Itoa(size[1]) + ")"
}

func enumValues(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, sqlLiteral(value))
	}
	return strings.Join(quoted, ", ")
}

func sortedRowKeys(row Row) []string {
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func buildInsert(d Dialect, table string, row Row) (SQLStatement, error) {
	if len(row) == 0 {
		return SQLStatement{}, fmt.Errorf("migrate: insert row is empty")
	}
	keys := sortedRowKeys(row)
	columns := make([]string, 0, len(keys))
	values := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for _, key := range keys {
		columns = append(columns, d.QuoteColumn(key))
		switch value := row[key].(type) {
		case Expression:
			values = append(values, string(value))
		default:
			args = append(args, value)
			values = append(values, d.Placeholder(len(args)))
		}
	}
	return SQLStatement{
		Query: fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", d.QuoteTable(table), strings.Join(columns, ", "), strings.Join(values, ", ")),
		Args:  args,
	}, nil
}

func buildBatchInsert(d Dialect, table string, columns []string, rows [][]any) (SQLStatement, error) {
	if len(columns) == 0 {
		return SQLStatement{}, fmt.Errorf("migrate: batch insert columns are empty")
	}
	if len(rows) == 0 {
		return SQLStatement{}, fmt.Errorf("migrate: batch insert rows are empty")
	}
	quotedColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, d.QuoteColumn(column))
	}
	args := make([]any, 0, len(columns)*len(rows))
	valueGroups := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) != len(columns) {
			return SQLStatement{}, fmt.Errorf("migrate: batch insert row has %d values for %d columns", len(row), len(columns))
		}
		values := make([]string, 0, len(row))
		for _, value := range row {
			if expr, ok := value.(Expression); ok {
				values = append(values, string(expr))
				continue
			}
			args = append(args, value)
			values = append(values, d.Placeholder(len(args)))
		}
		valueGroups = append(valueGroups, "("+strings.Join(values, ", ")+")")
	}
	return SQLStatement{
		Query: fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", d.QuoteTable(table), strings.Join(quotedColumns, ", "), strings.Join(valueGroups, ", ")),
		Args:  args,
	}, nil
}

func buildUpdate(d Dialect, table string, row Row, condition string, condArgs ...any) (SQLStatement, error) {
	if len(row) == 0 {
		return SQLStatement{}, fmt.Errorf("migrate: update row is empty")
	}
	keys := sortedRowKeys(row)
	args := make([]any, 0, len(row)+len(condArgs))
	sets := make([]string, 0, len(keys))
	for _, key := range keys {
		if expr, ok := row[key].(Expression); ok {
			sets = append(sets, d.QuoteColumn(key)+" = "+string(expr))
			continue
		}
		args = append(args, row[key])
		sets = append(sets, d.QuoteColumn(key)+" = "+d.Placeholder(len(args)))
	}
	query := fmt.Sprintf("UPDATE %s SET %s", d.QuoteTable(table), strings.Join(sets, ", "))
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
		args = append(args, condArgs...)
	}
	return SQLStatement{Query: query, Args: args}, nil
}

func buildDelete(d Dialect, table string, condition string, args ...any) (SQLStatement, error) {
	query := "DELETE FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return SQLStatement{Query: query, Args: append([]any(nil), args...)}, nil
}
