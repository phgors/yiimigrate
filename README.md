# yiimigrate

`yiimigrate` 是一个受 Yii2 Migration 启发的 Go 数据库迁移库。它使用 Go 标准库 `database/sql`，提供接近 Yii2 的字段声明、Schema 操作、迁移执行、dry-run、迁移锁和查询辅助方法。

当前阶段仅支持 MySQL。SQLite、PostgreSQL、SQL Server 尚未实现，也不会在当前版本中以半成品方言暴露。

## 特性

- 基于 `migration` 表记录已执行迁移。
- 支持 `up`、`down`、`redo`、`history`、`new`、`mark`、`to`、`create` 命令。
- 默认用事务执行每个 migration。
- MySQL 下使用 `GET_LOCK` / `RELEASE_LOCK` 串行化迁移。
- 支持 dry-run，预览 SQL 但不执行迁移 SQL。
- 支持不可变链式 `ColumnBuilder`。
- 支持有序 `ColumnList`，保证 `CREATE TABLE` 字段顺序稳定。
- 支持链式 `SchemaPlan` 执行 DDL 和 DML。
- 支持 `RowExists`、`CountRows`、`QueryValue`、`QueryOne`、`QueryAll` 等迁移内查询 helper。
- DML 普通值使用参数绑定，`migrate.Expr(...)` 作为 SQL 表达式直接写入。
- 单元测试不需要 MySQL；MySQL 集成测试仅在 `MYSQL_TEST_DSN` 设置时运行。

## 安装

```bash
go get github.com/phgors/yiimigrate
```

CLI 使用 MySQL 驱动：

```go
import _ "github.com/go-sql-driver/mysql"
```

## 快速开始

创建迁移：

```bash
go run ./cmd/migrate create create_article_table \
  --fields="user_id:unsignedBigInteger:notNull,title:string(128):notNull,content:longText,metadata:json,status:unsignedTinyInteger:notNull:default(10)"
```

执行迁移：

```bash
set DB_DSN=root:password@tcp(127.0.0.1:3306)/test?parseTime=true
go run ./cmd/migrate up
```

回滚最近一个迁移：

```bash
set DB_DSN=root:password@tcp(127.0.0.1:3306)/test?parseTime=true
go run ./cmd/migrate down 1
```

预览 SQL：

```bash
set MIGRATE_DRY_RUN=1
set DB_DSN=root:password@tcp(127.0.0.1:3306)/test?parseTime=true
go run ./cmd/migrate up
```

## 迁移注册

项目默认提供 `migrations/register.go`。将生成的迁移类型注册到 `All()`：

```go
package migrations

import "github.com/phgors/yiimigrate/migrate"

func All() []migrate.Migration {
	return []migrate.Migration{
		M20260613_120000CreateArticleTable{},
	}
}
```

CLI 会通过 `migrations.All()` 获取迁移列表。

## 完整迁移示例

```go
package migrations

import (
	"context"

	"github.com/phgors/yiimigrate/migrate"
)

type M20260613_120000CreateArticleTable struct{}

func (M20260613_120000CreateArticleTable) Name() string {
	return "m20260613_120000_create_article_table"
}

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
			Add("slug", m.String(128).NotNull().Unique()).
			Add("content", m.LongText().Null()).
			Add("metadata", m.Json().Null()).
			Add("status", m.UnsignedTinyInteger().NotNull().DefaultValue(10)).
			Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")),
			"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		).
		CreateIndexIfNotExists(ctx, "idx-article-user_id", "article", []string{"user_id"}, false).
		AddForeignKeyIfNotExists(
			ctx,
			"fk-article-user_id",
			"article",
			[]string{"user_id"},
			"user",
			[]string{"id"},
			migrate.Cascade,
			migrate.Cascade,
		)

	if !exists {
		plan.Insert("article", migrate.Row{
			"user_id":  1,
			"title":    "Hello",
			"slug":     "hello",
			"content":  "Hello world",
			"metadata": migrate.Expr("JSON_OBJECT('source', 'migration')"),
			"status":   10,
		})
	}

	return plan.Exec(ctx)
}

func (M20260613_120000CreateArticleTable) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		DropForeignKeyIfExists(ctx, "fk-article-user_id", "article").
		DropIndexIfExists(ctx, "idx-article-user_id", "article").
		DropTableIfExists(ctx, "article").
		Exec(ctx)
}
```

## CLI

环境变量：

| 变量 | 说明 |
| --- | --- |
| `DB_DSN` | MySQL DSN，例如 `root:password@tcp(127.0.0.1:3306)/test?parseTime=true` |
| `MIGRATE_DRY_RUN` | 设置为 `1`、`true`、`TRUE` 或 `yes` 时启用 dry-run |
| `MIGRATE_TABLE` | 自定义迁移记录表名，默认 `migration` |

命令：

```bash
go run ./cmd/migrate up [n]
go run ./cmd/migrate down [n]
go run ./cmd/migrate redo [n]
go run ./cmd/migrate history [n]
go run ./cmd/migrate new [n]
go run ./cmd/migrate mark VERSION
go run ./cmd/migrate to VERSION
go run ./cmd/migrate create NAME
go run ./cmd/migrate create NAME --fields="title:string(128):notNull,body:longText"
```

命令说明：

| 命令 | 说明 |
| --- | --- |
| `up [n]` | 执行待执行迁移。`n` 省略或小于等于 0 时执行全部 |
| `down [n]` | 从最新迁移开始回滚。默认回滚 1 个 |
| `redo [n]` | 回滚并重新执行最近 `n` 个迁移。默认 1 个 |
| `history [n]` | 显示已执行迁移。默认显示 10 个 |
| `new [n]` | 显示待执行迁移。默认显示 10 个 |
| `mark VERSION` | 只修改迁移记录，不执行迁移代码 |
| `to VERSION` | 迁移到指定版本；`VERSION` 为 `0` 时回滚全部 |
| `create NAME` | 生成迁移文件 |

## 生成器

基础生成：

```bash
go run ./cmd/migrate create create_user_table
```

会生成类似：

```txt
migrations/m20260613_120000_create_user_table.go
```

带字段生成：

```bash
go run ./cmd/migrate create create_article_table \
  --fields="user_id:unsignedBigInteger:notNull,title:string(128):notNull,content:longText,metadata:json,status:unsignedTinyInteger:notNull:default(10)"
```

`--fields` 格式：

```txt
name:type[:modifier[:modifier...]]
```

常用 modifier：

| modifier | 生成代码 |
| --- | --- |
| `notNull` | `.NotNull()` |
| `null` | `.Null()` |
| `unsigned` | `.Unsigned()` |
| `default(10)` | `.DefaultValue(10)` |
| `default(foo)` | `.DefaultValue("foo")` |
| `defaultExpression(CURRENT_TIMESTAMP)` | `.DefaultExpression("CURRENT_TIMESTAMP")` |

如果字段未指定 `null` 或 `notNull`，生成器默认追加 `.Null()`。

## ColumnBuilder

字段 builder 是不可变链式设计。复用基础 builder 时不会互相污染：

```go
base := m.String(64)
required := base.NotNull()
optional := base.Null()
```

字段类型：

| 方法 | MySQL 类型 |
| --- | --- |
| `m.PrimaryKey()` | `int NOT NULL AUTO_INCREMENT PRIMARY KEY` |
| `m.BigPrimaryKey()` | `bigint NOT NULL AUTO_INCREMENT PRIMARY KEY` |
| `m.UnsignedPrimaryKey()` | `int UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY` |
| `m.UnsignedBigPrimaryKey()` | `bigint UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY` |
| `m.TinyInteger()` | `tinyint` |
| `m.SmallInteger()` | `smallint` |
| `m.Integer()` | `int` |
| `m.BigInteger()` | `bigint` |
| `m.UnsignedTinyInteger()` | `tinyint UNSIGNED` |
| `m.UnsignedSmallInteger()` | `smallint UNSIGNED` |
| `m.UnsignedInteger()` | `int UNSIGNED` |
| `m.UnsignedBigInteger()` | `bigint UNSIGNED` |
| `m.String(128)` | `varchar(128)` |
| `m.Char(36)` | `char(36)` |
| `m.Text()` | `text` |
| `m.TinyText()` | `tinytext` |
| `m.MediumText()` | `mediumtext` |
| `m.LongText()` | `longtext` |
| `m.Binary(16)` | `varbinary(16)` |
| `m.TinyBlob()` | `tinyblob` |
| `m.MediumBlob()` | `mediumblob` |
| `m.LongBlob()` | `longblob` |
| `m.Boolean()` | `tinyint(1)` |
| `m.Float()` | `float` |
| `m.Double()` | `double` |
| `m.Decimal(10, 2)` | `decimal(10,2)` |
| `m.Money(19, 4)` | `decimal(19,4)` |
| `m.Date()` | `date` |
| `m.DateTime(3)` | `datetime(3)` |
| `m.Time(6)` | `time(6)` |
| `m.Timestamp(0)` | `timestamp(0)` |
| `m.Json()` | `json` |
| `m.UUID()` | `char(36)` |
| `m.Enum("draft", "published")` | `enum('draft','published')` |
| `m.Set("read", "write")` | `set('read','write')` |

链式方法：

```go
m.String(64).
	NotNull().
	Unique().
	DefaultValue("admin").
	Comment("用户名").
	Charset("utf8mb4").
	Collate("utf8mb4_bin").
	Check("username <> ''").
	After("id")
```

支持的方法：

```go
NotNull()
Null()
Unsigned()
PrimaryKey()
AutoIncrement()
Unique()
DefaultValue(v any)
DefaultExpression(sql string)
Comment(comment string)
Check(expr string)
After(column string)
First()
Append(sql string)
Charset(charset string)
Collate(collation string)
GeneratedAs(expr string)
Stored()
Virtual()
```

## SchemaPlan

`SchemaPlan` 用于按顺序执行 DDL 和 DML 写操作：

```go
return m.Schema().
	CreateTable("user", migrate.Columns().
		Add("id", m.UnsignedBigPrimaryKey()).
		Add("username", m.String(64).NotNull()),
		"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	).
	CreateIndex("idx-user-username", "user", []string{"username"}, true).
	Insert("user", migrate.Row{
		"username":   "admin",
		"created_at": migrate.Expr("UNIX_TIMESTAMP()"),
	}).
	Exec(ctx)
```

DDL 方法：

```go
Raw(sql string, args ...any)
CreateTable(table string, columns *ColumnList, options ...string)
CreateTableIfNotExists(ctx context.Context, table string, columns *ColumnList, options ...string)
DropTable(table string)
DropTableIfExists(ctx context.Context, table string)
RenameTable(oldName, newName string)
TruncateTable(table string)
AddColumn(table, column string, builder *ColumnBuilder)
AddColumnIfNotExists(ctx context.Context, table, column string, builder *ColumnBuilder)
AlterColumn(table, column string, builder *ColumnBuilder)
DropColumn(table, column string)
DropColumnIfExists(ctx context.Context, table, column string)
RenameColumn(table, oldName, newName string)
CreateIndex(name, table string, columns []string, unique bool)
CreateIndexIfNotExists(ctx context.Context, name, table string, columns []string, unique bool)
DropIndex(name, table string)
DropIndexIfExists(ctx context.Context, name, table string)
AddPrimaryKey(name, table string, columns []string)
AddPrimaryKeyIfNotExists(ctx context.Context, name, table string, columns []string)
DropPrimaryKey(name, table string)
DropPrimaryKeyIfExists(ctx context.Context, name, table string)
AddForeignKey(name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction)
AddForeignKeyIfNotExists(ctx context.Context, name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction)
DropForeignKey(name, table string)
DropForeignKeyIfExists(ctx context.Context, name, table string)
AddCommentOnColumn(table, column, comment string)
DropCommentFromColumn(table, column string)
AddCommentOnTable(table, comment string)
DropCommentFromTable(table string)
```

DML 方法：

```go
Insert(table string, row Row)
BatchInsert(table string, columns []string, rows [][]any)
Update(table string, row Row, condition string, args ...any)
Delete(table string, condition string, args ...any)
```

`Row` 是 `map[string]any`。普通值会被参数绑定；`migrate.Expr(...)` 会作为 SQL 表达式直接写入：

```go
m.Schema().
	Insert("user", migrate.Row{
		"username":   "admin",
		"created_at": migrate.Expr("UNIX_TIMESTAMP()"),
	}).
	Exec(ctx)
```

## 查询 helper

查询 helper 挂在 `MigrationContext` 上，适合在迁移中立即分支判断：

```go
exists, err := m.RowExists(ctx, "user", "username = ?", "admin")
if err != nil {
	return err
}

if !exists {
	return m.Schema().
		Insert("user", migrate.Row{
			"username": "admin",
		}).
		Exec(ctx)
}
```

可用方法：

```go
QueryValue(ctx context.Context, query string, args ...any) (any, error)
QueryOne(ctx context.Context, query string, args ...any) (Row, error)
QueryAll(ctx context.Context, query string, args ...any) ([]Row, error)
RowExists(ctx context.Context, table string, condition string, args ...any) (bool, error)
CountRows(ctx context.Context, table string, condition string, args ...any) (int64, error)
```

行为说明：

- `QueryValue` 查询不到时返回 `nil, nil`。
- `QueryOne` 查询不到时返回 `nil, nil`。
- `QueryAll` 查询不到时返回空切片。
- `NULL` 转换为 `nil`。
- `[]byte` 转换为 `string`。

## 对象存在判断

```go
m.TableExists(ctx, "user")
m.ColumnExists(ctx, "user", "email")
m.IndexExists(ctx, "user", "idx-user-email")
m.ForeignKeyExists(ctx, "profile", "fk-profile-user_id")
m.ConstraintExists(ctx, "profile", "fk-profile-user_id")
```

MySQL 使用 `information_schema` 查询对象是否存在。

## 迁移锁和 dry-run

mutating 命令会默认使用 MySQL advisory lock：

```sql
SELECT GET_LOCK(?, ?)
SELECT RELEASE_LOCK(?)
```

可以通过 `Migrator` 配置：

```go
migrator := migrate.NewMigrator(db, migrate.MySQLDialect{}, migrations.All())
migrator.UseLock = true
migrator.LockName = "yiimigrate"
migrator.LockTimeoutSeconds = 30
migrator.DryRun = true
```

dry-run 模式下：

- 不执行迁移 SQL。
- 不写入 migration 表。
- 不获取迁移锁。
- 会调用迁移代码以收集并打印 SQL。

## 错误

```go
var ErrIrreversibleMigration = errors.New("irreversible migration")
var ErrMigrationNotFound = errors.New("migration not found")
var ErrMigrationLockTimeout = errors.New("migration lock timeout")
```

不可逆迁移：

```go
func (M20260613_120000DangerousMigration) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return migrate.ErrIrreversibleMigration
}
```

## 测试

运行所有单元测试：

```bash
go test ./...
```

运行 MySQL 集成测试：

```bash
set MYSQL_TEST_DSN=root:password@tcp(127.0.0.1:3306)/test?parseTime=true
go test ./...
```

未设置 `MYSQL_TEST_DSN` 时，MySQL 集成测试会跳过。

## MySQL 注意事项

- 当前仅实现 `MySQLDialect`。
- 标识符使用反引号 quote。
- 占位符使用 `?`。
- 索引列支持表达式，例如 `LOWER(email)`、`JSON_EXTRACT(...)`，不会被错误当作普通列名 quote。
- 表、字段、索引、外键、约束存在判断通过 `information_schema` 完成。
- 并发迁移锁通过 `GET_LOCK` / `RELEASE_LOCK` 完成。
- 不包含 ORM 依赖，不使用 GORM。

## Yii2 到 Go API 对照

| Yii2 | yiimigrate |
| --- | --- |
| `$this->createTable(...)` | `m.Schema().CreateTable(...)` |
| `$this->dropTable(...)` | `m.Schema().DropTable(...)` |
| `$this->addColumn(...)` | `m.Schema().AddColumn(...)` |
| `$this->dropColumn(...)` | `m.Schema().DropColumn(...)` |
| `$this->createIndex(...)` | `m.Schema().CreateIndex(...)` |
| `$this->addForeignKey(...)` | `m.Schema().AddForeignKey(...)` |
| `$this->insert(...)` | `m.Schema().Insert(...)` |
| `$this->batchInsert(...)` | `m.Schema().BatchInsert(...)` |
| `$this->update(...)` | `m.Schema().Update(...)` |
| `$this->delete(...)` | `m.Schema().Delete(...)` |
| `$this->execute(...)` | `m.Execute(...)` 或 `m.Schema().Raw(...)` |
| `safeUp()` | `Up(ctx, m)` 默认事务执行 |
| `safeDown()` | `Down(ctx, m)` 默认事务执行 |
