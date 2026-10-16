# 04: 方言支持矩阵、注意事项与新增方言指南

**What to build:** docs/dialects.md 三段式：能力矩阵表（操作 × 四方言：支持 / unsupported 报错 / 怪癖，从一致性矩阵与各 adapter 的 UnsupportedOperationError 清单提炼为用户视角）；四方言注意事项各一节（含 drop 严格契约、postgres Comment 修饰符静默丢弃与非串行主键怪癖、sqlserver 的 ] 转义/非自增主键/enum 与 set 不支持等）；「新增一个方言」扩展指南（baseDialect 内嵌 + 真实 delta 清单 + 一致性矩阵期望列 + 方言注册表登记）。

**Blocked by:** 03（引用迁移编写页的 SchemaPlan/ColumnBuilder 概念）

**Status:** resolved

- [x] 能力矩阵以表格呈现且与当前四个 adapter 实际行为逐格核对（以一致性矩阵用例为事实源）
- [x] unsupported 项全部如实标注（对应 UnsupportedOperationError 可触发的操作）
- [x] 四方言注意事项覆盖已知怪癖与严格契约语义（ADR-0001/0002 的用户视角表述）
- [x] 新增方言指南按当前扩展机制写成步骤化流程（delta 清单、矩阵期望、注册表登记、测试要求）
- [x] 矩阵与注意事项无互相矛盾；页内术语与 CONTEXT.md 一致

## Comments

**验收证据（D04，交付 docs/dialects.md，提交 31041c3；认领提交 32e7da1）：**

- 结构为拍板的三段式：§1 能力矩阵（9 张表、33 行，覆盖约 40 个操作/能力格）→ §2 四方言注意事项 → §3 新增方言五步指南。
- 矩阵事实源：`migrate/dialect_conformance_test.go` 全部 sqlCase/stmtCase 的 want 与 wantUnsupported 断言，逐格对照 adapter 源码（mysql.go / sqlite.go / postgres.go / sqlserver.go）与 base_dialect.go；drop 严格契约（无 IF EXISTS）与未映射类型/sqlserver enum+set 报 unsupported 均按 05/04 轮后的现状核对。
- 怪癖清单（均溯源到 adapter 覆写点或专属测试）：postgres 非串行主键静默丢 PRIMARY KEY、.Comment() 修饰符静默丢弃、AlterColumn 只改类型、pg_advisory_lock 无超时参数；sqlserver `]`→`]]` 与字符串字面量 `'` 翻倍、IDENTITY 主键形态、ConstraintExists 只查 sys.check_constraints（AddPrimaryKeyIfNotExists 恒发语句 / DropPrimaryKeyIfExists 恒跳过）；SQLite 主键列折叠为 INTEGER PRIMARY KEY AUTOINCREMENT、FK/约束常量探针 SELECT 0 的下游后果、无 advisory lock；mysql DropPrimaryKey 忽略 name、Unsigned 非数值静默忽略。
- 扩展指南按当前机制步骤化：实现 Dialect interface（baseDialect 内嵌 + delta 清单表）→ conformanceDialects/wantAll 补期望列（漏格 fail loudly 机制）→ dialect_registry.go 登记（TestDialectRegistryCoversConformanceDialects 防漂移）→ cmd/migrate 驱动 import 与别名表 → `*_TEST_DSN` 集成测试门槛（MYSQL/POSTGRES/SQLSERVER_TEST_DSN 现有三例）。
- 互链：migrations.md（页首导航 + 方言差异速查 + 下一步）、getting-started.md（下一步）均指向 dialects.md；dialects.md 反链 getting-started#dialect / migrations#ifexists / migrations#dryrun / cli#lock。术语与 CONTEXT.md 一致（Dialect/adapter/baseDialect/方言注册表/一致性矩阵/IfExists 探针）。
- 验证：未触碰任何 Go 文件；`go test ./...` 全绿（626 passed in 5 packages）。
