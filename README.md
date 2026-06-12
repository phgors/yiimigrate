# yiimigrate

Yii2-inspired migrations for Go, built on `database/sql`.

Current phase: MySQL only. SQLite, PostgreSQL, and SQL Server are not implemented or advertised as supported.

## Install

```bash
go get github.com/phgors/yiimigrate
```

## Configure CLI

```bash
set DB_DSN=root:password@tcp(127.0.0.1:3306)/test?parseTime=true
set MIGRATE_TABLE=migration
```

Dry-run previews SQL without executing migration SQL:

```bash
set MIGRATE_DRY_RUN=1
go run ./cmd/migrate up
```

## Commands

```bash
go run ./cmd/migrate up [n]
go run ./cmd/migrate down [n]
go run ./cmd/migrate redo [n]
go run ./cmd/migrate history [n]
go run ./cmd/migrate new [n]
go run ./cmd/migrate mark VERSION
go run ./cmd/migrate to VERSION
go run ./cmd/migrate create NAME
go run ./cmd/migrate create create_article_table --fields="user_id:unsignedBigInteger:notNull,title:string(128):notNull,content:longText,metadata:json,status:unsignedTinyInteger:notNull:default(10)"
```

## Migration Example

```go
func (M20260613_120000CreateArticleTable) Up(ctx context.Context, m *migrate.MigrationContext) error {
	exists, err := m.RowExists(ctx, "article", "slug = ?", "hello")
	if err != nil {
		return err
	}

	plan := m.Schema().
		CreateTableIfNotExists(ctx, "article", migrate.Columns().
			Add("id", m.UnsignedBigPrimaryKey()).
			Add("user_id", m.UnsignedBigInteger().NotNull()).
			Add("title", m.String(128).NotNull().Unique()).
			Add("content", m.LongText().Null()).
			Add("metadata", m.Json().Null()).
			Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")),
			"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		).
		CreateIndexIfNotExists(ctx, "idx-article-user_id", "article", []string{"user_id"}, false)

	if !exists {
		plan.Insert("article", migrate.Row{
			"user_id": 1,
			"title": "Hello",
			"metadata": migrate.Expr("JSON_OBJECT('source', 'migration')"),
		})
	}

	return plan.Exec(ctx)
}
```

## Field Builders

| Yii2 style | Go |
| --- | --- |
| `$this->primaryKey()` | `m.PrimaryKey()` |
| `$this->bigPrimaryKey()->unsigned()` | `m.UnsignedBigPrimaryKey()` |
| `$this->string(128)->notNull()` | `m.String(128).NotNull()` |
| `$this->longText()->null()` | `m.LongText().Null()` |
| `$this->json()->null()` | `m.Json().Null()` |
| `$this->timestamp(0)->defaultExpression(...)` | `m.Timestamp(0).DefaultExpression(...)` |

## MySQL Notes

- Uses backtick quoting and `?` placeholders.
- Uses `information_schema` for table, column, index, foreign key, and constraint checks.
- Uses `GET_LOCK` and `RELEASE_LOCK` to serialize mutating migration commands.
- Unit tests do not require MySQL. Integration tests should be guarded by `MYSQL_TEST_DSN`.
