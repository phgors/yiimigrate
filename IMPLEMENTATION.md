# Go 复刻 Yii2 Migration 数据迁移库实施文档

> 目标：使用 Go 实现一个接近 Yii2 Migration 使用体验的数据迁移库。
> 当前方向：以真实、可测试的方言实现逐步适配 Yii2 Migration 的多数据库体验；每个新方言必须明确自身支持能力，并对数据库无法提供的操作返回清晰错误。
> 适用对象：准备使用 Codex 分阶段完成开发的 Go 项目。

---

## 1. 项目目标

本项目目标是使用 Go 实现一个接近 Yii2 Migration 使用体验的数据迁移库，重点复刻以下能力：

1. 基于 `migration` 表记录迁移版本。
2. 支持 `up / down / redo / history / new / mark / to / create` 等迁移命令。
3. 支持事务迁移，类似 Yii2 的 `safeUp()` / `safeDown()`。
4. 支持链式字段声明：

```go
m.String(64).NotNull().Unique().Comment("用户名")
m.UnsignedBigInteger().NotNull()
m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")
m.LongText().Null()
m.Json().Null()
```

5. 支持链式 schema 操作：

```go
return m.Schema().
	CreateTableIfNotExists(ctx, "user", migrate.Columns().
		Add("id", m.UnsignedBigPrimaryKey()).
		Add("username", m.String(64).NotNull().Unique()).
		Add("profile", m.Json().Null()).
		Add("bio", m.LongText().Null()),
		"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	).
	CreateIndexIfNotExists(ctx, "idx-user-username", "user", []string{"username"}, true).
	Exec(ctx)
```

6. 支持对象存在判断：

```go
m.TableExists(ctx, "user")
m.ColumnExists(ctx, "user", "email")
m.IndexExists(ctx, "user", "idx-user-email")
m.ForeignKeyExists(ctx, "profile", "fk-profile-user_id")
m.ConstraintExists(ctx, "profile", "fk-profile-user_id")
```

7. 支持迁移中查询数据是否存在：

```go
m.RowExists(ctx, "user", "username = ?", "admin")
m.CountRows(ctx, "user", "status = ?", 10)
m.QueryOne(ctx, "SELECT * FROM user WHERE id = ?", 1)
m.QueryAll(ctx, "SELECT * FROM user WHERE status = ?", 10)
m.QueryValue(ctx, "SELECT COUNT(*) FROM user")
```

8. 通过 Dialect 抽象逐步支持更多数据库，已实现的方言必须真实可用、可测试。

---

## 2. 技术范围

### 2.1 首期必须支持

- Go module 项目结构
- `database/sql`
- MySQL 方言
- SQLite 方言
- 链式 `ColumnBuilder`
- 链式 `SchemaPlan`
- 迁移记录表
- 事务执行
- 并发迁移锁
- dry-run SQL 预览
- 基础 CLI
- 查询型 DML helper
- 单元测试
- MySQL 集成测试，允许通过环境变量启用
- SQLite SQL 生成单元测试，不要求 SQLite 驱动

### 2.2 首期不强制支持

- ORM
- 自动根据数据库结构反向生成迁移
- 复杂 schema diff
- 多租户连接管理
- GUI
- 未经测试验证的数据库方言

---

## 3. 推荐目录结构

```txt
go-yii-migrate/
  go.mod
  README.md
  AGENTS.md

  migrate/
    migration.go
    migrator.go
    context.go
    column_builder.go
    columns.go
    schema_plan.go
    dialect.go
    mysql.go
    dml.go
    query.go
    exists.go
    lock.go
    logger.go
    errors.go

  cmd/
    migrate/
      main.go
      command_up.go
      command_down.go
      command_history.go
      command_create.go
      command_redo.go
      command_mark.go
      command_to.go
      command_new.go

  migrations/
    register.go
    m20260612_120000_create_user_table.go

  internal/
    generator/
      generator.go
      parser.go
      template.go

  tests/
    migrator_test.go
    column_builder_test.go
    mysql_dialect_test.go
    schema_plan_test.go
    dml_test.go
    query_test.go
```

---

## 4. 核心 API 设计

### 4.1 Migration 接口

```go
type Migration interface {
	Name() string
	Up(ctx context.Context, m *MigrationContext) error
	Down(ctx context.Context, m *MigrationContext) error
}
```

### 4.2 DBTX 接口

因为迁移中既要执行 SQL，也要查询数据，所以 `DBTX` 必须支持：

```go
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
```

`*sql.DB` 和 `*sql.Tx` 都满足这个接口。

### 4.3 MigrationContext

`MigrationContext` 负责暴露 Yii2 风格的方法。

必须提供：

```go
func (m *MigrationContext) Schema() *SchemaPlan
func (m *MigrationContext) Execute(ctx context.Context, sql string, args ...any) error

func (m *MigrationContext) TableExists(ctx context.Context, table string) (bool, error)
func (m *MigrationContext) ColumnExists(ctx context.Context, table, column string) (bool, error)
func (m *MigrationContext) IndexExists(ctx context.Context, table, index string) (bool, error)
func (m *MigrationContext) ForeignKeyExists(ctx context.Context, table, name string) (bool, error)
func (m *MigrationContext) ConstraintExists(ctx context.Context, table, name string) (bool, error)

func (m *MigrationContext) QueryValue(ctx context.Context, query string, args ...any) (any, error)
func (m *MigrationContext) QueryOne(ctx context.Context, query string, args ...any) (Row, error)
func (m *MigrationContext) QueryAll(ctx context.Context, query string, args ...any) ([]Row, error)
func (m *MigrationContext) RowExists(ctx context.Context, table string, condition string, args ...any) (bool, error)
func (m *MigrationContext) CountRows(ctx context.Context, table string, condition string, args ...any) (int64, error)
```

---

## 5. 字段 Builder 方法

`MigrationContext` 必须提供以下字段 builder 方法：

```go
func (m *MigrationContext) PrimaryKey() *ColumnBuilder
func (m *MigrationContext) BigPrimaryKey() *ColumnBuilder
func (m *MigrationContext) UnsignedPrimaryKey() *ColumnBuilder
func (m *MigrationContext) UnsignedBigPrimaryKey() *ColumnBuilder

func (m *MigrationContext) TinyInteger(length ...int) *ColumnBuilder
func (m *MigrationContext) SmallInteger(length ...int) *ColumnBuilder
func (m *MigrationContext) Integer(length ...int) *ColumnBuilder
func (m *MigrationContext) BigInteger(length ...int) *ColumnBuilder

func (m *MigrationContext) UnsignedTinyInteger(length ...int) *ColumnBuilder
func (m *MigrationContext) UnsignedSmallInteger(length ...int) *ColumnBuilder
func (m *MigrationContext) UnsignedInteger(length ...int) *ColumnBuilder
func (m *MigrationContext) UnsignedBigInteger(length ...int) *ColumnBuilder

func (m *MigrationContext) String(size ...int) *ColumnBuilder
func (m *MigrationContext) Char(size int) *ColumnBuilder
func (m *MigrationContext) Text() *ColumnBuilder
func (m *MigrationContext) TinyText() *ColumnBuilder
func (m *MigrationContext) MediumText() *ColumnBuilder
func (m *MigrationContext) LongText() *ColumnBuilder

func (m *MigrationContext) Binary(length ...int) *ColumnBuilder
func (m *MigrationContext) TinyBlob() *ColumnBuilder
func (m *MigrationContext) MediumBlob() *ColumnBuilder
func (m *MigrationContext) LongBlob() *ColumnBuilder

func (m *MigrationContext) Boolean() *ColumnBuilder
func (m *MigrationContext) Float(precision ...int) *ColumnBuilder
func (m *MigrationContext) Double(precision ...int) *ColumnBuilder
func (m *MigrationContext) Decimal(precision, scale int) *ColumnBuilder
func (m *MigrationContext) Money(precision, scale int) *ColumnBuilder

func (m *MigrationContext) Date() *ColumnBuilder
func (m *MigrationContext) DateTime(precision ...int) *ColumnBuilder
func (m *MigrationContext) Time(precision ...int) *ColumnBuilder
func (m *MigrationContext) Timestamp(precision ...int) *ColumnBuilder

func (m *MigrationContext) Json() *ColumnBuilder
func (m *MigrationContext) UUID() *ColumnBuilder
func (m *MigrationContext) Enum(values ...string) *ColumnBuilder
func (m *MigrationContext) Set(values ...string) *ColumnBuilder
```

---

## 6. ColumnBuilder 设计

### 6.1 目标

支持类似 Yii2 的链式字段声明。

示例：

```go
m.String(128).NotNull().Unique().Comment("标题")
m.UnsignedBigInteger().NotNull().After("id")
m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")
m.Decimal(10, 2).Unsigned().NotNull().DefaultValue(0)
m.LongText().Null()
m.Json().Null()
```

### 6.2 必须支持的链式方法

```go
func (c *ColumnBuilder) NotNull() *ColumnBuilder
func (c *ColumnBuilder) Null() *ColumnBuilder
func (c *ColumnBuilder) Unsigned() *ColumnBuilder
func (c *ColumnBuilder) PrimaryKey() *ColumnBuilder
func (c *ColumnBuilder) AutoIncrement() *ColumnBuilder
func (c *ColumnBuilder) Unique() *ColumnBuilder
func (c *ColumnBuilder) DefaultValue(v any) *ColumnBuilder
func (c *ColumnBuilder) DefaultExpression(sql string) *ColumnBuilder
func (c *ColumnBuilder) Comment(comment string) *ColumnBuilder
func (c *ColumnBuilder) Check(expr string) *ColumnBuilder
func (c *ColumnBuilder) After(column string) *ColumnBuilder
func (c *ColumnBuilder) First() *ColumnBuilder
func (c *ColumnBuilder) Append(sql string) *ColumnBuilder
func (c *ColumnBuilder) Charset(charset string) *ColumnBuilder
func (c *ColumnBuilder) Collate(collation string) *ColumnBuilder
func (c *ColumnBuilder) GeneratedAs(expr string) *ColumnBuilder
func (c *ColumnBuilder) Stored() *ColumnBuilder
func (c *ColumnBuilder) Virtual() *ColumnBuilder
```

### 6.3 不可变链式设计

每个链式方法应该返回 clone 后的新对象，避免复用 builder 时发生状态污染。

正确行为：

```go
base := m.String(64)
a := base.NotNull()
b := base.Null()
```

`a` 和 `b` 不应该互相影响。

### 6.4 Expression 类型

新增：

```go
type Expression string

func Expr(sql string) Expression {
	return Expression(sql)
}
```

用途：

```go
m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")

m.Schema().
	Update("user", migrate.Row{
		"updated_at": migrate.Expr("UNIX_TIMESTAMP()"),
	}, "id = ?", 1).
	Exec(ctx)
```

`Expression` 不应该被当成普通字符串加引号，也不应该作为参数绑定。

---

## 7. Columns 有序结构

Go map 无序，建表字段必须保持声明顺序，所以使用有序结构：

```go
type ColumnDef struct {
	Name   string
	Column *ColumnBuilder
}

type ColumnList struct {
	items []ColumnDef
}

func Columns() *ColumnList
func (c *ColumnList) Add(name string, column *ColumnBuilder) *ColumnList
func (c *ColumnList) Items() []ColumnDef
```

使用示例：

```go
migrate.Columns().
	Add("id", m.UnsignedBigPrimaryKey()).
	Add("username", m.String(64).NotNull()).
	Add("created_at", m.Integer().Unsigned().NotNull())
```

---

## 8. Dialect 设计

### 8.1 数据库支持范围

项目通过方言类型声明数据库支持。已实现方言示例：

```go
type MySQLDialect struct{}
type SQLiteDialect struct{}
```

后续可以继续增加真实方言，例如：

```go
type PostgresDialect struct{}
type SQLServerDialect struct{}
```

方言实现原则：

1. 公开的方言必须有 SQL 生成单元测试。
2. 不同数据库的 DDL 差异必须体现在方言代码和能力错误中。
3. 数据库无法提供的操作必须返回明确错误，不能生成误导性 SQL。
4. README 只能宣称已经真实实现并通过测试的数据库。

### 8.2 Dialect 接口

```go
type Dialect interface {
	QuoteTable(name string) string
	QuoteColumn(name string) string
	QuoteIndexColumn(name string) string
	Placeholder(index int) string

	TableExistsSQL(table string) (string, []any)
	ColumnExistsSQL(table, column string) (string, []any)
	IndexExistsSQL(table, index string) (string, []any)
	ForeignKeyExistsSQL(table, name string) (string, []any)
	ConstraintExistsSQL(table, name string) (string, []any)

	BuildRowExistsSQL(table string, condition string) string
	BuildCountRowsSQL(table string, condition string) string

	CreateTable(table string, columns *ColumnList, options string) string
	DropTable(table string) string
	RenameTable(oldName, newName string) string
	TruncateTable(table string) string

	AddColumn(table, column string, builder *ColumnBuilder) string
	AlterColumn(table, column string, builder *ColumnBuilder) string
	DropColumn(table, column string) string
	RenameColumn(table, oldName, newName string) string

	CreateIndex(name, table string, columns []string, unique bool) string
	DropIndex(name, table string) string

	AddPrimaryKey(name, table string, columns []string) string
	DropPrimaryKey(name, table string) string

	AddForeignKey(
		name string,
		table string,
		columns []string,
		refTable string,
		refColumns []string,
		onDelete ForeignKeyAction,
		onUpdate ForeignKeyAction,
	) string

	DropForeignKey(name, table string) string

	AddCommentOnColumn(table, column, comment string) string
	DropCommentFromColumn(table, column string) string
	AddCommentOnTable(table, comment string) string
	DropCommentFromTable(table string) string

	Insert(table string, row Row) (string, []any)
	BatchInsert(table string, columns []string, rows [][]any) (string, []any)
	Update(table string, row Row, condition string, args ...any) (string, []any)
	Delete(table string, condition string, args ...any) (string, []any)

	AcquireLockSQL(lockName string, timeoutSeconds int) (string, []any)
	ReleaseLockSQL(lockName string) (string, []any)
}
```

### 8.3 MySQL 实现要求

MySQL 方言必须处理：

- 反引号 quote
- `?` 占位符
- `AUTO_INCREMENT`
- `UNSIGNED`
- `COMMENT`
- `DEFAULT CURRENT_TIMESTAMP`
- `ON UPDATE CURRENT_TIMESTAMP`
- `ENUM`
- `SET`
- `JSON`
- `LONGTEXT`
- `LONGBLOB`
- `DROP INDEX name ON table`
- `ALTER TABLE table DROP FOREIGN KEY name`
- `information_schema` 对象存在查询
- `GET_LOCK` / `RELEASE_LOCK`

### 8.4 表、字段、索引、外键、约束存在判断

MySQL 使用 `information_schema`：

```sql
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name = ?
```

```sql
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND column_name = ?
```

```sql
SELECT COUNT(*)
FROM information_schema.statistics
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND index_name = ?
```

```sql
SELECT COUNT(*)
FROM information_schema.table_constraints
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND constraint_name = ?
  AND constraint_type = 'FOREIGN KEY'
```

```sql
SELECT COUNT(*)
FROM information_schema.table_constraints
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND constraint_name = ?
```

---

## 9. SchemaPlan 链式操作

### 9.1 基础结构

```go
type SQLStatement struct {
	Query string
	Args  []any
}

type SchemaPlan struct {
	ctx    *MigrationContext
	sqls   []SQLStatement
	err    error
	dryRun bool
}
```

注意：不要只保存字符串，DML 需要参数绑定。

### 9.2 必须支持的 DDL 方法

```go
func (p *SchemaPlan) Raw(sql string, args ...any) *SchemaPlan

func (p *SchemaPlan) CreateTable(table string, columns *ColumnList, options ...string) *SchemaPlan
func (p *SchemaPlan) CreateTableIfNotExists(ctx context.Context, table string, columns *ColumnList, options ...string) *SchemaPlan
func (p *SchemaPlan) DropTable(table string) *SchemaPlan
func (p *SchemaPlan) DropTableIfExists(ctx context.Context, table string) *SchemaPlan
func (p *SchemaPlan) RenameTable(oldName, newName string) *SchemaPlan
func (p *SchemaPlan) TruncateTable(table string) *SchemaPlan

func (p *SchemaPlan) AddColumn(table, column string, builder *ColumnBuilder) *SchemaPlan
func (p *SchemaPlan) AddColumnIfNotExists(ctx context.Context, table, column string, builder *ColumnBuilder) *SchemaPlan
func (p *SchemaPlan) AlterColumn(table, column string, builder *ColumnBuilder) *SchemaPlan
func (p *SchemaPlan) DropColumn(table, column string) *SchemaPlan
func (p *SchemaPlan) DropColumnIfExists(ctx context.Context, table, column string) *SchemaPlan
func (p *SchemaPlan) RenameColumn(table, oldName, newName string) *SchemaPlan

func (p *SchemaPlan) CreateIndex(name, table string, columns []string, unique bool) *SchemaPlan
func (p *SchemaPlan) CreateIndexIfNotExists(ctx context.Context, name, table string, columns []string, unique bool) *SchemaPlan
func (p *SchemaPlan) DropIndex(name, table string) *SchemaPlan
func (p *SchemaPlan) DropIndexIfExists(ctx context.Context, name, table string) *SchemaPlan

func (p *SchemaPlan) AddPrimaryKey(name, table string, columns []string) *SchemaPlan
func (p *SchemaPlan) AddPrimaryKeyIfNotExists(ctx context.Context, name, table string, columns []string) *SchemaPlan
func (p *SchemaPlan) DropPrimaryKey(name, table string) *SchemaPlan
func (p *SchemaPlan) DropPrimaryKeyIfExists(ctx context.Context, name, table string) *SchemaPlan

func (p *SchemaPlan) AddForeignKey(name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) *SchemaPlan
func (p *SchemaPlan) AddForeignKeyIfNotExists(ctx context.Context, name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) *SchemaPlan
func (p *SchemaPlan) DropForeignKey(name, table string) *SchemaPlan
func (p *SchemaPlan) DropForeignKeyIfExists(ctx context.Context, name, table string) *SchemaPlan

func (p *SchemaPlan) AddCommentOnColumn(table, column, comment string) *SchemaPlan
func (p *SchemaPlan) DropCommentFromColumn(table, column string) *SchemaPlan
func (p *SchemaPlan) AddCommentOnTable(table, comment string) *SchemaPlan
func (p *SchemaPlan) DropCommentFromTable(table string) *SchemaPlan
```

### 9.3 必须支持的 DML 写方法

```go
type Row map[string]any
type Expression string

func Expr(sql string) Expression
```

```go
func (p *SchemaPlan) Insert(table string, row Row) *SchemaPlan
func (p *SchemaPlan) BatchInsert(table string, columns []string, rows [][]any) *SchemaPlan
func (p *SchemaPlan) Update(table string, row Row, condition string, args ...any) *SchemaPlan
func (p *SchemaPlan) Delete(table string, condition string, args ...any) *SchemaPlan
```

使用示例：

```go
return m.Schema().
	Insert("user", migrate.Row{
		"username": "admin",
		"email": "admin@example.com",
		"created_at": migrate.Expr("UNIX_TIMESTAMP()"),
	}).
	BatchInsert("role", []string{"name", "created_at"}, [][]any{
		{"admin", migrate.Expr("UNIX_TIMESTAMP()")},
		{"member", migrate.Expr("UNIX_TIMESTAMP()")},
	}).
	Exec(ctx)
```

### 9.4 Exec 行为

`Exec(ctx)` 必须：

1. 如果 `SchemaPlan.err != nil`，立即返回错误。
2. 按添加顺序执行 SQL。
3. 打印 SQL 日志。
4. 支持 dry-run，只打印不执行。
5. DML 使用参数绑定。
6. 返回第一条失败 SQL 的错误。

---

## 10. 查询型 DML Helper

### 10.1 为什么需要查询型 helper

迁移过程中经常需要根据已有数据决定是否执行写操作，例如：

- 判断管理员账号是否已存在。
- 判断某个配置项是否已初始化。
- 判断旧数据是否需要修复。
- 判断某条关联数据是否存在后再插入。
- 查询旧字段值后迁移到新字段。

因此除了 `Insert / BatchInsert / Update / Delete`，还必须提供查询型 helper。

### 10.2 设计原则

查询型 DML 不建议放入 `SchemaPlan` 链式队列中，因为迁移通常需要立即根据查询结果做分支判断。

正确设计：

```go
exists, err := m.RowExists(ctx, "user", "username = ?", "admin")
if err != nil {
	return err
}

if !exists {
	return m.Schema().
		Insert("user", migrate.Row{
			"username": "admin",
			"email": "admin@example.com",
			"created_at": migrate.Expr("UNIX_TIMESTAMP()"),
		}).
		Exec(ctx)
}

return nil
```

不建议设计成：

```go
m.Schema().
	RowExists(...).
	Insert(...).
	Exec(ctx)
```

查询需要立即执行，链式 plan 只负责顺序执行 DDL / DML 写操作。

### 10.3 MigrationContext 查询方法

```go
func (m *MigrationContext) QueryValue(ctx context.Context, query string, args ...any) (any, error)

func (m *MigrationContext) QueryOne(ctx context.Context, query string, args ...any) (Row, error)

func (m *MigrationContext) QueryAll(ctx context.Context, query string, args ...any) ([]Row, error)

func (m *MigrationContext) RowExists(ctx context.Context, table string, condition string, args ...any) (bool, error)

func (m *MigrationContext) CountRows(ctx context.Context, table string, condition string, args ...any) (int64, error)
```

### 10.4 使用示例：判断数据是否存在

```go
func (M20260612_180000SeedAdminUser) Up(ctx context.Context, m *migrate.MigrationContext) error {
	exists, err := m.RowExists(ctx, "user", "username = ?", "admin")
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return m.Schema().
		Insert("user", migrate.Row{
			"username": "admin",
			"email": "admin@example.com",
			"status": 10,
			"created_at": migrate.Expr("UNIX_TIMESTAMP()"),
			"updated_at": migrate.Expr("UNIX_TIMESTAMP()"),
		}).
		Exec(ctx)
}
```

### 10.5 使用示例：统计旧数据

```go
func (M20260612_181000FixEmptyEmail) Up(ctx context.Context, m *migrate.MigrationContext) error {
	count, err := m.CountRows(ctx, "user", "email IS NULL OR email = ''")
	if err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	return m.Schema().
		Update(
			"user",
			migrate.Row{
				"email": migrate.Expr("CONCAT('unknown_', id, '@example.com')"),
				"updated_at": migrate.Expr("UNIX_TIMESTAMP()"),
			},
			"email IS NULL OR email = ''",
		).
		Exec(ctx)
}
```

### 10.6 使用示例：查询一行数据

```go
func (M20260612_182000MigrateSetting) Up(ctx context.Context, m *migrate.MigrationContext) error {
	row, err := m.QueryOne(ctx, "SELECT value FROM setting WHERE name = ?", "site_name")
	if err != nil {
		return err
	}

	if row == nil {
		return m.Schema().
			Insert("setting", migrate.Row{
				"name": "site_name",
				"value": "Default Site",
			}).
			Exec(ctx)
	}

	return nil
}
```

### 10.7 QueryOne 行为

`QueryOne` 的行为要求：

1. 查询不到数据时返回 `nil, nil`。
2. 查询到数据时返回 `Row`。
3. SQL 错误时返回错误。
4. 字段名使用数据库返回的 column name。
5. `NULL` 值应保存为 `nil`。

示例：

```go
row, err := m.QueryOne(ctx, "SELECT id, username FROM user WHERE id = ?", 1)
if err != nil {
	return err
}

if row == nil {
	return nil
}

id := row["id"]
username := row["username"]
```

### 10.8 QueryAll 行为

`QueryAll` 的行为要求：

1. 查询不到数据时返回空切片，不返回 nil。
2. SQL 错误时返回错误。
3. 每行是一个 `Row`。
4. `NULL` 值应保存为 `nil`。

### 10.9 Dialect 查询 SQL 构造

为了支持多数据库，`Dialect` 接口需要包含：

```go
BuildRowExistsSQL(table string, condition string) string
BuildCountRowsSQL(table string, condition string) string
Placeholder(index int) string
```

MySQL 实现：

```go
func (d MySQLDialect) BuildRowExistsSQL(table string, condition string) string {
	sql := "SELECT EXISTS(SELECT 1 FROM " + d.QuoteTable(table)

	if condition != "" {
		sql += " WHERE " + condition
	}

	sql += " LIMIT 1)"

	return sql
}

func (d MySQLDialect) BuildCountRowsSQL(table string, condition string) string {
	sql := "SELECT COUNT(*) FROM " + d.QuoteTable(table)

	if condition != "" {
		sql += " WHERE " + condition
	}

	return sql
}

func (d MySQLDialect) Placeholder(index int) string {
	return "?"
}
```

注意：

- 已实现方言必须真实可用并覆盖 SQL 生成测试。
- 接口设计必须允许后续扩展更多数据库。
- 所有自动生成 SQL 的地方应通过 `Dialect.Placeholder(index)` 生成占位符。
- MySQL 返回 `?`。
- PostgreSQL 后续可以返回 `$1`、`$2`、`$3`。

### 10.10 Query 方法实现要求

新增文件：

```txt
migrate/query.go
```

建议实现：

```go
func (m *MigrationContext) QueryValue(ctx context.Context, query string, args ...any) (any, error) {
	row := m.DB.QueryRowContext(ctx, query, args...)

	var value any
	if err := row.Scan(&value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return normalizeDBValue(value), nil
}
```

`QueryOne` 和 `QueryAll` 需要把 `*sql.Rows` 转成 `[]Row`。

扫描要求：

```go
func scanRows(rows *sql.Rows) ([]Row, error)
```

行为：

1. 使用 `rows.Columns()` 获取列名。
2. 每一列扫描到 `[]any`。
3. `[]byte` 类型应转换为 `string`，避免调用方拿到 MySQL driver 的 raw bytes。
4. `NULL` 转为 `nil`。
5. 关闭 rows。
6. 检查 `rows.Err()`。

示例转换函数：

```go
func normalizeDBValue(v any) any {
	switch x := v.(type) {
	case []byte:
		return string(x)
	default:
		return x
	}
}
```

---

## 11. Migrator 设计

### 11.1 Migrator 结构

```go
type Migrator struct {
	DB             *sql.DB
	Dialect        Dialect
	MigrationTable string
	Migrations      []Migration

	UseTransaction bool
	DryRun          bool

	UseLock            bool
	LockName           string
	LockTimeoutSeconds int

	Logger Logger
}
```

默认值：

```go
MigrationTable = "migration"
UseTransaction = true
DryRun = false
UseLock = true
LockName = "go_yii_migrate"
LockTimeoutSeconds = 10
```

### 11.2 必须支持的方法

```go
func NewMigrator(db *sql.DB, dialect Dialect, migrations []Migration) *Migrator

func (m *Migrator) EnsureMigrationTable(ctx context.Context) error
func (m *Migrator) Applied(ctx context.Context) (map[string]int64, error)
func (m *Migrator) Pending(ctx context.Context) ([]Migration, error)

func (m *Migrator) Up(ctx context.Context, limit int) error
func (m *Migrator) Down(ctx context.Context, limit int) error
func (m *Migrator) Redo(ctx context.Context, limit int) error
func (m *Migrator) History(ctx context.Context, limit int) error
func (m *Migrator) New(ctx context.Context, limit int) error
func (m *Migrator) Mark(ctx context.Context, version string) error
func (m *Migrator) To(ctx context.Context, version string) error
```

### 11.3 迁移顺序

- `Up` 按 `Name()` 升序执行。
- `Down` 按 `Name()` 降序回滚。
- 执行成功后写入 migration 表。
- 回滚成功后删除 migration 表中的版本记录。
- 如果执行失败，当前迁移应整体失败。
- 事务开启时，失败必须 rollback。

### 11.4 migration 表结构

MySQL：

```sql
CREATE TABLE IF NOT EXISTS `migration` (
  `version` varchar(180) NOT NULL PRIMARY KEY,
  `apply_time` int NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
```

---

## 12. 并发迁移锁

### 12.1 目标

防止多个进程同时执行迁移。

### 12.2 MySQL 实现

加锁：

```sql
SELECT GET_LOCK(?, ?)
```

释放锁：

```sql
SELECT RELEASE_LOCK(?)
```

要求：

- `Up / Down / Redo / To / Mark` 执行前加锁。
- 函数退出时必须释放锁。
- 获取锁失败时返回明确错误。
- dry-run 模式可以不加锁。

---

## 13. 错误设计

新增：

```go
var ErrIrreversibleMigration = errors.New("irreversible migration")
var ErrMigrationNotFound = errors.New("migration not found")
var ErrMigrationLockTimeout = errors.New("migration lock timeout")
```

不可回滚迁移示例：

```go
func (M20260612_180000DangerousMigration) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return migrate.ErrIrreversibleMigration
}
```

CLI 遇到 `ErrIrreversibleMigration` 时输出：

```txt
Migration cannot be reverted.
```

---

## 14. CLI 设计

### 14.1 命令

```bash
migrate up [n]
migrate down [n]
migrate redo [n]
migrate history [n]
migrate new [n]
migrate mark VERSION
migrate to VERSION
migrate create NAME
migrate create NAME --fields="title:string(128):notNull,body:longText,status:tinyInteger:unsigned:notNull:default(10)"
```

### 14.2 环境变量

```bash
DB_DSN='root:password@tcp(127.0.0.1:3306)/test?parseTime=true'
MIGRATE_DRY_RUN=1
MIGRATE_TABLE=migration
```

### 14.3 CLI 验收标准

执行：

```bash
go run ./cmd/migrate up
```

应该输出：

```txt
>>> applying m20260612_120000_create_user_table
    > CREATE TABLE `user` ...
<<< applied m20260612_120000_create_user_table
```

执行：

```bash
go run ./cmd/migrate down 1
```

应该输出：

```txt
>>> reverting m20260612_120000_create_user_table
    > DROP TABLE `user`
<<< reverted m20260612_120000_create_user_table
```

---

## 15. 代码生成器

### 15.1 create 命令

```bash
go run ./cmd/migrate create create_user_table
```

生成文件：

```txt
migrations/m20260612_120000_create_user_table.go
```

内容：

```go
package migrations

import (
	"context"

	"your_project/migrate"
)

type M20260612_120000CreateUserTable struct{}

func (M20260612_120000CreateUserTable) Name() string {
	return "m20260612_120000_create_user_table"
}

func (M20260612_120000CreateUserTable) Up(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		Raw("-- TODO: add migration SQL").
		Exec(ctx)
}

func (M20260612_120000CreateUserTable) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return migrate.ErrIrreversibleMigration
}
```

### 15.2 fields 参数

输入：

```bash
go run ./cmd/migrate create create_article_table \
  --fields="user_id:unsignedBigInteger:notNull,title:string(128):notNull,content:longText,metadata:json,status:unsignedTinyInteger:notNull:default(10)"
```

生成 Up：

```go
return m.Schema().
	CreateTable("article", migrate.Columns().
		Add("id", m.UnsignedBigPrimaryKey()).
		Add("user_id", m.UnsignedBigInteger().NotNull()).
		Add("title", m.String(128).NotNull()).
		Add("content", m.LongText().Null()).
		Add("metadata", m.Json().Null()).
		Add("status", m.UnsignedTinyInteger().NotNull().DefaultValue(10)),
		"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	).
	Exec(ctx)
```

生成 Down：

```go
return m.Schema().
	DropTable("article").
	Exec(ctx)
```

---

## 16. 示例迁移文件

```go
package migrations

import (
	"context"

	"your_project/migrate"
)

type M20260612_120000CreateUserAndProfile struct{}

func (M20260612_120000CreateUserAndProfile) Name() string {
	return "m20260612_120000_create_user_and_profile"
}

func (M20260612_120000CreateUserAndProfile) Up(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		CreateTableIfNotExists(ctx, "user", migrate.Columns().
			Add("id", m.UnsignedBigPrimaryKey()).
			Add("username", m.String(64).NotNull().Unique().Comment("用户名")).
			Add("email", m.String(128).NotNull().Unique().Comment("邮箱")).
			Add("password_hash", m.String(255).NotNull()).
			Add("status", m.UnsignedTinyInteger().NotNull().DefaultValue(10).Comment("状态")).
			Add("profile", m.Json().Null()).
			Add("bio", m.LongText().Null()).
			Add("created_at", m.UnsignedInteger().NotNull()).
			Add("updated_at", m.UnsignedInteger().Null()),
			"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		).
		CreateTableIfNotExists(ctx, "profile", migrate.Columns().
			Add("id", m.UnsignedBigPrimaryKey()).
			Add("user_id", m.UnsignedBigInteger().NotNull()).
			Add("nickname", m.String(64).Null()).
			Add("avatar", m.String(255).Null()).
			Add("extra", m.Json().Null()),
			"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		).
		CreateIndexIfNotExists(ctx, "idx-profile-user_id", "profile", []string{"user_id"}, true).
		AddForeignKeyIfNotExists(
			ctx,
			"fk-profile-user_id",
			"profile",
			[]string{"user_id"},
			"user",
			[]string{"id"},
			migrate.Cascade,
			migrate.Cascade,
		).
		Exec(ctx)
}

func (M20260612_120000CreateUserAndProfile) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		DropForeignKeyIfExists(ctx, "fk-profile-user_id", "profile").
		DropIndexIfExists(ctx, "idx-profile-user_id", "profile").
		DropTableIfExists(ctx, "profile").
		DropTableIfExists(ctx, "user").
		Exec(ctx)
}
```

---

## 17. 测试要求

### 17.1 单元测试

必须覆盖：

1. ColumnBuilder 生成 SQL。
2. `Unsigned()` 是否正确输出。
3. `LongText()` 输出 `LONGTEXT`。
4. `Json()` 输出 `JSON`。
5. `DefaultValue()` 是否正确使用参数或 literal。
6. `DefaultExpression()` 是否不会被加引号。
7. `CreateTable()` 字段顺序是否稳定。
8. `CreateIndex()` 是否支持表达式索引。
9. `Insert / BatchInsert / Update / Delete` 参数绑定是否正确。
10. `IfExists / IfNotExists` 在对象存在或不存在时行为是否正确。
11. 不可变链式 builder 是否不会互相污染。
12. `QueryValue` 查询单个值。
13. `QueryValue` 查询不到时返回 `nil, nil`。
14. `QueryOne` 查询一行。
15. `QueryOne` 查询不到时返回 `nil, nil`。
16. `QueryAll` 查询多行。
17. `QueryAll` 查询不到时返回空切片。
18. `RowExists` 数据存在时返回 true。
19. `RowExists` 数据不存在时返回 false。
20. `CountRows` 返回正确数量。
21. `NULL` 字段转换为 `nil`。
22. `[]byte` 字段转换为 `string`。
23. SQL 错误必须返回 error。

### 17.2 集成测试

通过环境变量启用：

```bash
MYSQL_TEST_DSN='root:password@tcp(127.0.0.1:3306)/test?parseTime=true' go test ./...
```

必须覆盖：

1. 自动创建 migration 表。
2. `Up` 后 migration 表有记录。
3. `Down` 后 migration 表记录删除。
4. 重复执行 `Up` 不会重复执行已应用迁移。
5. `CreateTableIfNotExists` 可重复执行。
6. `AddColumnIfNotExists` 可重复执行。
7. `CreateIndexIfNotExists` 可重复执行。
8. `AddForeignKeyIfNotExists` 可重复执行。
9. 事务失败会 rollback。
10. 并发执行时锁生效。
11. 查询型 DML helper 在 MySQL 下行为正确。

---

## 18. Codex 执行顺序

建议分多个 Codex task 完成，不要一次性让 Codex 写完整库。

### Task 1：初始化项目和核心接口

Prompt：

```txt
请在当前 Go 仓库中实现一个 Yii2 风格 migration 库的基础结构。

要求：
1. 创建 migrate 包。
2. 实现 Migration、MigrationContext、DBTX、Dialect、Migrator 基础结构。
3. 实现 migration 表创建、Applied、Pending、Up、Down、History。
4. 默认使用事务执行每个 migration。
5. 添加 ErrIrreversibleMigration。
6. 实现首批真实可用的 Dialect，并保持接口可继续扩展更多数据库。
7. 添加基础单元测试。
8. 代码必须 gofmt，go test ./... 通过。
```

### Task 2：实现 ColumnBuilder 和字段类型

Prompt：

```txt
请实现 Yii2 风格的 ColumnBuilder。

要求：
1. 支持 PrimaryKey、BigPrimaryKey、UnsignedPrimaryKey、UnsignedBigPrimaryKey。
2. 支持 TinyInteger、SmallInteger、Integer、BigInteger 和对应 Unsigned 版本。
3. 支持 String、Char、Text、TinyText、MediumText、LongText。
4. 支持 Binary、TinyBlob、MediumBlob、LongBlob。
5. 支持 Boolean、Float、Double、Decimal、Money。
6. 支持 Date、DateTime、Time、Timestamp。
7. 支持 Json、UUID、Enum、Set。
8. 支持链式方法 NotNull、Null、Unsigned、PrimaryKey、AutoIncrement、Unique、DefaultValue、DefaultExpression、Comment、Check、After、First、Append、Charset、Collate、GeneratedAs、Stored、Virtual。
9. ColumnBuilder 必须是不可变链式设计。
10. 添加完整单元测试。
11. go test ./... 必须通过。
```

### Task 3：实现 MySQLDialect

Prompt：

```txt
请实现 MySQLDialect。

要求：
1. 支持 QuoteTable、QuoteColumn、QuoteIndexColumn、Placeholder。
2. 支持 CreateTable、DropTable、RenameTable、TruncateTable。
3. 支持 AddColumn、AlterColumn、DropColumn、RenameColumn。
4. 支持 CreateIndex、DropIndex。
5. 支持 AddPrimaryKey、DropPrimaryKey。
6. 支持 AddForeignKey、DropForeignKey。
7. 支持表注释和字段注释。
8. 支持 ColumnBuilder 的所有 MySQL SQL 输出。
9. QuoteIndexColumn 遇到 LOWER(email)、JSON_EXTRACT(...) 等表达式时不能错误加反引号。
10. 新增数据库方言时必须提供真实 SQL 生成实现和测试覆盖。
11. 添加 mysql_dialect_test.go。
12. go test ./... 必须通过。
```

### Task 4：实现对象存在判断

Prompt：

```txt
请为 migration 库实现对象存在判断。

要求：
1. MigrationContext 支持 TableExists、ColumnExists、IndexExists、ForeignKeyExists、ConstraintExists。
2. MySQLDialect 使用 information_schema 实现对应 SQL。
3. SchemaPlan 增加 CreateTableIfNotExists、DropTableIfExists、AddColumnIfNotExists、DropColumnIfExists、CreateIndexIfNotExists、DropIndexIfExists、AddForeignKeyIfNotExists、DropForeignKeyIfExists、AddPrimaryKeyIfNotExists、DropPrimaryKeyIfExists。
4. IfExists / IfNotExists 方法如果查询失败，必须把错误保存到 SchemaPlan.err，并在 Exec 时返回。
5. 添加单元测试和集成测试。
6. go test ./... 必须通过。
```

### Task 5：实现 SchemaPlan 和 DML 写操作

Prompt：

```txt
请完善 SchemaPlan 并支持 DML 写操作。

要求：
1. SchemaPlan 内部保存 SQLStatement，包括 Query 和 Args。
2. Exec 按顺序执行所有 SQLStatement。
3. 支持 dry-run，只打印 SQL 不执行。
4. 支持 Raw。
5. 支持 Insert、BatchInsert、Update、Delete。
6. 新增 Row map[string]any。
7. 新增 Expression 类型和 Expr(sql string) 函数。
8. DML 中 Expression 不应被作为参数绑定，而是直接写入 SQL。
9. 普通值必须使用参数绑定。
10. 所有占位符通过 Dialect.Placeholder(index) 生成。
11. MySQL 和 SQLite 返回 ?，PostgreSQL 等后续方言可按数据库规则返回不同占位符。
12. 添加 dml_test.go。
13. go test ./... 必须通过。
```

### Task 6：实现查询型 DML Helper

Prompt：

```txt
请为当前 migration 库实现查询型 DML helper。

要求：
1. 新增 migrate/query.go。
2. MigrationContext 增加 QueryValue、QueryOne、QueryAll、RowExists、CountRows。
3. QueryOne 查询不到时返回 nil, nil。
4. QueryAll 查询不到时返回空切片。
5. NULL 字段转为 nil。
6. []byte 字段转为 string。
7. RowExists 使用 Dialect.BuildRowExistsSQL。
8. CountRows 使用 Dialect.BuildCountRowsSQL。
9. Dialect 增加 BuildRowExistsSQL、BuildCountRowsSQL、Placeholder。
10. MySQLDialect 实现这些方法。
11. 已实现的 Dialect 必须提供这些方法，新增方言也必须补齐对应测试。
12. 添加 query_test.go。
13. MySQL 集成测试只在 MYSQL_TEST_DSN 存在时运行。
14. gofmt。
15. go test ./... 必须通过。
```

验收示例：

```go
func (M20260612_180000SeedAdminUser) Up(ctx context.Context, m *migrate.MigrationContext) error {
	exists, err := m.RowExists(ctx, "user", "username = ?", "admin")
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return m.Schema().
		Insert("user", migrate.Row{
			"username": "admin",
			"email": "admin@example.com",
			"created_at": migrate.Expr("UNIX_TIMESTAMP()"),
		}).
		Exec(ctx)
}
```

最终必须支持：

```go
m.QueryValue(ctx, "SELECT COUNT(*) FROM user")
m.QueryOne(ctx, "SELECT * FROM user WHERE id = ?", 1)
m.QueryAll(ctx, "SELECT * FROM user WHERE status = ?", 10)
m.RowExists(ctx, "user", "username = ?", "admin")
m.CountRows(ctx, "user", "status = ?", 10)
```

### Task 7：实现迁移锁和日志

Prompt：

```txt
请为 Migrator 添加迁移锁和日志。

要求：
1. Migrator 增加 UseLock、LockName、LockTimeoutSeconds。
2. MySQLDialect 实现 GET_LOCK 和 RELEASE_LOCK。
3. Up、Down、Redo、To、Mark 执行前加锁，退出时释放锁。
4. dry-run 模式不需要加锁。
5. 新增 Logger 接口。
6. 默认 logger 输出到 stdout。
7. 日志包含 migration 名称、SQL、执行耗时。
8. 添加测试。
9. go test ./... 必须通过。
```

### Task 8：实现 CLI

Prompt：

```txt
请实现 cmd/migrate CLI。

要求：
1. 支持 up [n]。
2. 支持 down [n]。
3. 支持 redo [n]。
4. 支持 history [n]。
5. 支持 new [n]。
6. 支持 mark VERSION。
7. 支持 to VERSION。
8. 支持 create NAME。
9. 从 DB_DSN 读取数据库连接。
10. 从 MIGRATE_DRY_RUN 读取 dry-run 配置。
11. 从 MIGRATE_TABLE 读取 migration 表名。
12. CLI 错误需要清晰输出并以非 0 状态退出。
13. 根据 `DB_DIALECT` 选择已实现的数据库方言，默认 `mysql`，支持 `sqlite`。
14. go test ./... 必须通过。
```

### Task 9：实现迁移文件生成器

Prompt：

```txt
请实现 migration create 代码生成器。

要求：
1. 命令：go run ./cmd/migrate create create_user_table。
2. 生成 migrations/mYYYYMMDD_HHMMSS_create_user_table.go。
3. 自动生成结构体名，例如 M20260612_120000CreateUserTable。
4. 自动生成 Name、Up、Down 方法。
5. 支持 --fields 参数。
6. fields 示例：
   username:string(64):notNull,email:string(128):notNull,profile:json,bio:longText,status:unsignedTinyInteger:notNull:default(10)
7. 解析 fields 并生成 CreateTable 代码。
8. 不要自动覆盖已有文件。
9. 添加 parser 和 generator 单元测试。
10. go test ./... 必须通过。
```

### Task 10：补 README 和示例

Prompt：

```txt
请补充 README.md 和示例迁移。

要求：
1. README 说明安装、初始化、配置 DB_DSN。
2. README 说明 up/down/redo/history/create 用法。
3. README 明确说明当前版本已经真实支持的数据库。
4. README 不宣称未经实现和测试验证的数据库。
5. README 提供完整迁移示例。
6. README 提供字段 builder 对照表。
7. README 提供 Yii2 到 Go API 对照表。
8. README 提供 MySQL 注意事项。
9. 示例迁移包含 unsigned big primary key、longText、json、index、foreign key、batch insert、RowExists。
10. go test ./... 必须通过。
```

---

## 19. 建议 AGENTS.md

建议在仓库根目录创建 `AGENTS.md`：

```md
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
- Design the dialect layer so the library can grow toward full Yii2-style multi-database support.
- New dialects should be implemented as real, tested dialects with explicit unsupported-operation errors where the database cannot provide an operation.
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
```

---

## 20. 最终验收标准

项目完成后，必须满足：

```bash
go test ./...
```

通过。

可以创建迁移：

```bash
go run ./cmd/migrate create create_article_table \
  --fields="user_id:unsignedBigInteger:notNull,title:string(128):notNull,content:longText,metadata:json,status:unsignedTinyInteger:notNull:default(10)"
```

可以执行迁移：

```bash
DB_DSN='root:password@tcp(127.0.0.1:3306)/test?parseTime=true' go run ./cmd/migrate up
```

可以回滚迁移：

```bash
DB_DSN='root:password@tcp(127.0.0.1:3306)/test?parseTime=true' go run ./cmd/migrate down 1
```

可以预览 SQL：

```bash
MIGRATE_DRY_RUN=1 DB_DSN='root:password@tcp(127.0.0.1:3306)/test?parseTime=true' go run ./cmd/migrate up
```

最终开发者可以写出类似 Yii2 风格的 Go 迁移：

```go
func (M20260612_120000CreateArticleTable) Up(ctx context.Context, m *migrate.MigrationContext) error {
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
			Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")).
			Add("updated_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP")),
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
			"user_id": 1,
			"title": "Hello",
			"slug": "hello",
			"content": "Hello world",
			"metadata": migrate.Expr("JSON_OBJECT('source', 'migration')"),
			"status": 10,
		})
	}

	return plan.Exec(ctx)
}
```

---

## 21. 关键决策总结

1. 通过真实、可测试的 Dialect 实现逐步支持更多数据库。
2. Dialect 接口要为 MySQL、SQLite、PostgreSQL、SQL Server 等数据库保留扩展点。
3. 字段声明使用不可变链式 `ColumnBuilder`。
4. 建表字段必须使用有序 `ColumnList`。
5. DDL / DML 写操作使用 `SchemaPlan` 链式队列。
6. 查询型 DML helper 直接挂在 `MigrationContext` 上，立即返回结果，不进入 `SchemaPlan`。
7. DML 普通值必须参数绑定，`Expression` 才能直接拼 SQL。
8. 所有自动生成 SQL 的占位符必须通过 `Dialect.Placeholder(index)`。
9. 生产环境必须有迁移锁。
10. 所有 Codex 任务都必须 `gofmt` 并通过 `go test ./...`。
