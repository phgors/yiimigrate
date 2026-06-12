package migrate

// DefaultMigrationTable is the default table used to record applied migrations.
const DefaultMigrationTable = "migration"

// Dialect builds SQL for a supported database engine.
type Dialect interface {
	// QuoteTable quotes a table identifier.
	QuoteTable(name string) string
	// QuoteColumn quotes a column identifier.
	QuoteColumn(name string) string
	// QuoteIndexColumn quotes an index column unless it is an expression.
	QuoteIndexColumn(name string) string
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
	// TableExistsSQL returns SQL and args for checking table existence.
	TableExistsSQL(table string) (string, []any)
	// ColumnExistsSQL returns SQL and args for checking column existence.
	ColumnExistsSQL(table, column string) (string, []any)
	// IndexExistsSQL returns SQL and args for checking index existence.
	IndexExistsSQL(table, index string) (string, []any)
	// ForeignKeyExistsSQL returns SQL and args for checking foreign key existence.
	ForeignKeyExistsSQL(table, name string) (string, []any)
	// ConstraintExistsSQL returns SQL and args for checking constraint existence.
	ConstraintExistsSQL(table, name string) (string, []any)
	// BuildRowExistsSQL returns SQL for checking row existence.
	BuildRowExistsSQL(table string, condition string) string
	// BuildCountRowsSQL returns SQL for counting rows.
	BuildCountRowsSQL(table string, condition string) string

	// CreateTable returns SQL that creates a table.
	CreateTable(table string, columns *ColumnList, options string) string
	// DropTable returns SQL that drops a table.
	DropTable(table string) string
	// RenameTable returns SQL that renames a table.
	RenameTable(oldName, newName string) string
	// TruncateTable returns SQL that truncates a table.
	TruncateTable(table string) string

	// AddColumn returns SQL that adds a column.
	AddColumn(table, column string, builder *ColumnBuilder) string
	// AlterColumn returns SQL that alters a column.
	AlterColumn(table, column string, builder *ColumnBuilder) string
	// DropColumn returns SQL that drops a column.
	DropColumn(table, column string) string
	// RenameColumn returns SQL that renames a column.
	RenameColumn(table, oldName, newName string) string

	// CreateIndex returns SQL that creates an index.
	CreateIndex(name, table string, columns []string, unique bool) string
	// DropIndex returns SQL that drops an index.
	DropIndex(name, table string) string

	// AddPrimaryKey returns SQL that adds a primary key constraint.
	AddPrimaryKey(name, table string, columns []string) string
	// DropPrimaryKey returns SQL that drops the primary key constraint.
	DropPrimaryKey(name, table string) string

	// AddForeignKey returns SQL that adds a foreign key constraint.
	AddForeignKey(name string, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) string
	// DropForeignKey returns SQL that drops a foreign key constraint.
	DropForeignKey(name, table string) string

	// AddCommentOnColumn returns SQL that adds a column comment.
	AddCommentOnColumn(table, column, comment string) string
	// DropCommentFromColumn returns SQL that drops a column comment.
	DropCommentFromColumn(table, column string) string
	// AddCommentOnTable returns SQL that adds a table comment.
	AddCommentOnTable(table, comment string) string
	// DropCommentFromTable returns SQL that drops a table comment.
	DropCommentFromTable(table string) string

	// Insert returns SQL and args for an INSERT statement.
	Insert(table string, row Row) (string, []any)
	// BatchInsert returns SQL and args for a multi-row INSERT statement.
	BatchInsert(table string, columns []string, rows [][]any) (string, []any)
	// Update returns SQL and args for an UPDATE statement.
	Update(table string, row Row, condition string, args ...any) (string, []any)
	// Delete returns SQL and args for a DELETE statement.
	Delete(table string, condition string, args ...any) (string, []any)
	// AcquireLockSQL returns SQL and args for acquiring a migration lock.
	AcquireLockSQL(lockName string, timeoutSeconds int) (string, []any)
	// ReleaseLockSQL returns SQL and args for releasing a migration lock.
	ReleaseLockSQL(lockName string) (string, []any)
}
