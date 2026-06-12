package migrate

import "context"

// TableExists reports whether a table exists.
func (m *MigrationContext) TableExists(ctx context.Context, table string) (bool, error) {
	query, args := m.dialect.TableExistsSQL(table)
	return m.exists(ctx, query, args...)
}

// ColumnExists reports whether a column exists.
func (m *MigrationContext) ColumnExists(ctx context.Context, table, column string) (bool, error) {
	query, args := m.dialect.ColumnExistsSQL(table, column)
	return m.exists(ctx, query, args...)
}

// IndexExists reports whether an index exists.
func (m *MigrationContext) IndexExists(ctx context.Context, table, index string) (bool, error) {
	query, args := m.dialect.IndexExistsSQL(table, index)
	return m.exists(ctx, query, args...)
}

// ForeignKeyExists reports whether a foreign key exists.
func (m *MigrationContext) ForeignKeyExists(ctx context.Context, table, name string) (bool, error) {
	query, args := m.dialect.ForeignKeyExistsSQL(table, name)
	return m.exists(ctx, query, args...)
}

// ConstraintExists reports whether a constraint exists.
func (m *MigrationContext) ConstraintExists(ctx context.Context, table, name string) (bool, error) {
	query, args := m.dialect.ConstraintExistsSQL(table, name)
	return m.exists(ctx, query, args...)
}

func (m *MigrationContext) exists(ctx context.Context, query string, args ...any) (bool, error) {
	value, err := m.QueryValue(ctx, query, args...)
	if err != nil {
		return false, err
	}
	count, ok := toInt64(value)
	if !ok {
		return false, nil
	}
	return count > 0, nil
}
