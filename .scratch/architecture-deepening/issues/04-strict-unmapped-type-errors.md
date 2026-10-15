# 04: 未映射列类型一律报 unsupported 错误（ADR-0002）

**What to build:** 任何 adapter 遇到类型映射未命中的列类型都返回明确的 unsupported 错误（含类型名与方言名），四个 adapter 同一失败模式；「大写透传成 SQL」的 default 分支删除；sqlserver 对 enum/set 的空串静默机制同步废弃为 unsupported 报错。已映射类型的生成结果零变化。

**Blocked by:** 02（矩阵先枚举现有映射缺口，避免翻转默认值时踩隐性依赖）

**Status:** ready-for-agent

- [ ] 四个 adapter 对未映射类型返回同型错误，错误信息可定位（类型名 + 方言名）
- [ ] 不再存在把类型名直接拼进 SQL 的透传路径
- [ ] sqlserver enum/set 从空串改为 unsupported 错误，相关调用方与测试更新
- [ ] 一致性矩阵佐证：现有全部已映射类型的期望 SQL 不变
- [ ] `go test ./...` 全绿
