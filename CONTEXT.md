# yiimigration

受 Yii2 Migration 启发的 Go 迁移库：用 `database/sql` 在多数据库上执行带版本的 schema 迁移。

## Language

### 核心概念

**Migration**:
一次带版本号的 schema 变更单元，具备 Up/Down 两个方向。
_Avoid_: 脚本、script、变更集

**迁移历史表 (Migration table)**:
记录已应用迁移版本的库表（version + apply_time）。由 Migrator 拥有，其表结构是 Migrator 的领域概念，各数据库只负责翻译建表语法。
_Avoid_: history table、迁移日志

**Dry-run**:
只生成并展示语句、不执行的执行模式。计划中的语句照常积累，但永不触达数据库。
_Avoid_: 模拟执行

### Dialect 层

**Dialect**:
库对「一种数据库的 SQL 方言」的抽象 interface——所有 schema 操作与 DML 的生成都要穿过它。
_Avoid_: driver（指 database/sql 驱动）、provider、数据库适配层

**Adapter**:
Dialect interface 在某个具体数据库（mysql / sqlite / postgres / sqlserver）上的实现。
_Avoid_: driver、方言实现（口语可用，文档统一用 adapter）

**baseDialect**:
内嵌在四个 adapter 中的共享语句实现底座；adapter 只覆写真实分歧（类型映射、引号、探针、锁、注释、不支持项）。
_Avoid_: 基类（Go 无继承，是内嵌组合）

**方言注册表 (Dialect registry)**:
`migrate` 包内 name → Dialect 的枚举注册表，CLI 与生成器共用，是方言知识的唯一入口。
_Avoid_: 方言工厂、driver 映射表

### 计划与测试

**SchemaPlan**:
链式累积 schema 语句的计划；dry-run 下只积累不执行。`*IfExists` 变体负责幂等。
_Avoid_: builder（列构造器才是 builder）、语句列表

**IfExists 探针 (existence probe)**:
plan 层的幂等机制：执行前先探测对象是否存在，存在则跳过。幂等只属于这里，adapter 层的 drop 一律严格（见 ADR-0001）。
_Avoid_: IF EXISTS 语法（那是某数据库的 SQL 特性，不是本库概念）

**一致性矩阵 (Conformance matrix)**:
表驱动测试套件：op × 输入 → 期望 SQL，遍历全部 adapter；方言专属测试文件只保留真实分歧。
_Avoid_: 方言测试套件（指整份 per-dialect 文件时可以说，但矩阵专指共享表）
