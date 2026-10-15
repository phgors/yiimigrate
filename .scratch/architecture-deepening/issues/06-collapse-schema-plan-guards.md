# 06: 收敛 SchemaPlan 的十个同形 If-guard

**What to build:** SchemaPlan 中十个 `*IfExists` 变体重复的 guard 逻辑（dryRun 直发、存在性探针、已存在跳过、出错记入 p.err）收敛为一个内部 helper，每个变体缩为一行调用。公共链式 API 完全冻结：方法签名、链式行为、dry-run 语义零变化。

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] guard 逻辑单点实现（一个内部 helper 承载 dryRun 判断 + 探针 + 跳过 + 记错）
- [x] 十个 If-variant 复用该 helper，重复块消失
- [x] 公共方法集合、签名与链式行为不变（现有调用方式全部照常编译运行）
- [x] helper 本身有直接单元测试（dryRun / 已存在 / 探针出错三条路径）
- [x] `go test ./...` 全绿

## Comments

- 2026-10-15 落地提交 `866a58e`（main，fast-forward 自 `feat/06-collapse-plan-guards`）。
- 收敛：新增未导出 `(*SchemaPlan).existenceGuard(ctx, emitOnExists, probe, emit)`——dryRun 直发 emit；否则跑探针，出错记入 `p.err`，`exists != emitOnExists` 跳过；drop 类传 `true`（存在才发）、create/add 类传 `false`（缺失才发）。十个 `*IfExists` 变体各缩为一次 helper 调用，原有 13 行同形块全部删除。
- 净缩行数：`schema_plan.go` 110 删 / 53 增，净 -57 行（含新增 helper 本体约 20 行）；重复块消失。
- 公共 API 零变化：方法集合、签名、doc 注释、链式返回同一 `*SchemaPlan`、dry-run 语义逐分支等价（含「探针错误覆盖先前 err」「plan 已有 err 时仍会探针」这两处原有可观察行为，均按原样保留）。
- 新增测试：`TestSchemaPlanExistenceGuard`（6 子用例：dry-run 不探针直发 / drop 极性存在即发 / drop 极性缺失即跳 / create 极性缺失即发 / create 极性存在即跳 / 探针出错记入 plan.err 且不发）。工单 05 的 `TestSchemaPlanDropIfExistsProbesBeforeStrictDrop`、`TestSchemaPlanDropIfExistsDryRunEmitsStrictDrop` 及其余现有测试未改一行、全部通过。
- 验证：改动文件 `gofmt` 干净（经 LF 规范化后无 diff）；`go vet ./...` 干净；`go clean -testcache` 后 `go test ./...` 全绿。集成测试因本机未设 `*_TEST_DSN` 自然跳过。
