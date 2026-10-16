# 架构：模块关系与执行时序

本页面向想理解库结构的使用者，用三张图讲清三件事：模块之间怎么组合（seam 在哪）、一次 `migrate up` 在库内怎么执行（锁 -> 迁移历史表 -> 版本比对 -> 执行与记录）、CLI 怎么走进库。三张图的节点与连线均与 v0.1.0 源码逐一核对；API 用法不在本页展开（见[迁移编写指南](migrations.md)），只讲结构与其背后的设计意图。术语与仓库根目录 `CONTEXT.md` 词汇表一致。

## 模块图：Dialect seam 与组合关系

```mermaid
flowchart TB
    MIG["Migrator<br/>版本调度 · 迁移锁 · 迁移历史表"]
    PLAN["SchemaPlan<br/>链式累积 schema 语句"]
    SEAM[["Dialect interface（seam）<br/>所有 SQL 生成穿过它"]]
    BASE["baseDialect<br/>共享语句实现底座"]
    MYSQL["MySQLDialect"]
    SQLITE["SQLiteDialect"]
    PG["PostgreSQLDialect"]
    MSSQL["SQLServerDialect"]
    REG["方言注册表（migrate 包）<br/>DialectByName / DialectNames"]

    MIG -->|"依赖"| SEAM
    PLAN -->|"依赖"| SEAM
    MYSQL -->|"实现"| SEAM
    SQLITE -->|"实现"| SEAM
    PG -->|"实现"| SEAM
    MSSQL -->|"实现"| SEAM
    MYSQL -->|"内嵌"| BASE
    SQLITE -->|"内嵌"| BASE
    PG -->|"内嵌"| BASE
    MSSQL -->|"内嵌"| BASE
    REG --> MYSQL
    REG --> SQLITE
    REG --> PG
    REG --> MSSQL
```

- **Dialect 是唯一的 seam**。Migrator 与 SchemaPlan 都不自己拼 SQL：语句生成都穿过 `Dialect` interface，执行则始终经 `database/sql` 完成——adapter 只产出语句文本，不碰连接，本库不引入 ORM。Migrator 执行每个迁移时构造 `MigrationContext`（携带同一个 Dialect 与当前事务），迁移代码经它取得 SchemaPlan，因此两个使用方落到的是同一个 seam。
- **迁移历史表是 Migrator 的领域概念**。表结构（version + apply_time）由 Migrator 决定，各数据库只负责翻译建表语法：mysql / sqlite / postgres 生成 `CREATE TABLE IF NOT EXISTS`，sqlserver 没有 `IF NOT EXISTS`，改为 `IF OBJECT_ID(...) IS NULL` 判存后建表。对调用方始终是同一句「确保历史表存在」。
- **内嵌组合，不是基类**。baseDialect 内嵌在四个 adapter 中，承载四方言一致的语句形状；adapter 只覆写真实分歧：类型映射、引号、存在性探针、迁移锁、注释、不支持项。因此新增一个方言 = 实现一个 adapter + 补一致性矩阵期望列 + 注册表登记，步骤见[方言参考 - 新增一个方言](dialects.md)。
- **方言注册表是方言知识的唯一入口**。`migrate` 包内 `name -> Dialect` 的不可变注册表，经 `DialectByName` / `DialectNames` 暴露；CLI 与迁移生成器都从这里取 Dialect（见下方流程图），不存在第二份方言名单。

## 时序图：一次 migrate up

与 `migrate/migrator.go` 的实际执行顺序一致。注意锁包裹整个执行段：建迁移历史表、读已应用版本、逐个执行迁移，全部在锁内完成。

```mermaid
sequenceDiagram
    autonumber
    participant C as 调用方（CLI 或用户代码）
    participant M as Migrator
    participant D as Dialect（adapter）
    participant DB as database/sql

    C->>M: Up(ctx, limit)

    alt UseLock 且非 dry-run
        M->>D: AcquireLockSQL(LockName, LockTimeoutSeconds)
        D-->>M: 取锁语句
        M->>DB: 执行取锁语句
        Note over M: 抢锁失败（返回值 != 1）则中止：<br/>ErrMigrationLockTimeout
    else dry-run 或关闭锁
        Note over M: 跳过锁，直接进入执行
    end

    M->>M: Pending()：待应用 = 已注册 - 已应用

    alt dry-run
        Note over M: 不触达数据库：不建迁移历史表、<br/>已应用版本视为空（全部迁移视为待应用）
    else 正常执行
        M->>D: CreateMigrationTableSQL(迁移历史表)
        D-->>M: 建表语句（幂等，存在则跳过）
        M->>DB: 执行（确保迁移历史表存在）
        M->>DB: SELECT version, apply_time（读已应用版本）
    end

    loop 每个待应用迁移（旧 -> 新）
        alt dry-run
            M->>M: migration.Up（dry-run 的 MigrationContext）
            Note over M: 语句只打印、不执行，<br/>不写迁移历史表
        else 正常执行
            M->>DB: BEGIN
            M->>M: migration.Up（事务内的 MigrationContext）
            M->>DB: INSERT version + apply_time（记录到迁移历史表）
            M->>DB: COMMIT（任一步失败则 ROLLBACK）
        end
    end

    opt 之前已取锁
        M->>D: ReleaseLockSQL(LockName)
        D-->>M: 释放锁语句
        M->>DB: 执行释放锁语句
    end

    M-->>C: 本次应用的版本列表
```

- **锁语句由 Dialect 生成**：MySQL 用 `GET_LOCK` / `RELEASE_LOCK`，SQL Server 用 `sp_getapplock`，Migrator 只判断抢锁结果；dry-run 与显式关闭锁时整段跳过。
- **一个迁移一个事务**：`Up` 的语句与写入迁移历史表的 INSERT 在同一事务内提交，任一步失败整体回滚，历史表不会记录半次迁移（语义见[迁移编写指南 - 执行模型](migrations.md)）。
- **dry-run 分支**：不取锁、不建迁移历史表、已应用版本视为空（因此全部已注册迁移都会被重放展示），语句只打印、永不触达数据库。

## 流程图：CLI 到库的调用路径

与 `cmd/migrate/main.go` 一致。驱动注册与方言解析是两条独立的线：database/sql 驱动经 blank import 在进程启动时注册（main.go 顶部的四个驱动 import）；方言本体则始终经方言注册表取得。

```mermaid
flowchart TD
    DRV["进程启动：四个 database/sql 驱动<br/>经 blank import 注册"]
    A(["run(args)"]) --> C{"第一个参数<br/>子命令分发"}
    C -->|"create"| CR["runCreate"]
    C -->|"up / down / redo /<br/>history / new / mark / to"| DBCMD["runDBCommand"]
    C -->|"其他"| ERR["unknown command 报错"]

    DBCMD --> ENV["读环境变量<br/>DB_DSN（必填）· DB_DIALECT（默认 mysql）"]
    ENV --> ALIAS["拼写别名归一<br/>sqlite3 -> sqlite · mssql -> sqlserver"]
    ALIAS --> REG["方言注册表取 Dialect<br/>migrate.DialectByName"]
    REG -->|"未注册"| UNS["unsupported DB_DIALECT 报错<br/>（列出 DialectNames 全名单）"]
    REG -->|"ok"| OPEN["sql.Open(驱动名, DSN)<br/>驱动来自启动时注册"]
    DRV -.-> OPEN
    OPEN --> NEW["NewMigrator(db, dialect, migrations.All())<br/>应用 MIGRATE_TABLE / MIGRATE_DRY_RUN"]
    NEW --> CMD["migrator.Up / Down / Redo /<br/>History / New / Mark / To"]

    CR --> CRFLAG["解析 --fields / --dialect 旗标<br/>（dialect 默认取 DB_DIALECT，否则 mysql）"]
    CRFLAG --> GEN["generator.Generate<br/>内部同样经 DialectByName 取 Dialect"]
    GEN --> OUT["写出迁移模板文件"]
```

- CLI 内的别名表只做拼写归一，把 `DB_DIALECT` 的各种写法映射到规范驱动名；方言本体永远来自 `DialectByName`，未注册的取值报错并列出 `DialectNames()` 的全名单。
- `create` 分支不连数据库：生成器内部同样经 `DialectByName` 取 Dialect，用其类型映射渲染模板——注册表被 CLI 与生成器共用的另一处体现。
- 八个子命令的旗标、环境变量与报错行为逐项见[CLI 参考](cli.md)。

## 三条贯穿的设计决定

1. **drop 一律严格**：adapter 层的删除类操作永不生成 `IF EXISTS`，对象不存在交给数据库自然报错——四个方言同一失败模式，不静默跳过。
2. **幂等只归 plan 层探针**：`*IfExists` 变体在发语句前先探测对象存在性，存在才跳过；幂等是 plan 层的概念，不依赖任何数据库的 SQL 特性。
3. **显式报 unsupported**：数据库给不了的操作返回 `UnsupportedOperationError`，未映射的列类型同样报错、不透传拼 SQL。

前两条的逐操作语义见[方言参考 - 图例与两条全局契约](dialects.md#legend)。

## 下一步

- [快速上手](getting-started.md)：从零跑通第一次迁移的完整闭环。
- [迁移编写指南](migrations.md)：ColumnBuilder、SchemaPlan 与查询 helper 的 API 主参考。
- [方言参考](dialects.md)：能力矩阵、各方言注意事项与新增方言指南。
- [CLI 参考](cli.md)：全部子命令、旗标、环境变量与报错行为。
