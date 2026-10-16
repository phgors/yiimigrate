# 06: 排错指南、升级注意与 CHANGELOG

**What to build:** 三件配套：docs/troubleshooting.md（常见错误目录：UnsupportedOperationError 的读法与处置、迁移锁未释放、历史表问题、dry-run 与真实执行差异、各方言连接错误示例）、docs/upgrade-notes.md（v0.1.0 三处行为变化的说明与迁移建议：drop 严格化、未映射类型报错、sqlserver enum/set 报错）、根目录 CHANGELOG.md（Keep a Changelog 风格，首条 v0.1.0，此前工作按主题归并简述）。

**Blocked by:** 04（错误目录需引用方言行为差异的事实）

**Status:** claimed

- [ ] troubleshooting 错误目录按「现象 → 原因 → 处置」组织，覆盖上述类别且与实际错误类型/信息一致
- [ ] upgrade-notes 说清三处行为变化的旧/新对照与受影响写法的改法
- [ ] CHANGELOG.md 采用 Keep a Changelog 结构（类型分组：Added/Changed/Fixed），首条为 v0.1.0（2026-10-16）
- [ ] v0.1.0 条目内容与实际提交历史相符（Added：四方言/注册表/CLI 方言；Changed：严格契约等）
- [ ] 三件之间交叉链接有效，措辞与 dialects.md 无矛盾
