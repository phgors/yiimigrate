# 02: 一致性矩阵——表驱动 conformance 套件覆盖全部 adapter

**What to build:** 新增共享的一致性矩阵测试：每类 Dialect 操作按「op × 输入 → 期望 SQL」的表驱动用例遍历全部 adapter 断言。MySQL 与 SQLite 的 DDL 生成自动获得与 postgres/sqlserver 同级的覆盖；两个新方言测试文件中与矩阵重复的期望被删除，只保留真实分歧（unsupported 操作、存在性探针、锁、方言怪癖）。

**Blocked by:** 01（新 interface 方法需进入矩阵）

**Status:** resolved

- [x] 存在共享矩阵测试文件，新增一个用例即可同时覆盖四个 adapter（含 01 引入的历史表方法）
- [x] MySQL 的 DDL 生成覆盖度不再只有一个单测，与 postgres/sqlserver 对齐
- [x] postgres/sqlserver 测试文件仅保留分歧场景，与矩阵重复的期望已删除
- [x] 删除的期望总量可感知（两文件明显缩短），矩阵用例数 ≥ 各文件原有覆盖面
- [x] `go test ./...` 全绿

## Comments

- 验收证据（2026-10-15）：
  - 新增 `migrate/dialect_conformance_test.go`（789 行）：104 个表驱动用例 × 4 adapter = 416 个方言级断言，全部通过。分组：命名/引号/占位符 10、BuildColumn 类型与修饰符 53、Schema DDL 30（含 01 引入的 `CreateMigrationTableSQL`）、DML 语句 12（query + args）。runner 对缺少任一方言期望的用例直接报错（"case has no expectation for dialect %s"），保证「新增一个 case 同时覆盖四方言」。
  - 删除重复期望：postgres_test.go 710 → 100 行（仅保留 information_schema/pg_indexes 存在性探针 + pg_advisory 锁）；sqlserver_test.go 725 → 105 行（仅保留 sys.* 探针 + sp_getapplock 锁 + `]` 转义、非自增主键两个怪癖）；dialect_test.go 125 → 58 行（SQLite sqlite_master/pragma 探针 + 具名约束哨兵探针，并补齐 Index/FK/Constraint 探针断言）。删减提交净 -1,294 行（+43/-1,337）。
  - 覆盖面对比：原三文件共 ~102 个测试函数，矩阵 104 个用例且每用例同时断言四方言（原 postgres/sqlserver 各只断言单方言）。
  - `go test ./...` 全绿（`ok github.com/phgors/yiimigrate/migrate` 等全部 ok；集成测试按约定仅在 *_TEST_DSN 存在时运行）；`go vet ./...` 通过；改动文件已 gofmt。
  - 落地提交：535ff8f（新增矩阵）、5f1098d（删除重复期望），已 --ff-only 合并至 main 并删除 feat/02-conformance-matrix 分支。
