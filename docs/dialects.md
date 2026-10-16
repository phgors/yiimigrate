# 方言参考：能力矩阵、注意事项与新增方言

本页是四个已注册方言（`mysql` / `sqlite` / `postgres` / `sqlserver`）的横向参考：§1 能力矩阵（操作 × 方言，每一格都与 adapter 源码和 `migrate/dialect_conformance_test.go` 的一致性矩阵用例逐格核对）；§2 各方言注意事项（怪癖 + 严格契约语义）；§3 「新增一个方言」的步骤化扩展指南。ColumnBuilder 与 SchemaPlan 的概念定义在[迁移编写指南](migrations.md)，本页直接引用、不重复。

- 怎么选方言、DSN 怎么写：[快速上手 - 选择方言与 DSN](getting-started.md#dialect)。
- 迁移锁的通用行为（哪些命令取锁、锁名与超时默认值）：[CLI 参考 - 迁移锁](cli.md#lock)。

## 图例与两条全局契约

<a id="legend"></a>

矩阵单元格的含义：

| 标记 | 含义 |
| --- | --- |
| ✓ | 支持：该方言为这个操作生成有效 SQL |
| ✗ `操作名` | 报不支持：生成语句时返回 `*UnsupportedOperationError`，错误信息为 `"<方言> does not support <操作名>"` |
| ⚠ 简注 | 支持但有怪癖：语句照常生成，细节见 §2 对应方言小节 |

两条契约对四个方言一致成立，先记住它们，矩阵里的很多格就不再意外：

- **drop 一律严格**：`DropTable`、`DropIndex`、`DropPrimaryKey`、`DropForeignKey` 等不带 `IfExists` 的基线删除操作**永不**生成 `IF EXISTS`——对象不存在时由数据库报自然错误。mysql / sqlite / postgres 原生支持 `IF EXISTS` 语法但同样不用它，为的是四个方言错误模式一致。幂等的唯一来源是 plan 层的 `*IfExists` 变体（先探测对象存在性再决定是否发语句），语义见[迁移编写指南 - `*IfExists` 与存在性探针](migrations.md#ifexists)。
- **未映射的列类型一律报错**：ColumnBuilder 的类型构造器映射到各方言的类型表，映射未命中时**不做透传**，统一报 `"<方言> does not support <类型名>"`（如 `mysql does not support geometry`）——四个方言同一种失败模式。sqlserver 的 `m.Enum` / `m.Set` 同理（`sqlserver does not support enum` / `set`）。

这两种错误在 dry-run 下同样会暴露（见[迁移编写指南 - dry-run](migrations.md#dryrun)）：不依赖某个数据库时，用 dry-run 扫一遍即可提前发现。

## 1. 能力矩阵

<a id="matrix"></a>

操作按 SchemaPlan 的链式方法分组（完整方法签名见[迁移编写指南 - 操作清单](migrations.md#schema-plan)）；所有 `*IfExists` 变体的能力等同其基线操作（探针行为差异见[下文](#probes)）。

### 表

| 操作 | mysql | sqlite | postgres | sqlserver |
| --- | --- | --- | --- | --- |
| `CreateTable`（不带 options） | ✓ | ✓ | ✓ | ✓ |
| `CreateTable` 的 `options` 参数 | ✓ 原样拼接 | ✗ `TABLE OPTIONS` | ✓ 原样拼接 | ✗ `TABLE OPTIONS` |
| `DropTable` / `DropTableIfExists` | ✓ 严格 | ✓ 严格 | ✓ 严格 | ✓ 严格 |
| `RenameTable` | ✓ `RENAME TABLE` | ✓ | ✓ | ⚠ `sp_rename`，对象名嵌在字符串字面量里 |
| `TruncateTable` | ✓ | ✗ `TRUNCATE TABLE` | ✓ | ✓ |

### 列

| 操作 | mysql | sqlite | postgres | sqlserver |
| --- | --- | --- | --- | --- |
| `AddColumn` | ✓ `ADD COLUMN` | ✓ `ADD COLUMN` | ✓ `ADD COLUMN` | ⚠ `ADD`（无 `COLUMN` 关键字） |
| `AlterColumn` | ✓ 完整定义（`MODIFY COLUMN`） | ✗ `ALTER COLUMN` | ⚠ 只改类型，其余修饰符丢弃 | ✓ 完整定义（`ALTER COLUMN`） |
| `DropColumn` | ✓ | ✗ `DROP COLUMN` | ✓ | ✓ |
| `RenameColumn` | ✓ | ✗ `RENAME COLUMN` | ✓ | ⚠ `sp_rename`，对象名嵌在字符串字面量里 |

### 索引

| 操作 | mysql | sqlite | postgres | sqlserver |
| --- | --- | --- | --- | --- |
| `CreateIndex`（普通 / 唯一 / 复合 / 表达式） | ✓ | ✓ | ✓ | ✓ |
| `DropIndex` | ✓ 带 `ON table` | ⚠ 按名删，`table` 参数被忽略 | ⚠ 按名删，`table` 参数被忽略 | ✓ 带 `ON table` |

表达式索引列（如 `LOWER(title)`、JSON 路径 `data->>'$.user'`）在四个方言中原样保留、不加引号。

### 主键与外键

| 操作 | mysql | sqlite | postgres | sqlserver |
| --- | --- | --- | --- | --- |
| `AddPrimaryKey` | ✓ | ✗ `ADD PRIMARY KEY` | ✓ | ✓ |
| `DropPrimaryKey` | ⚠ `DROP PRIMARY KEY`，`name` 参数不进 SQL | ✗ `DROP PRIMARY KEY` | ✓ `DROP CONSTRAINT name` | ✓ `DROP CONSTRAINT name` |
| `AddForeignKey` / `DropForeignKey` | ✓ | ✗ `ADD FOREIGN KEY` / `DROP FOREIGN KEY` | ✓ | ✓ |

### 注释

| 操作 | mysql | sqlite | postgres | sqlserver |
| --- | --- | --- | --- | --- |
| 列修饰符 `.Comment(text)` | ✓ 内联 `COMMENT '...'` | ✗ `COLUMN COMMENT` | ⚠ 静默丢弃（渲染为空） | ⚠ 静默丢弃（渲染为空） |
| `Add/DropCommentOnColumn` | ✓ `MODIFY COLUMN ... COMMENT` | ✗ `COLUMN COMMENT` | ✓ `COMMENT ON COLUMN` | ⚠ 扩展属性（`sp_add/dropextendedproperty`） |
| `Add/DropCommentOnTable` | ✓ `COMMENT = '...'` | ✗ `TABLE COMMENT` | ✓ `COMMENT ON TABLE` | ⚠ 扩展属性 |

postgres 与 sqlserver 的列注释走独立语句：`.Comment()` 修饰符在 `CreateTable` / `AddColumn` 里不渲染任何内容（不报错），要真正落库用 `AddCommentOnColumn`。

### 列类型与修饰符

`migrations.md` 的[方言差异速查](migrations.md#dialect-gaps)是这组矩阵的速览版，下面是逐格完整版：

| 能力 | mysql | sqlite | postgres | sqlserver |
| --- | --- | --- | --- | --- |
| `.Unsigned()` | ⚠ 数值类型追加 `UNSIGNED`；非数值类型静默忽略 | ⚠ 一律静默忽略 | ✗ `UNSIGNED` | ✗ `UNSIGNED` |
| `.Charset` / `.Collate` / `.After` / `.First` | ✓ | ✗（`CHARACTER SET` / `COLLATE` / `AFTER` / `FIRST`） | ✗ 同左 | ✗ 同左 |
| `.Comment(text)` | ✓ 内联 | ✗ `COLUMN COMMENT` | ⚠ 静默丢弃 | ⚠ 静默丢弃 |
| `.PrimaryKey()` 标记 | ✓ 内联渲染 | ⚠ 主键列折叠为 `INTEGER PRIMARY KEY AUTOINCREMENT`（声明的类型被替换） | ⚠ 仅串行主键（`SERIAL` / `BIGSERIAL`）渲染；非串行主键静默丢失 `PRIMARY KEY` | ⚠ 自增主键固定 `IDENTITY(1,1) PRIMARY KEY` 形态；非自增主键正常内联 |
| `.GeneratedAs`（`.Stored` / `.Virtual`） | ✓ | ✓ | ⚠ 仅存储式：不加 `.Stored()` 时生成的语句会被 PostgreSQL 拒绝 | ✗ `GENERATED AS` |
| `.Check` / `.Unique` / `.DefaultValue` / `.DefaultExpression` / `.Append` | ✓ | ⚠ 普通列 ✓；主键列折叠为固定形式 | ⚠ serial 主键列丢弃可空性与默认值修饰符 | ✓ |
| `m.Enum(...)` / `m.Set(...)` | ✓ 原生 `ENUM(...)` / `SET(...)` | ⚠ 落为 `TEXT`，取值列表不进 SQL | ⚠ 落为 `TEXT`，取值列表不进 SQL | ✗ `enum` / `set` |
| 未映射的类型名 | ✗ 报类型名 | ✗ 报类型名 | ✗ 报类型名 | ✗ 报类型名 |

### 数据变更与查询 helper

| 操作 | mysql | sqlite | postgres | sqlserver |
| --- | --- | --- | --- | --- |
| `Insert` / `BatchInsert` / `Update` / `Delete` | ✓ | ✓ | ✓ | ✓ |
| `m.RowExists` / `m.CountRows` | ✓ | ✓ | ✓ | ⚠ `RowExists` 包成 `SELECT CASE WHEN EXISTS(...) THEN 1 ELSE 0 END` |

DML 四个操作四个方言都支持，差异只在引号与占位符风格：`` `x` `` + `?`、`"x"` + `?`、`"x"` + `$N`、`[x]` + `@pN`。

### 迁移锁

| 能力 | mysql | sqlite | postgres | sqlserver |
| --- | --- | --- | --- | --- |
| `AcquireLockSQL` / `ReleaseLockSQL` | ✓ `GET_LOCK` / `RELEASE_LOCK`，超时参数生效 | ✗ `ADVISORY LOCK` | ⚠ `pg_advisory_lock`，无超时参数（见 [postgres 注意事项](#postgres)） | ✓ `sp_getapplock` / `sp_releaseapplock`，超时参数生效 |

### `*IfExists` 探针的实现

<a id="probes"></a>

`*IfExists` 变体与 `m.TableExists` 等 helper 共用同一组探针（`SELECT COUNT(*)` 形态），各方言查的目录不同：

| 探针 | mysql | sqlite | postgres | sqlserver |
| --- | --- | --- | --- | --- |
| `TableExists` | `information_schema.tables`（`DATABASE()` 限定） | `sqlite_master`（含视图） | `information_schema.tables`（`current_schema()` 限定） | `sys.tables`（`SCHEMA_NAME()` 限定） |
| `ColumnExists` | `information_schema.columns` | `pragma_table_info` | `information_schema.columns` | `sys.columns` ⋈ `sys.tables` |
| `IndexExists` | `information_schema.statistics` | `sqlite_master` | `pg_indexes` | `sys.indexes` ⋈ `sys.tables` |
| `ForeignKeyExists` | `information_schema.table_constraints`（FK） | ⚠ 恒返回「不存在」（`SELECT 0`） | `information_schema.table_constraints`（FK） | `sys.foreign_keys` |
| `ConstraintExists` | `information_schema.table_constraints` | ⚠ 恒返回「不存在」（`SELECT 0`） | `information_schema.table_constraints` | ⚠ 只查 `sys.check_constraints`，主键不在其中 |

两个探针边界直接改变 `*IfExists` 的行为，使用前值得过一眼：

- **SQLite**：没有命名约束的元数据可查，`ForeignKeyExists` / `ConstraintExists` 恒返回「不存在」。因此 `DropForeignKeyIfExists` / `DropPrimaryKeyIfExists` 在 SQLite 上永远跳过；`AddForeignKeyIfNotExists` 每次都会尝试生成语句——随即命中 `ADD FOREIGN KEY` 的不支持错误（[迁移编写指南](migrations.md#ifexists)记录了这一边界）。
- **SQL Server**：`ConstraintExists` 只查 `sys.check_constraints`，主键约束名（在 `sys.key_constraints` 里）查不到。因此 `AddPrimaryKeyIfNotExists` 的探针恒「不存在」（每次都发语句，重复添加由数据库的严格错误兜底），`DropPrimaryKeyIfExists` 永远跳过。

## 2. 各方言注意事项

<a id="notes"></a>

### MySQL

<a id="mysql"></a>

mysql 是默认方言，也是 MySQL 专属语法唯一全数可用的一方：`.Charset` / `.Collate` / `.After` / `.First` 修饰符、内联 `.Comment()`、`CreateTable` 的 `options` 参数（惯用 `"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"`）。注意项：

- **`.Unsigned()` 只对数值类型生效**：tiny/small/int/bigInteger、float、double、decimal、money 追加 `UNSIGNED`；其他类型静默忽略（不报错）。
- **`DropPrimaryKey` 忽略 `name` 参数**：生成 `ALTER TABLE ... DROP PRIMARY KEY`——MySQL 按约束种类定位主键，不吃约束名。
- **类型映射习惯**：`Boolean` → `TINYINT(1)`、`Money` → `DECIMAL(p, s)`（不映射 MySQL 原生 `MONEY`）、`UUID` → `CHAR(36)`；整型显示宽度原样保留（`Integer(11)` → `INT(11)`）。
- **`Binary()` 不带尺寸生成裸 `VARBINARY`**（无默认宽度），建表前显式传尺寸更稳妥。
- **事务边界受引擎限制**：MySQL 的 DDL 会隐式提交，迁移事务里已执行的 DDL 无法回滚（执行模型见[迁移编写指南](migrations.md#structure)）。
- 锁与探针：`GET_LOCK(?, ?)` / `RELEASE_LOCK(?)`（`LockTimeoutSeconds` 生效）；探针查 `information_schema` 并用 `DATABASE()` 限定当前库。

### SQLite

<a id="sqlite"></a>

sqlite 是唯一的文件 / 内嵌数据库，也是不支持面最大的一方。直接报不支持的操作：`TruncateTable`、`AlterColumn`、`DropColumn`、`RenameColumn`、`Add/DropPrimaryKey`、`Add/DropForeignKey`、列与表注释、表 options、advisory lock。

- **类型系统折叠为五类亲和性**：所有整型（含 `Boolean`）→ `INTEGER`（显示宽度丢弃）；所有文本型 + 日期时间型（`Date` / `DateTime` / `Time` / `Timestamp`）+ `Json` / `UUID` / `Enum` / `Set` → `TEXT`；所有二进制 → `BLOB`；`Float` / `Double` → `REAL`；`Decimal` / `Money` → `NUMERIC(p, s)`。`Enum` / `Set` 的取值列表不进 SQL。
- **主键列折叠为固定形式**：主键列一律渲染 `INTEGER PRIMARY KEY AUTOINCREMENT`——声明的列类型被替换（`m.String(36).PrimaryKey()` 在 SQLite 上也变成 `INTEGER` 列），除 `.Append()` 外的修饰符（可空性、默认值、UNIQUE、CHECK、生成列）全部丢弃。
- **改表结构要重建表**：列的类型变更、删除、重命名本库在 SQLite 上均不支持。惯用替代路径：`CreateTable` 建新表 → `Raw` / `Insert ... Select` 搬数据 → `DropTable` 旧表 → `RenameTable` 改名。
- **外键只能写进建表语句**：`AddForeignKey` 报不支持，因此 SQLite 上的外键约束只能作为 `CreateTable` 的表级 / 列级子句出现（借 `Raw` 或列的 `.Append()` 拼入）；需要启用外键强制时用 `Raw` 执行 `PRAGMA foreign_keys = ON`。
- **没有迁移锁**：取锁直接报 `sqlite does not support ADVISORY LOCK`。写操作命令要么用 `MIGRATE_DRY_RUN=1` 预览（dry-run 不取锁），要么内嵌 `Migrator` 并设 `UseLock = false`（见[快速上手 - 集成到自己的项目](getting-started.md#embed)）。

### PostgreSQL

<a id="postgres"></a>

- **非串行主键会静默丢失 `PRIMARY KEY`**：只有 `m.PrimaryKey()` / `m.BigPrimaryKey()`（integer / bigInteger + 自增）渲染为 `SERIAL` / `BIGSERIAL PRIMARY KEY`；其他主键列（如 `m.UUID().PrimaryKey()`、`m.String(36).PrimaryKey()`）只渲染类型本身，**既不渲染 `PRIMARY KEY` 也不报错**。需要非自增主键时改用 `CreateTable` 之后 `AddPrimaryKey`，或接受串行主键。这是四个方言里唯一会静默丢主键标记的。
- **serial 主键列丢弃可空性与默认值修饰符**：`SERIAL` 自带 `NOT NULL DEFAULT nextval(...)`，显式设置的 `.Null()` / `.NotNull()` / `.DefaultValue` / `.DefaultExpression` 在 serial 列上不渲染。
- **`.Comment()` 修饰符静默丢弃**：postgres 的列 / 表注释走独立语句（`COMMENT ON ...`），修饰符在 `CreateTable` / `AddColumn` 里渲染为空。要落库注释用 `AddCommentOnColumn` / `AddCommentOnTable`。
- **`AlterColumn` 只改类型**：生成 `ALTER TABLE ... ALTER COLUMN ... TYPE ...`，builder 上的可空性、默认值等修饰符静默丢弃。需要 `SET NOT NULL` / `SET DEFAULT` 用 `Raw` 补独立语句。
- **生成列只支持存储式**：PostgreSQL 只接受 `GENERATED ALWAYS AS (...) STORED`；不加 `.Stored()` 生成的语句会被服务器拒绝。
- **`Enum` / `Set` 落为 `TEXT`**：取值列表不进 SQL，需要取值约束时自己加 `.Check(...)`（postgres 原生 `CREATE TYPE` 枚举本库未集成，可用 `Raw`）。
- **`.Unsigned()` 报不支持**：换更大的类型，或用 `CHECK (col >= 0)` 约束。
- **类型映射习惯**：`TinyInteger` → `SMALLINT`、`Money` → `MONEY`、`Json` → `JSONB`、`Double` → `DOUBLE PRECISION`；表 options 支持原样拼接（如 `WITH (fillfactor = 90)`）。
- **锁无超时**：`pg_advisory_lock(hashtext($1))` 没有等待超时参数，`LockTimeoutSeconds` 在 postgres 上不生效——拿不到锁就一直等（除非 context 被取消）。mysql / sqlserver 的锁实现都遵守该超时并报 `migration lock timeout`。

### SQL Server

<a id="sqlserver"></a>

- **标识符与字面量转义**：标识符用方括号括起，`]` 转义为 `]]`（`name]` → `[name]]]`）；`RenameTable` / `RenameColumn` 与注释操作把对象名嵌进字符串字面量，其中的 `'` 一律翻倍（`weird'table` → `'weird''table'`）。
- **主键两种形态**：自增主键（`m.PrimaryKey()` / `m.BigPrimaryKey()`）固定渲染 `INT|BIGINT IDENTITY(1,1) PRIMARY KEY NOT NULL`；非自增主键正常内联（`m.String(36).PrimaryKey()` → `NVARCHAR(36) NOT NULL PRIMARY KEY`）——与 postgres 的静默丢弃对比鲜明，跨这两个方言写非自增主键时优先在这里验证。
- **`m.Enum` / `m.Set` 报不支持**：SQL Server 没有等价类型，用 `CHECK` 约束或参考表建模替代。
- **专属不支持项**：`GENERATED AS`（计算列用 `Raw`）、`UNSIGNED`、`CHARACTER SET` / `COLLATE` / `AFTER` / `FIRST`、表 options。
- **类型映射习惯**：字符串全走 Unicode——`String` → `NVARCHAR`、`Char` → `NCHAR`、文本家族与 `Json` → `NVARCHAR(MAX)`；`Timestamp` → `DATETIME2`（不映射 SQL Server 已废弃的 `timestamp`/`rowversion` 类型）；`UUID` → `UNIQUEIDENTIFIER`。
- **语法差异**：`AddColumn` 生成 `ALTER TABLE ... ADD ...`（无 `COLUMN` 关键字）；`RenameTable` / `RenameColumn` 走 `sp_rename`；迁移历史表没有 `CREATE TABLE IF NOT EXISTS`，改为先探 `IF OBJECT_ID(N'...', 'U') IS NULL` 再建表（效果相同，见[快速上手](getting-started.md)）；`RowExists` 包成 `SELECT CASE WHEN EXISTS(...) THEN 1 ELSE 0 END`。
- **约束探针只查 CHECK**：`ConstraintExists` 查 `sys.check_constraints`，主键 / 唯一约束名查不到——`AddPrimaryKeyIfNotExists` 因此每次都发语句、`DropPrimaryKeyIfExists` 永远跳过（见[上文](#probes)）。
- **`Binary()` 不带尺寸生成裸 `VARBINARY`**，与 mysql 相同，显式传尺寸更稳妥。
- 锁：`sp_getapplock`（`@LockMode = 'Exclusive'`，`@LockTimeout` 接 `LockTimeoutSeconds`）/ `sp_releaseapplock`；占位符风格 `@pN`。

## 3. 新增一个方言

<a id="extend"></a>

本库按「adapter 只写真实分歧」的机制扩展方言：共享语句形状全部在 `baseDialect`（`migrate/base_dialect.go`），四个现有 adapter 内嵌它并只覆写类型映射、引号、探针、锁、注释与不支持项；一致性矩阵（`migrate/dialect_conformance_test.go`）用表驱动用例强制每个方言逐格有期望；方言注册表（`migrate/dialect_registry.go`）是方言知识的唯一入口。新增一个方言走下面五步。

### 第 1 步：实现 adapter（`migrate/<name>.go`）

定义内嵌 `baseDialect` 的无状态结构体并实现 `Dialect` interface（`migrate/dialect.go`）。最小骨架：

```go
// ExampleDialect generates SQL for Example.
type ExampleDialect struct {
	baseDialect
}

// Name returns the dialect name.
func (d ExampleDialect) Name() string { return "example" }

// QuoteTable quotes a table name.
func (d ExampleDialect) QuoteTable(name string) string { return quoteName(name, `"`) }

// QuoteColumn quotes a column name.
func (d ExampleDialect) QuoteColumn(name string) string { return quoteName(name, `"`) }

// QuoteIndexColumn quotes an index column unless it is an expression.
func (d ExampleDialect) QuoteIndexColumn(name string) string { return quoteIndexColumn(d, name) }

// Placeholder returns a placeholder.
func (d ExampleDialect) Placeholder(index int) string { return "?" }
```

只需要写 delta，逐项对照现有 adapter 决定覆写什么：

| 决策点 | 做法 | 参照 |
| --- | --- | --- |
| 类型映射 + `BuildColumn` | `switch typeName` 映射到原生类型；**`default` 分支必须 `return "", unsupported(d.Name(), c.typeName)`**，绝不透传 | 各 adapter 的 `xxxType` |
| 不支持的修饰符 | 显式 `unsupported(d.Name(), "UNSIGNED")`（照抄真实操作名，错误信息可读） | `postgres.go` BuildColumn 开头 |
| 支持但丢弃的修饰符 | 正常渲染、不报错，但必须记进方言注意事项并补专属测试 | postgres 的 `.Comment()` |
| 引号 / 占位符 | 复用 `quoteName` / `quoteIndexColumn` helper，传自家引号字符 | 全部 adapter |
| 存在性探针 × 5 | `TableExistsSQL` ... `ConstraintExistsSQL`，查该库的目录；查不了命名约束就学 SQLite 返回常量探针并在文档写明后果 | `sqlite.go` |
| 迁移锁 | `AcquireLockSQL` / `ReleaseLockSQL`；没有 advisory lock 就 `unsupported(d.Name(), "ADVISORY LOCK")` | `sqlite.go` |
| 注释 × 4 | 按该库机制实现；完全没有就报不支持 | mysql / postgres / sqlserver 三种范式 |
| 删除类操作 | **一律严格**：永不拼 `IF EXISTS`，缺对象交给数据库自然报错 | `baseDialect.buildDropTable` |
| 其余共享形状 | 直接转发 `baseDialect`：`return d.buildInsert(d, table, row)` 这样把自身作为 `d` 传回 | 全部 adapter |

`baseDialect` 的共享方法都带一个 `d Dialect` 参数用于显式回传外层 adapter——Go 内嵌是组合不是继承，`baseDialect` 上的方法**不会**自动调用外层覆写，转发时务必传 `d` 自身而不是省略。每个导出方法写 Go doc 注释（仓库约定）。

### 第 2 步：补一致性矩阵期望列

打开 `migrate/dialect_conformance_test.go`：

1. 把新 adapter 加进 `conformanceDialects` 切片——加进去的瞬间，矩阵的**每个用例**都会遍历到新方言。
2. 扩展 `wantAll` 的固定参数表（当前按 mysql、sqlite、postgres、sqlserver 顺序四元）。
3. 给每个 `sqlCase` / `stmtCase` 的新方言列填 `want`（精确 SQL）或 `wantUnsupported`（操作名）。漏填会直接失败——runner 对没有期望的方言报 `case has no expectation for dialect`，这正是矩阵防止「悄悄缺格」的机制。
4. 真实分歧（探针、锁、引号怪癖）写进独立的 `<name>_test.go`，文件头注释注明「只保留真实分歧」（现有 `postgres_test.go` / `sqlserver_test.go` / `dialect_test.go` 是模板）。

### 第 3 步：方言注册表登记

在 `migrate/dialect_registry.go` 的 `dialectRegistry` map 里加一行 `"example": ExampleDialect{}`。`DialectByName`（查找不区分大小写、忽略首尾空白）与 `DialectNames`（排序输出）随即自动生效；`TestDialectRegistryCoversConformanceDialects` 会强制「矩阵遍历到的每个方言都已注册」，两边不会漂移。

CLI 的方言别名（如 `mssql`）加在 `cmd/migrate/main.go` 的别名表里；`create` 生成器（`internal/generator`）只接受规范名，列表来自 `migrate.DialectNames()`，无需改动。

### 第 4 步：CLI 驱动接入

`cmd/migrate/main.go` 以 blank import 内置全部方言的 `database/sql` 驱动——新方言的驱动 import 也要加在这里，CLI 才能连库。

### 第 5 步：集成测试门槛

沿用 `*_TEST_DSN` 约定（现有三例：`MYSQL_TEST_DSN` / `POSTGRES_TEST_DSN` / `SQLSERVER_TEST_DSN`）：新建 `<name>_integration_test.go`，环境变量未设置时 `t.Skip`，设置时才连真实库执行。两条硬约束：

- 单元测试（一致性矩阵 + 专属测试文件）**不得要求真实数据库**；
- 不设任何 `*_TEST_DSN` 时 `go test ./...` 必须全绿（改动的 Go 文件记得 `gofmt`）。

### 完成标准

- 矩阵每个用例对新方言都有期望（`want` 或 `wantUnsupported`），无遗漏格；
- 数据库给不了的操作全部显式报 `UnsupportedOperationError`，没有静默降级；删除类一律严格；
- 怪癖（丢弃的修饰符、常量探针、锁差异）各有专属测试覆盖，并记入本页 §2 新方言小节。

## 下一步

- [迁移编写指南](migrations.md)：ColumnBuilder、SchemaPlan 与查询 helper 的 API 主参考。
- [快速上手](getting-started.md)：从零跑通第一次迁移的完整闭环。
- [CLI 参考](cli.md)：全部子命令、旗标、环境变量与报错行为。
- [架构](architecture.md)：模块图、一次 migrate up 的执行时序与 CLI 调用流程。
- [pkg.go.dev 文档](https://pkg.go.dev/github.com/phgors/yiimigrate/migrate)：`migrate` 包全部导出 API 的 godoc。
