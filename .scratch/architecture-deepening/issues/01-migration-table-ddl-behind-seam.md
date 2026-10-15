# 01: 历史表 DDL 收回 Dialect seam（修 SQL Server 活 bug）

**What to build:** 在四种支持的数据库上，`Migrator.Up/Down/Applied` 首次运行都能自动创建迁移历史表：历史表 DDL 由各 adapter 通过 Dialect interface 的新方法生成，Migrator 不再自带任何内联 DDL。SQL Server 用户从「第一条迁移就语法错误」变为可正常跑通迁移。

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] Dialect interface 新增迁移历史表建表方法（列结构 version varchar(180) PK + apply_time int 由 Migrator 侧定义，adapter 只翻译语法）
- [x] 四个 adapter 各自实现：mysql/sqlite/postgres 用原生 IF NOT EXISTS，sqlserver 用 IF OBJECT_ID(...) IS NULL 形式
- [x] Migrator 建历史表的调用全部穿过该方法，源码中不再有内联的 CREATE TABLE 语句
- [x] SQL Server 的历史表建表 SQL 有不依赖真实数据库的单元测试断言
- [x] `go test ./...` 全绿

## Comments

**2026-10-15 验收证据（agent）**

- 接口：`Dialect.CreateMigrationTableSQL(table string) (string, error)`（migrate/dialect.go），签名与拍板决策一致。
- adapter 实现：mysql/sqlite/postgres 生成 `CREATE TABLE IF NOT EXISTS ... ("version" varchar(180) NOT NULL PRIMARY KEY, "apply_time" int NOT NULL)`（各自引号风格）；sqlserver 生成 `IF OBJECT_ID(N'[migration]', 'U') IS NULL BEGIN CREATE TABLE [migration] ([version] varchar(180) NOT NULL PRIMARY KEY, [apply_time] int NOT NULL) END`。列结构与原 MySQL 输出逐字节一致，无行为回归。
- Migrator：`createMigrationTableSQL()` 已删除，`EnsureMigrationTable` 仅取 dialect 语句后 `ExecContext`；grep 确认 migrate/migrator.go 及 cmd/、internal/ 无内联 CREATE TABLE。
- 单元测试（不依赖真实数据库，全部通过）：
  - `TestSQLServerCreateMigrationTableSQL`（sqlserver_test.go，工单要求的 SQL Server 断言）
  - `TestMySQLCreateMigrationTableSQL` / `TestSQLiteCreateMigrationTableSQL`（dialect_test.go）、`TestPostgreSQLCreateMigrationTableSQL`（postgres_test.go）
  - `TestMigratorEnsureMigrationTableExecutesDialectSQL` / `TestMigratorEnsureMigrationTableUsesConfiguredTableName`（migrator_test.go，fake driver 断言实际执行的 SQL 与自定义表名穿缝）
- `go test ./...` 全绿（integration 测试按 *_TEST_DSN 未设置自然跳过）；gofmt 无差异。
- 落地：分支 feat/01-migration-table-ddl 以 --ff-only 合入 main，提交 `ac71cee`，分支已删除。
- 待人工项：如需真实 SQL Server 上验证 OBJECT_ID 形态，设置 `SQLSERVER_TEST_DSN` 后跑 `go test ./migrate/ -run Integration`。
