# 06: 收敛 SchemaPlan 的十个同形 If-guard

**What to build:** SchemaPlan 中十个 `*IfExists` 变体重复的 guard 逻辑（dryRun 直发、存在性探针、已存在跳过、出错记入 p.err）收敛为一个内部 helper，每个变体缩为一行调用。公共链式 API 完全冻结：方法签名、链式行为、dry-run 语义零变化。

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] guard 逻辑单点实现（一个内部 helper 承载 dryRun 判断 + 探针 + 跳过 + 记错）
- [ ] 十个 If-variant 复用该 helper，重复块消失
- [ ] 公共方法集合、签名与链式行为不变（现有调用方式全部照常编译运行）
- [ ] helper 本身有直接单元测试（dryRun / 已存在 / 探针出错三条路径）
- [ ] `go test ./...` 全绿
