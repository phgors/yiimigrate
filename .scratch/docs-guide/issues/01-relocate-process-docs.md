# 01: 过程文档迁入 dev-docs，docs/ 让位使用文档

**What to build:** 仓库文档分区归位：docs/ 下现存的 compose/superpowers 过程文档（plans/specs 共 7 份）与根目录 IMPLEMENTATION.md 全部迁入 dev-docs/（纯移动，内容不改），所有引用链接修正；迁移后 docs/ 为空置的使用文档区，仓库内无死链。

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] docs/ 下 compose/、superpowers/ 过程文档经 git mv 迁入 dev-docs/（保持可读的子目录结构，如 plans/specs 分组）
- [ ] 根目录 IMPLEMENTATION.md 迁入 dev-docs/
- [ ] 仓库内所有指向被移动文件的链接（README、其他文档、工单）修正为新路径
- [ ] 全仓库搜索无指向旧路径的死链（含相对链接拼写）
- [ ] 纯移动零内容改动（git log 可佐证 rename 检测），`go test ./...` 照常全绿（未触碰代码）
