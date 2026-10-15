# 07: README 瘦身为入口索引

**What to build:** README.md 重构为入口页：简介、特性一览、安装、最小 quickstart、指向 docs/ 各页与 CHANGELOG 的索引。首段过时宣称（「已实现并测试 MySQL 和 SQLite」）修正为四方言现状；现 README 中与 docs/ 页重复的详细章节（ColumnBuilder/SchemaPlan/CLI/注意事项等）迁入对应页面后删除，Yii2 对照表移驻合适页面；全仓库链接体检。

**Blocked by:** 02, 03, 04, 05, 06（全部内容页就位后方可索引与搬迁）

**Status:** ready-for-agent

- [ ] 首段宣称与 v0.1.0 现状一致（四方言已实现并有单元+集成测试覆盖，集成测试按 DSN 触发）
- [ ] README 仅保留：简介、特性、安装、quickstart、文档索引、CHANGELOG 链接；篇幅显著下降
- [ ] 原 README 详细内容全部有 docs/ 去处（含 Yii2 对照表），无内容凭空丢失
- [ ] README ↔ docs/ 双向链接与页间链接全部有效（无死链、无孤儿页）
- [ ] `go test ./...` 全绿（未触碰代码）
