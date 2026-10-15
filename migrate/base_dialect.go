package migrate

import (
	"fmt"
	"strings"
)

// baseDialect carries the statement shapes every adapter shares. Each adapter
// embeds it, so the shared builders are reachable through method promotion,
// and passes itself as the Dialect argument so quoting, placeholders and
// column rendering dispatch through the embedding adapter's implementations.
// Go embedding is composition, not inheritance: a method on baseDialect never
// calls the outer adapter's overrides on its own, hence the explicit self
// parameter on every shared shape.
type baseDialect struct{}

// buildInsert renders an INSERT statement for one row. Column order is sorted
// so the generated SQL is deterministic.
func (baseDialect) buildInsert(d Dialect, table string, row Row) (SQLStatement, error) {
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

// buildBatchInsert renders one INSERT statement for many rows of values.
func (baseDialect) buildBatchInsert(d Dialect, table string, columns []string, rows [][]any) (SQLStatement, error) {
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

// buildUpdate renders an UPDATE statement for the columns in row; Expression
// values are inlined, plain values become bound arguments.
func (baseDialect) buildUpdate(d Dialect, table string, row Row, condition string, condArgs ...any) (SQLStatement, error) {
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

// buildDelete renders a DELETE statement for the rows matching condition.
func (baseDialect) buildDelete(d Dialect, table string, condition string, args ...any) (SQLStatement, error) {
	query := "DELETE FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return SQLStatement{Query: query, Args: append([]any(nil), args...)}, nil
}
