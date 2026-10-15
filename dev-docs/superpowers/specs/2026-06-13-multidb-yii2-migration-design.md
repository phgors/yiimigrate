# Multi-Database Yii2 Migration Support Design

## Goal

Adapt the migration library toward Yii2-style multi-database support while keeping the existing `database/sql` foundation and Yii2-like developer experience.

The project already has real MySQL and SQLite dialects. The next direction is to make the dialect layer, documentation, tests, and CLI shape ready for additional real database dialects such as PostgreSQL and SQL Server. A dialect should be public only when it has concrete SQL generation behavior and tests. If a database cannot support an operation, the dialect must return a clear unsupported-operation error.

## Scope

This design covers the multi-database architecture and support policy:

- keep `database/sql` as the only database abstraction;
- keep the chainable `MigrationContext` and `SchemaPlan` APIs;
- keep `ColumnBuilder` immutable;
- preserve ordered `ColumnList` output for `CREATE TABLE`;
- make dialect support explicit and testable;
- let each dialect define identifier quoting, placeholders, column type mapping, metadata queries, DDL, DML, locking, and unsupported operations;
- document which databases are implemented and which are planned;
- avoid claiming support for a database before the dialect is real and covered by tests.

## Non-Goals

This design does not introduce an ORM, schema diff engine, GUI, or global mutable dialect registry.

It also does not require every database to support every Yii2 migration operation. Yii2-like developer experience means a common API where possible and clear dialect errors where a database lacks a feature.

## Dialect Contract

`Dialect` remains the boundary for SQL generation. Each implementation is responsible for:

- `Name`;
- table, column, and index-column quoting;
- positional placeholder formatting;
- column SQL generation from `ColumnBuilder`;
- table, column, index, foreign key, and constraint existence SQL;
- row existence and row count SQL;
- DDL statements used by `SchemaPlan`;
- DML statements with parameter binding and raw `Expression` support;
- advisory lock SQL when the database has a suitable lock primitive.

Unsupported operations must fail during SQL generation. `SchemaPlan` already records the first generation error and returns it from `Exec` before executing later statements.

## Dialect Capability Policy

The library should treat support as a per-database capability, not as a promise that every database behaves identically.

Examples:

- MySQL supports table options and table or column comments.
- SQLite supports `CREATE TABLE`, `ADD COLUMN`, indexes, and DML, but many post-create table alterations return unsupported-operation errors.
- PostgreSQL can later use `$1`, `$2`, `$3` placeholders and PostgreSQL catalog metadata queries.
- SQL Server can later use SQL Server identifier quoting, metadata queries, and locking behavior.

The README should list implemented dialects separately from planned dialects. A planned dialect is not supported until code and tests exist.

## Public API

The migration API remains Yii2-like:

```go
return m.Schema().
	CreateTableIfNotExists(ctx, "article", migrate.Columns().
		Add("id", m.UnsignedBigPrimaryKey()).
		Add("user_id", m.UnsignedBigInteger().NotNull()).
		Add("title", m.String(128).NotNull()).
		Add("content", m.LongText().Null()).
		Add("metadata", m.Json().Null()).
		Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")),
		"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	).
	CreateIndexIfNotExists(ctx, "idx-article-user_id", "article", []string{"user_id"}, false).
	Exec(ctx)
```

Query helpers remain immediate helpers on `MigrationContext`:

```go
exists, err := m.RowExists(ctx, "user", "username = ?", "admin")
if err != nil {
	return err
}

if !exists {
	return m.Schema().
		Insert("user", migrate.Row{
			"username":   "admin",
			"created_at": migrate.Expr("UNIX_TIMESTAMP()"),
		}).
		Exec(ctx)
}
```

## CLI Direction

The CLI should not be hard-coded to MySQL forever. It should select an implemented dialect through explicit configuration, for example an environment variable such as `DB_DIALECT`.

The initial valid values should match implemented dialects. Unknown values should fail with a clear error listing supported values.

## Testing

Unit tests must cover SQL generation without requiring external database services.

For every public dialect:

- quote behavior;
- placeholders;
- column type mapping;
- `CREATE TABLE` column order;
- DDL generation;
- DML generation and bound arguments;
- expression handling;
- metadata existence SQL;
- unsupported-operation errors.

Integration tests should remain opt-in behind environment variables such as `MYSQL_TEST_DSN`. Future database integration tests should follow the same pattern.

## Documentation

`AGENTS.md` and `IMPLEMENTATION.md` should describe the multi-database direction instead of restricting the project to the current database pair.

`README.md` should distinguish:

- implemented and tested dialects;
- planned dialects;
- database-specific notes;
- unsupported operations that return errors.

## Acceptance Criteria

The next implementation phase is complete when:

- database support restrictions are removed from project instructions;
- the support policy requires real, tested dialects;
- docs no longer forbid PostgreSQL or SQL Server implementation;
- future dialect work has a clear capability and testing standard;
- `go test ./...` passes.
