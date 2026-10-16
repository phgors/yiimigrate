# 07: README 瘦身为入口索引

**What to build:** README.md 重构为入口页：简介、特性一览、安装、最小 quickstart、指向 docs/ 各页与 CHANGELOG 的索引。首段过时宣称（「已实现并测试 MySQL 和 SQLite」）修正为四方言现状；现 README 中与 docs/ 页重复的详细章节（ColumnBuilder/SchemaPlan/CLI/注意事项等）迁入对应页面后删除，Yii2 对照表移驻合适页面；全仓库链接体检。

**Blocked by:** 02, 03, 04, 05, 06（全部内容页就位后方可索引与搬迁）

**Status:** resolved

- [x] 首段宣称与 v0.1.0 现状一致（四方言已实现并有单元+集成测试覆盖，集成测试按 DSN 触发）
- [x] README 仅保留：简介、特性、安装、quickstart、文档索引、CHANGELOG 链接；篇幅显著下降
- [x] 原 README 详细内容全部有 docs/ 去处（含 Yii2 对照表），无内容凭空丢失
- [x] README ↔ docs/ 双向链接与页间链接全部有效（无死链、无孤儿页）
- [x] `go test ./...` 全绿（未触碰代码）

## Comments

**验收证据（2026-10-15，提交 a18cf0f）**

- README 前后尺寸：558 行 / 17359 字节 -> 72 行 / 3904 字节（约原 22%，优于「≤ 三分之一」目标）。仅保留：简介（首段改写为四方言现状：mysql / postgres / sqlite / sqlserver 均有一致性矩阵单元测试，mysql / postgres / sqlserver 另有 `MYSQL_TEST_DSN` / `POSTGRES_TEST_DSN` / `SQLSERVER_TEST_DSN` 触发的集成测试）、特性一览、安装、最小 quickstart（旗标顺序已按 v0.1.0 实际行为修正为「旗标在 NAME 之前」）、docs/ 七页 + CHANGELOG 索引。仓库无 LICENSE 文件，按拍板决策不设许可证段。
- 内容去向：Yii2 对照表（13 行映射 + 新增两条差异提醒）-> docs/migrations.md 末尾新章节「Yii2 到 Go API 对照」（锚点 `yii2-map`，migrations.md 原无等价内容，无需去重）；「测试」段 -> docs/getting-started.md 新章节「跑测试」（更新为三个 DSN 现状）；ColumnBuilder / SchemaPlan / 查询 helper / 对象存在判断 / 锁与 dry-run / 错误 -> migrations.md 既有章节已覆盖；CLI / 生成器 -> cli.md；MySQL / SQLite 注意事项 -> dialects.md §2；完整迁移示例与注册 -> getting-started.md。README 旧 CLI 段的已知过时错误（方言清单不全、create 旗标顺序错）随瘦身一并消除。
- 链接体检：自写审计脚本遍历全部 36 个 tracked .md 文件的相对链接（文件存在性 + 锚点存在性，支持 `<a id>` 与 GitHub heading slug 两种锚点形态），0 死链；脚本有效性经注入坏链接反向验证（DEAD FILE / DEAD ANCHOR 均被检出后还原）。README 索引覆盖 docs/ 全部七页 + CHANGELOG，无孤儿页；docs/ 内无指向 README 锚点的链接，无悬空引用。
- `go test ./...`：cmd/migrate、internal/generator、migrate 三包全绿（纯文档改动，未触碰 Go 代码）。
- 落地：feat/docs07-readme -> main `--ff-only` 合并（a18cf0f），分支已删。
