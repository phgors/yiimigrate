# 05: 架构页——Mermaid 模块图与执行时序图

**What to build:** docs/architecture.md——用 Mermaid 图向库使用者讲清结构：模块图（Migrator、SchemaPlan、Dialect seam、baseDialect、四个 adapter、方言注册表的组合关系）、一次 migrate up 的时序图（锁 → 历史表 → 版本比对 → 执行 → 记录）、CLI 到库的调用流程图。辅以少量文字说明 seam 背后的设计意图（严格契约、幂等归探针）。

**Blocked by:** 01（docs/ 分区归位）

**Status:** ready-for-agent

- [ ] 模块图以 Mermaid flowchart 呈现，模块/接口/seam 命名与 CONTEXT.md 词汇表一致
- [ ] migrate up 时序图覆盖锁、迁移历史表、版本比对、执行、记录各阶段，与 migrator 实际流程一致
- [ ] CLI 流程图与 cmd/migrate 实际调用路径一致（含方言注册表取用）
- [ ] 全部图用 Mermaid 代码块（不用 ASCII 字符画），GitHub 可直接渲染
- [ ] 文字精炼，设计意图表述与 ADR-0001/0002 无冲突
