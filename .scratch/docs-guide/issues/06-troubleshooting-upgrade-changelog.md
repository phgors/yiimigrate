# 06: 排错指南、升级注意与 CHANGELOG

**What to build:** 三件配套：docs/troubleshooting.md（常见错误目录：UnsupportedOperationError 的读法与处置、迁移锁未释放、历史表问题、dry-run 与真实执行差异、各方言连接错误示例）、docs/upgrade-notes.md（v0.1.0 三处行为变化的说明与迁移建议：drop 严格化、未映射类型报错、sqlserver enum/set 报错）、根目录 CHANGELOG.md（Keep a Changelog 风格，首条 v0.1.0，此前工作按主题归并简述）。

**Blocked by:** 04（错误目录需引用方言行为差异的事实）

**Status:** resolved

- [x] troubleshooting 错误目录按「现象 → 原因 → 处置」组织，覆盖上述类别且与实际错误类型/信息一致
- [x] upgrade-notes 说清三处行为变化的旧/新对照与受影响写法的改法
- [x] CHANGELOG.md 采用 Keep a Changelog 结构（类型分组：Added/Changed/Fixed），首条为 v0.1.0（2026-10-16）
- [x] v0.1.0 条目内容与实际提交历史相符（Added：四方言/注册表/CLI 方言；Changed：严格契约等）
- [x] 三件之间交叉链接有效，措辞与 dialects.md 无矛盾

## Comments

**验收证据（D06，提交 `8fcecbe`（claim）+ `3ad5fa6`（三件落地，--ff-only 并入 main，分支已删）+ `9e5e9eb`（CHANGELOG 归并微调））：**

**① docs/troubleshooting.md**：六节结构 = 1 张 Mermaid 报错路由图 + **16 个「现象 → 原因 → 处置」条目**（§1 UnsupportedOperationError 读法 1 条：消息两组成部分（方言名/操作名或类型名）+ errors.As + 三步处置 + dry-run 预扫；§2 迁移锁 4 条：sqlite `ADVISORY LOCK`、`migration lock timeout`、锁未释放/会话级锁与连接池、postgres 无超时疑点（仅按代码事实：`pg_advisory_lock(hashtext($1))` 无超时参数、CLI 用 `context.Background()`）；§3 历史表 4 条：自动建表失败、自定义表名漂移、`migration not found:`、Name 重复撞主键；§4 dry-run 差异 4 条：仍连库（SELECT 照常）、语句多于真实执行（探针短路）、全量重放、dry 通过≠真通过（含 MySQL DDL 隐式提交）；§5 连接驱动 3 条：`sql: unknown driver "..." (forgotten import?)`（blank import 表 + sqlite 驱动名是 `sqlite` 不是 `sqlite3`）、`DB_DSN is required`/`unsupported DB_DIALECT`、DSN 语法/multiStatements）+ **§6 已知限制表 9 行**（postgres 非串行主键静默丢 PK、Comment 修饰符丢弃、AlterColumn 只改类型、sqlite 主键折叠、sqlite/sqlserver 探针边界对 *IfExists 的后果、mysql/sqlserver 裸 VARBINARY、enum/set 落 TEXT、mysql DropPrimaryKey 忽略 name——与 dialects.md §2 逐条一致）。错误消息原文逐字核对自源码：`types.go:102`（`%s does not support %s`）、`migration.go:12/15/18`、`sqlite.go:274`、`main.go:111/117`、`generator.go:170`、`base_dialect.go` DML 校验五条、`database/sql`（`forgotten import?`）。

**② docs/upgrade-notes.md**：三处变化各含旧/新对照表 + before/after 改法。事实核查要点（按 4473841 与 d52a0c3^/1cbfab6^ 双点核对）：(1) drop 旧态按方言如实分行——postgres `DropTable`/`DropIndex` 与 sqlserver `DropTable` 用 `IF EXISTS`（静默跳过；sqlserver 依赖 2016+ 语法），mysql/sqlite 基线本就严格无变化；改法示例 `DropTable` → `DropTableIfExists(ctx, ...)`。(2) 未映射类型旧态为 `default: strings.ToUpper(typeName)` 透传（含 postgres AlterColumn 路径）；公开构造器全集四方言均有映射，故如实注明「现有迁移无需改写」，原生类型改法给 `Raw`。(3) sqlserver enum/set 经 git 核对为「空串哨兵（mssqlType 返回 ""）+ BuildColumn `typeSQL == ""` 兜底转 UnsupportedOperationError」——对外错误自 2026-06 首版即存在（a901360 单测断言在案），故按 ADR-0002「空串机制废弃」表述机制变化并明示错误消息前后一致，未照搬工单「静默空串」速记（与源码事实不符处已按核查结果修正）。另附 d52a0c3 的 sqlserver 字面量转义修复说明与升级检查清单。

**③ CHANGELOG.md**：Keep a Changelog 结构，首条 `[v0.1.0] - 2026-10-16`；Added 10 条（奠基工作按主题归并；postgres/sqlserver 方言=1b56776/c46829e/869d92c/a901360/8b1d69a/d0909a0、别名=2893db2、Statements()=4473841、历史表 seam=ac71cee、一致性矩阵=535ff8f/5f1098d、baseDialect=56d7390/7bb5254、注册表与 create --dialect=ba59eea、文档=5b4aaf3/3d45829/2df10e3/e6a24bc/2d47166/ca53584/31041c3/9aa92f2）、Changed 3 条（严格 drop=d52a0c3/ADR-0001、类型报错与空串机制废弃=1cbfab6/ADR-0002、existenceGuard 折叠=866a58e 无行为变化）、Fixed 1 条（sqlserver 字面量 `'` 转义=d52a0c3）；条目内嵌提交号，产品性提交全部对号（claim/resolve 类 chore 不入列）。

**交叉链接与一致性**：三件互链 + 五个既有 docs 页「下一步」/导语补链（无孤儿页）；CHANGELOG → upgrade-notes#strict-drop，upgrade-notes → ADR-0001/0002/CHANGELOG/troubleshooting，troubleshooting ↔ dialects.md/cli.md/migrations.md/getting-started.md/architecture.md。24 个锚点逐一验证存在；Mermaid 路由图以 mermaid-cli 12.0.0 实际渲染通过（SVG exit 0）。与 dialects.md 无矛盾：锁（mysql/sqlserver 超时生效、postgres 无超时、sqlite 报 ADVISORY LOCK）、严格契约、探针边界、裸 VARBINARY 等表述逐条对照。

**验证**：未触碰任何 Go 文件（`git status` 仅新增三份 .md 与五个文档页补链）；`go clean -testcache && go test ./...` 全绿（cmd/migrate、internal/generator、migrate 三包 ok）。
