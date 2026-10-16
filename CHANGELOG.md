# Changelog

本仓库所有值得注意的变更都记录在此文件中。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循 [Semantic Versioning](https://semver.org/)。条目中的提交号对应本仓库 git 历史；v0.1.0 之前的奠基工作按主题归并简述。

## [v0.1.0] - 2026-10-16

首个版本：四方言（mysql / sqlite / postgres / sqlserver）迁移库与 `cmd/migrate` CLI，两条严格契约（drop 一律严格、未映射类型显式报错）与配套使用文档。

### Added

- 迁移库与 CLI 奠基（2026-06 前的初期工作，按主题归并）：`Migrator` 版本调度（每迁移一事务、迁移历史表自动创建、迁移锁 `UseLock` / `LockName` / `LockTimeoutSeconds`）、`SchemaPlan` 链式 schema 与 DML 操作、不可变 `ColumnBuilder`、`*IfExists` 存在性探针幂等、查询 helper（`RowExists` / `CountRows` / `QueryValue` / `QueryOne` / `QueryAll`）、`cmd/migrate` 八个子命令与 mysql / sqlite 两个初始方言。
- PostgreSQL 方言 adapter、单元测试与集成测试（`POSTGRES_TEST_DSN` 触发）及 CLI 接入与驱动依赖（1b56776、c46829e、8b1d69a、d0909a0）。
- SQL Server 方言 adapter、单元测试与集成测试（`SQLSERVER_TEST_DSN` 触发）（869d92c、a901360、d0909a0）。
- `DB_DIALECT` 方言别名（`postgresql` / `sqlite3` / `mssql`）与 sqlite 别名覆盖测试（2893db2、8b1d69a）。
- `SchemaPlan.Statements()` 访问器：返回计划累积语句的防御性拷贝，供测试断言与 dry-run 检查（4473841）。
- 迁移历史表 DDL 移入 Dialect seam：新增 `CreateMigrationTableSQL`，mysql / sqlite / postgres 生成 `CREATE TABLE IF NOT EXISTS`，sqlserver 以 `IF OBJECT_ID(...) IS NULL` 判存建表（ac71cee）。
- 四方言一致性矩阵：`migrate/dialect_conformance_test.go` 表驱动用例遍历全部 adapter，强制逐格有期望、无静默缺格；被矩阵覆盖的重复断言清理（535ff8f、5f1098d）。
- `baseDialect` 共享语句实现底座：内嵌于四个 adapter，吸收共享语句形状，adapter 只覆写真实分歧（类型映射、引号、探针、锁、注释、不支持项）（56d7390、7bb5254）。
- 方言注册表（`DialectByName` / `DialectNames`，CLI 与生成器共用的唯一方言入口）与 `create --dialect` 按方言生成迁移模板（ba59eea）。
- 使用文档与开发文档：`docs/` 五页（快速上手、CLI 参考、迁移编写指南、方言参考、架构）与 `dev-docs/`（术语表 `CONTEXT.md`、ADR-0001/0002、工单流）（2df10e3、e6a24bc、2d47166、ca53584、31041c3、9aa92f2）。

### Changed

- **adapter 层删除操作一律严格**：`DropTable` / `DropIndex` 等基线删除不再生成 `IF EXISTS`，对象不存在时由数据库报自然错误；幂等统一由 plan 层 `*IfExists` 存在性探针负责（ADR-0001）。受影响面：postgres 的 `DropTable` / `DropIndex` 与 sqlserver 的 `DropTable` 此前静默跳过缺对象，升级改法见[升级注意](docs/upgrade-notes.md#strict-drop)（d52a0c3）。
- **未映射的列类型统一报 `UnsupportedOperationError`**：四个 adapter 的类型映射 `default` 分支不再把未识别类型名大写透传进 SQL；sqlserver `enum` / `set` 的空串哨兵机制废弃，直接在类型映射报错（ADR-0002）（1cbfab6）。
- `SchemaPlan` 十个 `*IfExists` 守卫折叠为共享的 `existenceGuard`，dry-run 短路 / 探针 / 跳过语义不变（866a58e）。

### Fixed

- SQL Server 对象名含单引号时语句断裂 / 注入：把对象名嵌进字符串字面量的语句（`sp_rename`、注释扩展属性 `sp_add/dropextendedproperty`、历史表建表的 `OBJECT_ID` 探针）现在将 `'` 翻倍转义（d52a0c3）。
