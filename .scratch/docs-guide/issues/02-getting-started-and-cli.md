# 02: 入门指南与 CLI 参考（从零到跑通迁移）

**What to build:** 新用户从安装库到在四种方言任一下跑通首次迁移的完整路径：docs/getting-started.md（安装、DSN 配置、方言选择、历史表自动创建、首次 up/down）与 docs/cli.md（全命令与旗标参考：create/up/down、--dialect、--dry-run、环境变量 DB_DIALECT 等、退出行为）。两页互相链接，代码示例与 v0.1.0 实际行为一致。

**Blocked by:** 01（docs/ 分区归位后再落使用文档）

**Status:** resolved

- [x] getting-started.md 覆盖：安装、四方言 DSN 示例、方言选择（含 DialectByName 注册表口径）、首次迁移完整可复制流程
- [x] cli.md 覆盖全部子命令与旗标（含 07 轮新增的 create --dialect 与按方言模板行为），逐项与 cmd/migrate 实际实现核对
- [x] 所有代码示例可直接复制使用，命令输出描述与真实行为一致
- [x] 中文行文；与 README 现有内容的分工清晰（此处详、README 简）
- [x] 两页间及页内锚点链接有效

## Comments

**验收证据（2026-10-16，提交 2d47166，已 --ff-only 合入 main）**

事实核对来源清单：

- `cmd/migrate/main.go`：8 个子命令、`create --fields/--dialect`、`DB_DSN`/`DB_DIALECT`/`MIGRATE_TABLE`/`MIGRATE_DRY_RUN`（值为 `1/true/TRUE/yes`）、driverAliases 别名表（`sqlite3/postgresql/mssql`）、错误消息与退出码（stderr + exit 1）、DB_DSN 先于 mark/to 参数校验。
- `migrate/dialect_registry.go`：注册表四规范名 mysql/postgres/sqlite/sqlserver（DialectByName/DialectNames），大小写与首尾空白处理。
- `migrate/migrator.go` + `migrate/migration.go` + `migrate/types.go`：默认历史表 `migration`（version varchar(180) PK + apply_time int）、自动建表、按 Name() 排序、事务执行、锁默认开启（yiimigrate/30s）、dry-run 不取锁不执行不写历史。
- `internal/generator/generator.go`：文件名 `mYYYYMMDD_HHMMSS_name`、旗标模板差异表、无 fields 时 Raw 占位 + ErrIrreversibleMigration、表名剥离 create_/_table、拒覆盖。
- 实测（本机运行 CLI）：`create` 三种变体的真实输出与生成文件全文已入文；`--dialect=mssql` 报 `unknown dialect`；`DB_DIALECT=postgresql` + create 报 unknown dialect（别名不适用于 --dialect）；sqlite 实库 `up` 报 `sqlite does not support ADVISORY LOCK`、`MIGRATE_DRY_RUN=1` 正常预览；`new`/`history`/无参/未知命令/缺 DB_DSN 的输出与退出码逐条核对。
- 重要修正：`create` 旗标必须写在 NAME 之前（Go flag 解析在首个非旗标参数处停止），README 旧示例顺序会 usage 报错，两页均已按真实行为写。

测试摘要：`go test ./...` 全部通过（cmd/migrate、internal/generator、migrate ok；docs-only 改动未触碰代码与测试）。

遗留 / 待人工项（不在本票范围）：

- README 的 CLI/环境变量段落已过时（如宣称 DB_DIALECT 仅 mysql/sqlite/sqlite3，且 create 旗标顺序错误），与本页重叠，建议后续工单把 README 对应段精简为指向 docs/ 的链接。
- 观察项：`migrate/migrator.go` 的 `acquireLock` 把锁查询结果 Scan 进 int64；PostgreSQL 的 `pg_advisory_lock` 返回 void，真实 PG 上该 Scan 可能失败，且现有 PG 集成测试未覆盖锁路径——待人工在真实 PG 验证（文档未对此做超出代码事实的断言）。
- `examples/iphubtest/` 目前未跟踪；文档只引用了已跟踪的 `examples/migrations/` 示例，未链接未跟踪内容。
