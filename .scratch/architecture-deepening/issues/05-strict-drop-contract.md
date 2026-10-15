# 05: drop 严格契约 + 标识符字面量转义（ADR-0001）

**What to build:** 四个 adapter 的 DropTable/DropIndex 变严格：不再自带 IF EXISTS，缺对象由各数据库自然报错；幂等只由 plan 层的 IfExists 探针负责（其行为不变）。SQL Server 中把对象名嵌入字符串字面量的语句（sp_rename、扩展属性）对含 `'` 的名字安全。

**Blocked by:** 02（矩阵先固化各方言 drop 期望）

**Status:** claimed

- [ ] mysql/sqlite/postgres 的 drop 语句不再带 IF EXISTS 后缀，四 adapter 错误模式统一为「缺对象自然报错」
- [ ] DropIndex 对 table 参数的使用方式在矩阵中按方言显式记录且与实现一致
- [ ] plan 层 `*IfExists` 变体行为不变，drop 类 IfExists 探针路径有测试覆盖
- [ ] sqlserver 的 sp_rename / 扩展属性语句对含 `'` 的对象名生成合法 SQL（有单元测试）
- [ ] `go test ./...` 全绿
