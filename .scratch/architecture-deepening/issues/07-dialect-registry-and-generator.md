# 07: 方言注册表 + 生成器按方言出模板

**What to build:** `migrate` 包内建方言注册表：按名字取 Dialect、枚举全部方言名；CLI 的 driver↔dialect 映射改从注册表获取。`migrate create` 在四种方言下都生成能直接 `migrate up` 的迁移模板——postgres/sqlserver 不再用 UnsignedBigPrimaryKey 与 MySQL 表选项，按方言出正确的 PK 形态与表选项（或不出）。

**Blocked by:** 03（adapter 构造形状稳定后再接线）

**Status:** claimed

- [ ] migrate 包暴露按名取 Dialect 与列出全部方言名的能力
- [ ] CLI 不再自持 driver↔dialect 映射，从注册表取
- [ ] 四种方言下生成的迁移模板单测验证：PK 形态正确、表选项按方言取舍、生成的 Up 语句在对应方言的矩阵期望内合法
- [ ] generator 不再出现硬编码的 MySQL 专属模板片段
- [ ] `go test ./...` 全绿
