# 05: drop 严格契约 + 标识符字面量转义（ADR-0001）

**What to build:** 四个 adapter 的 DropTable/DropIndex 变严格：不再自带 IF EXISTS，缺对象由各数据库自然报错；幂等只由 plan 层的 IfExists 探针负责（其行为不变）。SQL Server 中把对象名嵌入字符串字面量的语句（sp_rename、扩展属性）对含 `'` 的名字安全。

**Blocked by:** 02（矩阵先固化各方言 drop 期望）

**Status:** resolved

- [x] mysql/sqlite/postgres 的 drop 语句不再带 IF EXISTS 后缀，四 adapter 错误模式统一为「缺对象自然报错」
- [x] DropIndex 对 table 参数的使用方式在矩阵中按方言显式记录且与实现一致
- [x] plan 层 `*IfExists` 变体行为不变，drop 类 IfExists 探针路径有测试覆盖
- [x] sqlserver 的 sp_rename / 扩展属性语句对含 `'` 的对象名生成合法 SQL（有单元测试）
- [x] `go test ./...` 全绿

## Comments

- 2026-10-15 落地提交 `d52a0c3`（main，fast-forward 自 `feat/05-strict-drop-contract`）。
- 严格化：`baseDialect.buildDropTable` 删除 ifExists 分支只输出严格 `DROP TABLE`；postgres `DropTable`/`DropIndex` 与 sqlserver `DropTable` 不再拼 IF EXISTS（mysql/sqlite 原本已严格）。矩阵改动仅两行：「drop table」postgres/sqlserver 期望去掉 IF EXISTS，「drop index」postgres 期望去掉 IF EXISTS；其余期望未动。
- DropIndex table 参数按方言记录在矩阵行注释：mysql/sqlserver 语法需要 `ON table`（实现走 `buildDropIndex`），postgres/sqlite 按名删除、忽略 table；期望值与实现一一对应。
- 转义：sqlserver 的 `RenameTable`/`RenameColumn`/`AddCommentOn*`/`DropCommentFrom*` 以及 `CreateMigrationTableSQL` 的 OBJECT_ID 字面量统一经 `escapeSQLString`（`'` -> `''`，与 `sqlLiteral` 共用同一规则）。新增 `TestSQLServerIdentifierLiteralsEscapeSingleQuotes`（7 个用例：rename table/column、4 条扩展属性、migration table）验证含 `'` 名字生成合法 SQL。
- plan 层未改一行；新增 `TestSchemaPlanDropIfExistsProbesBeforeStrictDrop`（5 个 drop 类 IfExists 变体：对象存在 -> 执行严格 drop 语句；缺失 -> 探针拦截、零执行）与 `TestSchemaPlanDropIfExistsDryRunEmitsStrictDrop`（dry-run 不探针、不执行但仍产出严格 drop 语句）。
- 验证：`go vet ./...` 干净；`go test ./...` 全绿（conformance 530 个子用例通过）；改动文件经 LF 规范化后 `gofmt -l` 无输出（仓库工作树为 CRLF，属既有现象）。集成测试因本机未设 `*_TEST_DSN` 自然跳过。

