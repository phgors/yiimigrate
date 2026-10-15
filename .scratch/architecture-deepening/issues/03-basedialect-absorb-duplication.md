# 03: baseDialect 内嵌吸收四个 adapter 的共享语句实现

**What to build:** 四个 adapter 内嵌共享的 baseDialect，共享语句形状（CreateIndex、AddForeignKey、RowExists/CountRows、Insert/Update/Delete 委托、CreateTable 组装骨架等）只实现一次并可覆写；dialect.go 中已有的 buildInsert/buildUpdate/buildDelete 家族并入其中。每个 adapter 文件只剩真实 delta：类型映射、引号、探针、锁、注释、不支持项。行为零变化。

**Blocked by:** 02（矩阵作重构安全网）

**Status:** claimed

- [ ] 共享语句形状在 baseDialect 单点实现，adapter 通过内嵌获得、按需覆写
- [ ] 两两逐字重复的方法体消失（RowExists、CreateIndex、AddForeignKey、DML 委托等不再 ×4）
- [ ] dialect.go 的共享构造函数家族迁入 baseDialect，原调用方迁移完毕
- [ ] 一致性矩阵与全部现有测试全绿，生成 SQL 逐字节不变
- [ ] adapter 四个文件总行数显著下降（预期净缩数百行）
