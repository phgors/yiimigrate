package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// QueryValue returns the first column from the first row.
func (m *MigrationContext) QueryValue(ctx context.Context, query string, args ...any) (any, error) {
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var value any
	if err := rows.Scan(&value); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return normalizeDBValue(value), nil
}

// QueryOne returns the first row as a Row.
func (m *MigrationContext) QueryOne(ctx context.Context, query string, args ...any) (Row, error) {
	rows, err := m.QueryAll(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// QueryAll returns all rows as Row values.
func (m *MigrationContext) QueryAll(ctx context.Context, query string, args ...any) ([]Row, error) {
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := make([]Row, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		row := make(Row, len(columns))
		for i, column := range columns {
			row[column] = normalizeDBValue(values[i])
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// RowExists reports whether a row matching condition exists.
func (m *MigrationContext) RowExists(ctx context.Context, table string, condition string, args ...any) (bool, error) {
	value, err := m.QueryValue(ctx, m.dialect.BuildRowExistsSQL(table, condition), args...)
	if err != nil {
		return false, err
	}
	count, ok := toInt64(value)
	return ok && count > 0, nil
}

// CountRows counts rows matching condition.
func (m *MigrationContext) CountRows(ctx context.Context, table string, condition string, args ...any) (int64, error) {
	value, err := m.QueryValue(ctx, m.dialect.BuildCountRowsSQL(table, condition), args...)
	if err != nil {
		return 0, err
	}
	count, ok := toInt64(value)
	if !ok {
		return 0, fmt.Errorf("count query returned %T", value)
	}
	return count, nil
}

func normalizeDBValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	default:
		return v
	}
}

func toInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case []byte:
		var out int64
		if _, err := fmt.Sscan(string(v), &out); err == nil {
			return out, true
		}
	case string:
		var out int64
		if _, err := fmt.Sscan(v, &out); err == nil {
			return out, true
		}
	case nil:
		return 0, false
	case sql.NullInt64:
		return v.Int64, v.Valid
	}
	return 0, false
}
