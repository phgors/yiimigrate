# Adapter 层 drop 一律严格，幂等只归 plan 层探针

四种数据库里 mysql/sqlite/postgres 原生支持 `DROP ... IF EXISTS`，但 sqlserver 不支持，导致同一 Dialect 方法在四个 adapter 下错误模式分裂（有的静默跳过、有的报错），且与 SchemaPlan 的 `*IfExists` 探针形成双重幂等。决定：adapter 层的 drop 一律严格（缺对象由该数据库自然报错），幂等只由 plan 层的存在性探针负责；探针的额外往返在迁移场景可忽略。附带约束：所有把标识符嵌入字符串字面量的语句（如 sqlserver 的 `sp_rename`、扩展属性）必须转义。

## Considered Options

- 让支持的方言原生 `IF EXISTS`、sqlserver 用探针兜底：省一次往返，但错误模式继续分裂，被否决。
