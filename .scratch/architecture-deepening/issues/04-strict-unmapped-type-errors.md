# 04: 未映射列类型一律报 unsupported 错误（ADR-0002）

**What to build:** 任何 adapter 遇到类型映射未命中的列类型都返回明确的 unsupported 错误（含类型名与方言名），四个 adapter 同一失败模式；「大写透传成 SQL」的 default 分支删除；sqlserver 对 enum/set 的空串静默机制同步废弃为 unsupported 报错。已映射类型的生成结果零变化。

**Blocked by:** 02（矩阵先枚举现有映射缺口，避免翻转默认值时踩隐性依赖）

**Status:** resolved

- [x] 四个 adapter 对未映射类型返回同型错误，错误信息可定位（类型名 + 方言名）
- [x] 不再存在把类型名直接拼进 SQL 的透传路径
- [x] sqlserver enum/set 从空串改为 unsupported 错误，相关调用方与测试更新
- [x] 一致性矩阵佐证：现有全部已映射类型的期望 SQL 不变
- [x] `go test ./...` 全绿

## Comments

- 落地提交：`1cbfab6`（feat 分支 `--ff-only` 合入 main，分支已删）。
- 改动：四个 adapter 的类型映射函数（`mysqlType` / `sqliteType` / `pgType` / `mssqlType`）签名改为 `(string, error)`，`default` 分支返回 `UnsupportedOperationError{Dialect, Operation: typeName}`（错误信息如 `mysql does not support geometry`）；`strings.ToUpper(typeName)` 透传全部删除。调用方 `BuildColumn`（×4）与 postgres `AlterColumn` 同步传播错误。
- sqlserver：`mssqlType` 删除 `case "enum", "set": return ""` 空串分支，enum/set 落入 default 报 unsupported；`BuildColumn` 的 `typeSQL == ""` 空串哨兵机制随之删除（identity 查找的空串语义不受影响，那是三态查找非失败路径）。
- 矩阵：104 既有行期望零改动（全部通过，已映射类型输出逐字节不变）；新增 2 行——`unmapped column type`（四方言均 wantUnsupported("geometry")）与 `alter column unmapped type`（mysql/postgres/sqlserver 报类型错误，sqlite 先报 ALTER COLUMN）——现共 106 行 × 4 方言 = 424 叶子用例全过。sqlserver enum/set 行在 02 已预置 wantUnsupported，本次无需改期望。
- 测试：`go test ./...` 全绿（migrate / cmd/migrate / internal/generator）；集成测试本机未设 `*_TEST_DSN` 自然跳过。gofmt 已跑改动文件。
- 备注：公开 API 无法构造未映射类型名（`newColumnBuilder` 未导出），矩阵用例经包内构造 `newColumnBuilder("geometry")` 覆盖；未来若增加 raw-type 构造器，strict 报错路径已就位。
