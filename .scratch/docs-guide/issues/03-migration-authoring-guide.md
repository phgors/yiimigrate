# 03: 迁移编写指南（API 使用主干）

**What to build:** docs/migrations.md——库集成者的核心参考：迁移注册与命名、Up/Down 结构、ColumnBuilder 全部列类型与修饰符、SchemaPlan 链式操作（含 *IfExists 幂等语义与存在性探针）、查询 helper（RowExists/Insert/Expr 等）、迁移锁与 dry-run、不可逆迁移。术语使用 CONTEXT.md 词汇表（迁移历史表、IfExists 探针等）。

**Blocked by:** 02（入门页概念先行，本页引用之）

**Status:** claimed

- [ ] 迁移注册与 Name 约定、Up/Down 结构、不可逆迁移（ErrIrreversibleMigration）成节
- [ ] ColumnBuilder 列类型与修饰符清单与 column_builder 实际 API 一一对应（不可变性说明到位）
- [ ] SchemaPlan 各操作 + *IfExists 的探针幂等语义（dry-run 直发、已存在跳过、探针出错行为）讲解清楚
- [ ] 查询 helper、锁、dry-run 各成节，示例可复制
- [ ] API 细节链接 pkg.go.dev，不在页内重复 godoc
- [ ] 术语与 CONTEXT.md 一致；页内示例经人工推敲与实际 API 签名吻合
