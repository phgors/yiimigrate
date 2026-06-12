# AGENTS.md

## Project

This repository implements a Go migration library inspired by Yii2 Migration.

## Language

Use Go.

## Requirements

- Always run `gofmt` on changed Go files.
- Always run `go test ./...` before finishing.
- Prefer small, focused commits.
- Keep public APIs simple and documented.
- Do not introduce ORM dependencies.
- Use `database/sql`.
- MySQL is the only supported dialect in the current phase.
- Do not implement fake or incomplete SQLite, PostgreSQL, or SQL Server dialects.
- Preserve Yii2-like developer experience where possible.

## Style

- Public types and public methods need Go doc comments.
- Return errors instead of panics.
- Use context-aware database methods.
- Keep SQL generation covered by unit tests.
- Avoid global mutable state.

## Testing

- Unit tests must not require MySQL.
- MySQL integration tests should run only when `MYSQL_TEST_DSN` is set.
- Do not skip `go test ./...`.

## API Design

The migration API should support this style:

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

The migration API should also support query helpers:

```go
exists, err := m.RowExists(ctx, "user", "username = ?", "admin")
if err != nil {
	return err
}

if !exists {
	return m.Schema().
		Insert("user", migrate.Row{
			"username": "admin",
			"created_at": migrate.Expr("UNIX_TIMESTAMP()"),
		}).
		Exec(ctx)
}
```

## Do Not

- Do not replace `database/sql` with GORM.
- Do not make ColumnBuilder mutable.
- Do not use unordered map iteration for CREATE TABLE column order.
- Do not execute SQL in dry-run mode.
- Do not quote SQL expressions like `LOWER(email)` as normal column names.
- Do not implement SQLiteDialect, PostgresDialect, or SQLServerDialect in the current phase.
