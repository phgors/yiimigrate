# 02: 一致性矩阵——表驱动 conformance 套件覆盖全部 adapter

**What to build:** 新增共享的一致性矩阵测试：每类 Dialect 操作按「op × 输入 → 期望 SQL」的表驱动用例遍历全部 adapter 断言。MySQL 与 SQLite 的 DDL 生成自动获得与 postgres/sqlserver 同级的覆盖；两个新方言测试文件中与矩阵重复的期望被删除，只保留真实分歧（unsupported 操作、存在性探针、锁、方言怪癖）。

**Blocked by:** 01（新 interface 方法需进入矩阵）

**Status:** ready-for-agent

- [ ] 存在共享矩阵测试文件，新增一个用例即可同时覆盖四个 adapter（含 01 引入的历史表方法）
- [ ] MySQL 的 DDL 生成覆盖度不再只有一个单测，与 postgres/sqlserver 对齐
- [ ] postgres/sqlserver 测试文件仅保留分歧场景，与矩阵重复的期望已删除
- [ ] 删除的期望总量可感知（两文件明显缩短），矩阵用例数 ≥ 各文件原有覆盖面
- [ ] `go test ./...` 全绿
