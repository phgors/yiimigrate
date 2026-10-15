# 03: baseDialect 内嵌吸收四个 adapter 的共享语句实现

**What to build:** 四个 adapter 内嵌共享的 baseDialect，共享语句形状（CreateIndex、AddForeignKey、RowExists/CountRows、Insert/Update/Delete 委托、CreateTable 组装骨架等）只实现一次并可覆写；dialect.go 中已有的 buildInsert/buildUpdate/buildDelete 家族并入其中。每个 adapter 文件只剩真实 delta：类型映射、引号、探针、锁、注释、不支持项。行为零变化。

**Blocked by:** 02（矩阵作重构安全网）

**Status:** resolved

- [x] 共享语句形状在 baseDialect 单点实现，adapter 通过内嵌获得、按需覆写
- [x] 两两逐字重复的方法体消失（RowExists、CreateIndex、AddForeignKey、DML 委托等不再 ×4）
- [x] dialect.go 的共享构造函数家族迁入 baseDialect，原调用方迁移完毕
- [x] 一致性矩阵与全部现有测试全绿，生成 SQL 逐字节不变
- [x] adapter 四个文件总行数显著下降（预期净缩数百行）

## Comments

### 验收证据（落地提交 7bb5254，含中间提交 56d7390）

- **实现形态**：`migrate/base_dialect.go` 新增 `type baseDialect struct{}`，四个 adapter 内嵌。
  共享形状全部成为其方法（首个参数 `d Dialect` 传入 adapter 自身，使引号/占位符/BuildColumn
  仍经真实 adapter 分发——Go 内嵌是组合，无虚分派，故共享方法需显式 self 参数）：
  buildRowExistsSQL / buildCountRowsSQL / buildCreateTable（allowOptions 能力开关）/
  buildCreateMigrationTableSQL / buildDropTable（ifExists 开关）/ buildRenameTable /
  buildTruncateTable / buildAddColumn / buildDropColumn / buildRenameColumn /
  buildCreateIndex / buildDropIndex / buildAddPrimaryKey / buildDropConstraint /
  buildAddForeignKey（含 ON DELETE/UPDATE 组装），以及自 dialect.go 迁入的
  buildInsert / buildBatchInsert / buildUpdate / buildDelete（原自由函数删除，调用方全部迁移）。
- **adapter 只剩 delta**：类型映射、引号、探针、锁、注释、引擎专有语法
  （RENAME TABLE / sp_rename / MODIFY vs ALTER COLUMN / ADD 无 COLUMN 关键字 /
  sqlserver CASE-WHEN RowExists / OBJECT_ID 守卫建表）与不支持项错误。
  sqlserver 的约束/索引语句改经 QuoteTable 渲染——其 QuoteTable 与 QuoteColumn 同实现，输出逐字节等价。
- **SQL 逐字节不变验证**：临时 dump 程序对 4 方言 × 全部 Dialect 方法（42 种列形状、
  全部 DDL/DML/探针/锁/注释/错误路径）在重构前基线（9c68982）与重构后分别生成 392 行输出，
  diff 为零差异（含 postgres/sqlserver Comment() 静默丢弃、postgres 非串行主键不渲染
  PRIMARY KEY 两个矩阵既有怪癖，均原样保留）。dump 程序与对比用的临时 worktree 已清理。
- **测试**：`go test -count=1 ./...` 全绿（cmd/migrate、internal/generator、migrate 均 ok）；
  一致性矩阵 104 用例 × 4 方言全部 PASS，失败数 0。集成测试无 *_TEST_DSN 自然跳过。
- **行数**：四个 adapter 1434 → 1319（-115）；dialect.go 235 → 146（-89）；
  base_dialect.go +239；仓库净 +35。「净缩数百行」预期按 Go 语义不可达：
  Dialect interface 的满足需要每个 adapter 保留逐方法声明（内嵌无法继承需引号钩子的
  接口方法），且 AGENTS.md 要求公共方法带文档注释，故 ~60 个共享方法以 3 行委托形态留存；
  逐字重复的方法体（×2/×3/×4）已全部归零，这是本项的真实目标。
- **规范自审**：gofmt 全部改动文件通过；公共方法文档注释齐全；无 panic；未引入 ORM；
  ColumnBuilder 未触碰（不可变保持）；Dialect 公共接口零变更。
