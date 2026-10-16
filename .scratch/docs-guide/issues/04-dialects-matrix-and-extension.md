# 04: 方言支持矩阵、注意事项与新增方言指南

**What to build:** docs/dialects.md 三段式：能力矩阵表（操作 × 四方言：支持 / unsupported 报错 / 怪癖，从一致性矩阵与各 adapter 的 UnsupportedOperationError 清单提炼为用户视角）；四方言注意事项各一节（含 drop 严格契约、postgres Comment 修饰符静默丢弃与非串行主键怪癖、sqlserver 的 ] 转义/非自增主键/enum 与 set 不支持等）；「新增一个方言」扩展指南（baseDialect 内嵌 + 真实 delta 清单 + 一致性矩阵期望列 + 方言注册表登记）。

**Blocked by:** 03（引用迁移编写页的 SchemaPlan/ColumnBuilder 概念）

**Status:** claimed

- [ ] 能力矩阵以表格呈现且与当前四个 adapter 实际行为逐格核对（以一致性矩阵用例为事实源）
- [ ] unsupported 项全部如实标注（对应 UnsupportedOperationError 可触发的操作）
- [ ] 四方言注意事项覆盖已知怪癖与严格契约语义（ADR-0001/0002 的用户视角表述）
- [ ] 新增方言指南按当前扩展机制写成步骤化流程（delta 清单、矩阵期望、注册表登记、测试要求）
- [ ] 矩阵与注意事项无互相矛盾；页内术语与 CONTEXT.md 一致
