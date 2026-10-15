# 01: 过程文档迁入 dev-docs，docs/ 让位使用文档

**What to build:** 仓库文档分区归位：docs/ 下现存的 compose/superpowers 过程文档（plans/specs 共 7 份）与根目录 IMPLEMENTATION.md 全部迁入 dev-docs/（纯移动，内容不改），所有引用链接修正；迁移后 docs/ 为空置的使用文档区，仓库内无死链。

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] docs/ 下 compose/、superpowers/ 过程文档经 git mv 迁入 dev-docs/（保持可读的子目录结构，如 plans/specs 分组）
- [x] 根目录 IMPLEMENTATION.md 迁入 dev-docs/
- [x] 仓库内所有指向被移动文件的链接（README、其他文档、工单）修正为新路径
- [x] 全仓库搜索无指向旧路径的死链（含相对链接拼写）
- [x] 纯移动零内容改动（git log 可佐证 rename 检测），`go test ./...` 照常全绿（未触碰代码）

## Comments

**2026-10-15 验收证据（落地提交 e6a24bc，claim 提交 2ed95dc，main --ff-only 合并）：**

迁移文件清单（8 份，git 全部识别为 rename 100%，0 insertions / 0 deletions）：

- `IMPLEMENTATION.md` -> `dev-docs/IMPLEMENTATION.md`
- `docs/compose/plans/2026-06-13-postgres-sqlserver-dialects.md` -> `dev-docs/compose/plans/...`
- `docs/compose/specs/2026-06-13-postgres-sqlserver-dialects-design.md` -> `dev-docs/compose/specs/...`
- `docs/superpowers/plans/2026-06-13-multidb-yii2-migration.md` -> `dev-docs/superpowers/plans/...`
- `docs/superpowers/plans/2026-06-13-sqlite-support.md` -> `dev-docs/superpowers/plans/...`
- `docs/superpowers/specs/2026-06-13-multidb-yii2-migration-design.md` -> `dev-docs/superpowers/specs/...`
- `docs/superpowers/specs/2026-06-13-sqlite-support-design.md` -> `dev-docs/superpowers/specs/...`
- `docs/superpowers/specs/2026-06-14-iphubtest-migrations-design.md` -> `dev-docs/superpowers/specs/...`

子目录结构按来源分组（dev-docs/compose/、dev-docs/superpowers/，各含 plans/specs），IMPLEMENTATION.md 置于 dev-docs/ 顶层。

修正的链接数：0。全仓库 grep（含 md/json/yml 与全部 git 跟踪文件）证明：README、AGENTS.md、dev-docs/、其余工单均无指向旧路径的链接；8 份被移动文件内部亦无 markdown 链接或 `docs/` 路径字符串（仅存在对 `IMPLEMENTATION.md` 的纯文字提及，按零内容改动约束未触碰）。唯一提及旧路径的是本工单自身的任务描述（规格原文，未改动）。

测试输出摘要：`go test ./...` 全绿 —— `ok github.com/phgors/yiimigrate/cmd/migrate`、`ok .../internal/generator`、`ok .../migrate`（examples/migrations、migrations 无测试文件）。未触碰任何 Go 代码。

迁移后 docs/ 为空目录（使用文档区已让位）。
