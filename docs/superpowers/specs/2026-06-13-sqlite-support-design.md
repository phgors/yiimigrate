# SQLite Support Design

## Goal

Add first-class SQLite support to the migration library while preserving the Yii2-like developer experience and the existing `database/sql`-based architecture.

The repository currently contains only the package stub and a MySQL-first implementation document. This feature intentionally moves the project out of the prior MySQL-only phase. The supported dialect set for this phase becomes MySQL and SQLite.

## Scope

The implementation will add the migration core needed for dialect-backed SQL generation:

- immutable `ColumnBuilder` values;
- ordered `ColumnList` table definitions;
- a `Dialect` interface;
- `MigrationContext`;
- chainable `SchemaPlan` execution;
- DML helpers with parameter binding and raw SQL expressions;
- query helpers such as `RowExists` and `CountRows`;
- `MySQLDialect` and `SQLiteDialect`.

SQLite support must be real and documented. Unsupported SQLite DDL must return explicit errors during SQL generation instead of emitting fake or incomplete SQL.

## SQLite Dialect Behavior

`SQLiteDialect` will support:

- double-quote identifier quoting;
- `?` placeholders;
- table existence checks through `sqlite_master`;
- column existence checks through `PRAGMA table_info`;
- index existence checks through `sqlite_master`;
- `CREATE TABLE`, `DROP TABLE`, `ALTER TABLE ... RENAME TO`, and `ALTER TABLE ... ADD COLUMN`;
- `CREATE INDEX`, `CREATE UNIQUE INDEX`, and `DROP INDEX`;
- `INSERT`, `BATCH INSERT`, `UPDATE`, and `DELETE`;
- `RowExists` and `CountRows` SQL builders.

`SQLiteDialect` will return explicit unsupported-operation errors for:

- `ALTER COLUMN`;
- `DROP COLUMN`;
- `RENAME COLUMN`;
- primary key alteration after table creation;
- adding or dropping foreign keys after table creation;
- table and column comments;
- advisory migration locks.

Foreign key clauses are supported inside `CREATE TABLE` only when modeled as appended SQL on column definitions. SQLite's post-create foreign key changes are not supported.

## Column Type Mapping

SQLite uses dynamic typing, so builders map Yii-like types to SQLite affinities:

- primary keys: `INTEGER PRIMARY KEY AUTOINCREMENT`;
- integer family: `INTEGER`;
- string and text family: `VARCHAR(n)` or `TEXT`;
- binary and blob family: `BLOB`;
- boolean: `INTEGER`;
- floating-point types: `REAL`;
- decimal and money: `NUMERIC(precision, scale)`;
- date, time, datetime, timestamp, UUID, enum, and set: `TEXT`;
- JSON: `TEXT`.

SQLite does not support MySQL-only modifiers such as `UNSIGNED`, `AFTER`, `FIRST`, `CHARACTER SET`, or table options. The dialect will ignore harmless type decorations that have no SQLite meaning and reject clauses that would create misleading SQL.

## API

The public API remains simple:

```go
ctx := context.Background()
db, _ := sql.Open("sqlite3", ":memory:")
m := migrate.NewMigrationContext(db, migrate.SQLiteDialect{})

err := m.Schema().
	CreateTableIfNotExists(ctx, "article", migrate.Columns().
		Add("id", m.PrimaryKey()).
		Add("title", m.String(128).NotNull()).
		Add("metadata", m.Json().Null()).
		Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")),
	).
	CreateIndexIfNotExists(ctx, "idx-article-title", "article", []string{"title"}, false).
	Exec(ctx)
```

Query helpers work the same way across dialects:

```go
exists, err := m.RowExists(ctx, "article", "title = ?", "hello")
if err != nil {
	return err
}
if !exists {
	return m.Schema().
		Insert("article", migrate.Row{
			"title":      "hello",
			"created_at": migrate.Expr("CURRENT_TIMESTAMP"),
		}).
		Exec(ctx)
}
```

## Error Handling

SQL generation methods that can fail return `(string, error)` or `(SQLStatement, error)` through the dialect and are captured by `SchemaPlan`. `SchemaPlan.Exec` returns the first accumulated error before executing any later statement.

This keeps unsupported SQLite operations deterministic and testable. It also prevents dry-run mode from printing SQL that would never be valid.

## Testing

Unit tests must not require MySQL or SQLite drivers. Tests will cover SQL generation, column order, immutable builders, DML argument binding, expressions, query SQL builders, SQLite metadata SQL, and unsupported-operation errors.

`go test ./...` remains mandatory before finishing. Integration tests with real databases can be added later behind environment variables and optional drivers.

## Documentation Updates

`AGENTS.md` and `IMPLEMENTATION.md` must be updated so they no longer forbid SQLite. They should state that the current phase supports MySQL and SQLite, while PostgreSQL and SQL Server remain out of scope.
