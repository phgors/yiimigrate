# 02: 入门指南与 CLI 参考（从零到跑通迁移）

**What to build:** 新用户从安装库到在四种方言任一下跑通首次迁移的完整路径：docs/getting-started.md（安装、DSN 配置、方言选择、历史表自动创建、首次 up/down）与 docs/cli.md（全命令与旗标参考：create/up/down、--dialect、--dry-run、环境变量 DB_DIALECT 等、退出行为）。两页互相链接，代码示例与 v0.1.0 实际行为一致。

**Blocked by:** 01（docs/ 分区归位后再落使用文档）

**Status:** claimed

- [ ] getting-started.md 覆盖：安装、四方言 DSN 示例、方言选择（含 DialectByName 注册表口径）、首次迁移完整可复制流程
- [ ] cli.md 覆盖全部子命令与旗标（含 07 轮新增的 create --dialect 与按方言模板行为），逐项与 cmd/migrate 实际实现核对
- [ ] 所有代码示例可直接复制使用，命令输出描述与真实行为一致
- [ ] 中文行文；与 README 现有内容的分工清晰（此处详、README 简）
- [ ] 两页间及页内锚点链接有效
