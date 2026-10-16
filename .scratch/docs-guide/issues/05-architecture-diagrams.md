# 05: 架构页——Mermaid 模块图与执行时序图

**What to build:** docs/architecture.md——用 Mermaid 图向库使用者讲清结构：模块图（Migrator、SchemaPlan、Dialect seam、baseDialect、四个 adapter、方言注册表的组合关系）、一次 migrate up 的时序图（锁 → 历史表 → 版本比对 → 执行 → 记录）、CLI 到库的调用流程图。辅以少量文字说明 seam 背后的设计意图（严格契约、幂等归探针）。

**Blocked by:** 01（docs/ 分区归位）

**Status:** resolved

- [x] 模块图以 Mermaid flowchart 呈现，模块/接口/seam 命名与 CONTEXT.md 词汇表一致
- [x] migrate up 时序图覆盖锁、迁移历史表、版本比对、执行、记录各阶段，与 migrator 实际流程一致
- [x] CLI 流程图与 cmd/migrate 实际调用路径一致（含方言注册表取用）
- [x] 全部图用 Mermaid 代码块（不用 ASCII 字符画），GitHub 可直接渲染
- [x] 文字精炼，设计意图表述与 ADR-0001/0002 无冲突

## Comments

**验收证据（2026-10-15）**：docs/architecture.md 落地，提交 `cb65d07`（claim）+ `9aa92f2`（落地，--ff-only 并入 main，分支已删）。

**图与源码核对点清单**：

模块图（flowchart）：
- baseDialect 内嵌于四 adapter：`migrate/mysql.go:10`、`sqlite.go:10`、`postgres.go:10`、`sqlserver.go:11`（`type XDialect struct { baseDialect }`，四文件逐一确认）。
- 方言注册表指向四 adapter：`migrate/dialect_registry.go:11-16`（`dialectRegistry` 含四实例）；消费方为 CLI（`cmd/migrate/main.go:103`）与生成器（`internal/generator/generator.go:168`），全仓 grep 确认无其他 DialectByName/DialectNames 调用方。
- Migrator 依赖 seam：`migrate/migrator.go:15`（`dialect Dialect` 字段），实际调用 `CreateMigrationTableSQL` / `AcquireLockSQL` / `ReleaseLockSQL` / `QuoteTable` / `Placeholder`。
- SchemaPlan 依赖 seam：`migrate/types.go:34-38`（MigrationContext 持 dialect）+ `schema_plan.go` 经 ctx 生成语句。

时序图（migrate up，与 migrator.go 执行顺序逐条核对）：
- 取锁包裹整段：`withLock`（408-421）；dry-run 或 `!UseLock` 跳过锁（409-411）。
- `acquireLock` → `AcquireLockSQL` → `QueryRowContext`，返回值 != 1 → `ErrMigrationLockTimeout`（423-436）。
- `Pending` → `Applied`：dry-run 直接返回空（70-72，不建表不读版本）；正常路径 `EnsureMigrationTable` → `CreateMigrationTableSQL` → `ExecContext`（54-66）→ `loadApplied` SELECT version, apply_time（290-309）。建表语句形状：baseDialect 共享 `CREATE TABLE IF NOT EXISTS`（base_dialect.go:60-65），sqlserver 覆写为 `IF OBJECT_ID(...) IS NULL`（sqlserver.go:247-252）——正文据此措辞「各库只翻译建表语法」。
- 逐迁移循环：dry-run 分支 `NewMigrationContext(WithDryRun(true))` 只打印、不写历史表（313-320）；正常分支 `BeginTx` → `migration.Up` → 同事务 `INSERT version + apply_time` → `Commit`（322-338，失败 `rollbackWithCause`）。
- 释放锁在 fn 之后：`releaseLock` → `ReleaseLockSQL`（438-445）。

CLI 流程图（与 main.go 调用路径核对）：
- 四驱动 blank import 于进程启动注册（main.go:13-16）；`args[0]` 分发：`create` / 七个库命令 / unknown（33-42）。
- `runDBCommand`：`DB_DSN` 必填（115-118）→ `driverAliases` 拼写归一（87-96，sqlite3→sqlite、mssql→sqlserver）→ `migrate.DialectByName`（103）→ 未注册报错并列 `DialectNames()`（110-112）→ `sql.Open`（123）→ `NewMigrator` + `MIGRATE_TABLE` / `MIGRATE_DRY_RUN`（129-133）→ 子命令 switch（136-169）。
- `create` 分支不连库：`--dialect` 默认 `DB_DIALECT` 否则 mysql（44-77），`generator.Generate` 内部同样经 `DialectByName`（generator.go:168）。

**渲染与回归验证**：三张图以 mermaid-cli 12.0.0 实际渲染（SVG exit 0，关键标签齐全，无中文标点/括号冲突）；`go test ./...` 全绿（未触碰 Go 代码）。术语与 CONTEXT.md 一致（Dialect / adapter / baseDialect / 方言注册表 / 迁移历史表 / IfExists 探针 / SchemaPlan / dry-run）；设计意图三条（drop 严格、幂等归探针、unsupported 显式报错）与 ADR-0001/0002 表述一致。互链：architecture.md ↔ getting-started / migrations / dialects / cli 四页（各页「下一步」或导语新增架构页链接）。
