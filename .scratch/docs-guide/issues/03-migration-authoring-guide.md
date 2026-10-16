# 03: 迁移编写指南（API 使用主干）

**What to build:** docs/migrations.md——库集成者的核心参考：迁移注册与命名、Up/Down 结构、ColumnBuilder 全部列类型与修饰符、SchemaPlan 链式操作（含 *IfExists 幂等语义与存在性探针）、查询 helper（RowExists/Insert/Expr 等）、迁移锁与 dry-run、不可逆迁移。术语使用 CONTEXT.md 词汇表（迁移历史表、IfExists 探针等）。

**Blocked by:** 02（入门页概念先行，本页引用之）

**Status:** resolved

- [x] 迁移注册与 Name 约定、Up/Down 结构、不可逆迁移（ErrIrreversibleMigration）成节
- [x] ColumnBuilder 列类型与修饰符清单与 column_builder 实际 API 一一对应（不可变性说明到位）
- [x] SchemaPlan 各操作 + *IfExists 的探针幂等语义（dry-run 直发、已存在跳过、探针出错行为）讲解清楚
- [x] 查询 helper、锁、dry-run 各成节，示例可复制
- [x] API 细节链接 pkg.go.dev，不在页内重复 godoc
- [x] 术语与 CONTEXT.md 一致；页内示例经人工推敲与实际 API 签名吻合

## Comments

### 验收证据（D03，落地提交 ca53584）

**API 清单核对来源**（逐文件通读，未沿用 README 旧文）：

- 迁移结构 / Name 排序 / ErrIrreversibleMigration / 每迁移一事务 / 锁与 dry-run 字段默认值：`migrate/migration.go`、`migrate/migrator.go`（`NewMigrator` 排序，`apply`/`revert` 事务，`withLock` 跳过条件，`UseLock=true`/`LockName="yiimigrate"`/`LockTimeoutSeconds=30`）
- ColumnBuilder 全部构造器（4 主键 + 8 整型 + 6 字符串 + 4 二进制 + 布尔 + 4 数值 + 4 时间 + Json/UUID/Enum/Set）与全部修饰符（18 个，含 `.Nullable()` 查询方法）及 clone 不可变语义：`migrate/column_builder.go`；有序列定义：`migrate/columns.go`
- SchemaPlan 全部操作（表 6 / 列 6 / 索引 4 / 主键 4 / 外键 4 / 注释 4 / DML 4 / Raw / Statements / Exec）与 `existenceGuard` 三态语义（dry-run 直发不查元数据、结果不匹配跳过、探针出错记 err 且后续短路）：`migrate/schema_plan.go`，对照 `schema_plan_test.go`（DryRunSkipsExecution、DryRunIfNotExistsDoesNotQueryMetadata、StopsOnAccumulatedError、DropIfExistsProbesBeforeStrictDrop）
- 查询 helper（RowExists/CountRows/QueryValue/QueryOne/QueryAll + 五个存在判断）：`migrate/query.go`；Row/Expr/SQLStatement/Execute/WithDryRun：`migrate/types.go`
- 方言差异表（Unsigned/Charset/Collate/After/First、Enum/Set 四方言落点、SQLite 外键/约束探针恒返回不存在）：`migrate/mysql.go`、`postgres.go`、`sqlite.go`、`sqlserver.go` 的类型映射与 BuildColumn 前置校验
- pkg.go.dev 模块路径 `github.com/phgors/yiimigrate` 核对自 `go.mod`

**示例可复制性**：页内全部代码段（迁移骨架、不可逆 Down、列定义链、不可变性派生、SchemaPlan 目标风格、RowExists+Insert+Expr）抽取到临时包 `go vet` 编译通过后删除，未入库。

**互链**：`docs/getting-started.md`（导语与"下一步"改指本页）、`docs/cli.md`（导语加本页链接）均已更新；本页对两页的出链只指向其显式锚点（#embed/#dialect/#create/#fields/#lock/#env），本页自建锚点 structure/column-builder/schema-plan/ifexists/query/lock/dryrun。

**测试**：`go test ./...` 全绿（626 passed, 5 packages），未触碰任何 Go 源码（git diff 仅 docs/ 三文件）。

**提交**：85503cb（claim）-> ca53584（docs，--ff-only 合入 main，分支 feat/docs03-authoring 已删）。
