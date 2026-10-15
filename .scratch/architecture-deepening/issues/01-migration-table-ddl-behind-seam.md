# 01: 历史表 DDL 收回 Dialect seam（修 SQL Server 活 bug）

**What to build:** 在四种支持的数据库上，`Migrator.Up/Down/Applied` 首次运行都能自动创建迁移历史表：历史表 DDL 由各 adapter 通过 Dialect interface 的新方法生成，Migrator 不再自带任何内联 DDL。SQL Server 用户从「第一条迁移就语法错误」变为可正常跑通迁移。

**Blocked by:** None (can start immediately)

**Status:** claimed

- [ ] Dialect interface 新增迁移历史表建表方法（列结构 version varchar(180) PK + apply_time int 由 Migrator 侧定义，adapter 只翻译语法）
- [ ] 四个 adapter 各自实现：mysql/sqlite/postgres 用原生 IF NOT EXISTS，sqlserver 用 IF OBJECT_ID(...) IS NULL 形式
- [ ] Migrator 建历史表的调用全部穿过该方法，源码中不再有内联的 CREATE TABLE 语句
- [ ] SQL Server 的历史表建表 SQL 有不依赖真实数据库的单元测试断言
- [ ] `go test ./...` 全绿
