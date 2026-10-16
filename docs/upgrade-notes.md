# 升级注意：v0.1.0 的行为变化

本页面向从 **v0.1.0 之前的开发快照**（2026-06 状态，即提交 `4473841` 一线）升级到 v0.1.0 的使用者，逐一说清三处行为变化的旧 / 新对照与受影响写法的改法。旧行为均按当时的源码与 [ADR-0001](../dev-docs/adr/0001-strict-drops-plan-level-idempotency.md) / [ADR-0002](../dev-docs/adr/0002-unmapped-column-type-errors.md) 核对；变更落点见 [CHANGELOG](../CHANGELOG.md)。CLI 用法与 API 本身在 v0.1.0 无破坏性变化，以下三条是全部的行为差异。

## 1. drop 严格化：基线删除不再带 `IF EXISTS`

<a id="strict-drop"></a>

**结论**：不带 `IfExists` 的基线删除操作（`DropTable`、`DropIndex` 等）在四个方言上**一律生成严格 DROP**——对象不存在时由数据库报自然错误，不再静默跳过。幂等删除改用 plan 层的 `*IfExists` 变体。

**旧 / 新对照**（按 2026-06 状态的各方言实现逐个核对）：

| 基线操作 | 旧行为（2026-06） | 新行为（v0.1.0） |
| --- | --- | --- |
| postgres `DropTable` | `DROP TABLE IF EXISTS ...`——缺表**静默跳过** | `DROP TABLE ...`——缺表由数据库报错（如 `42P01`） |
| postgres `DropIndex` | `DROP INDEX IF EXISTS ...`——缺索引**静默跳过** | `DROP INDEX ...`——缺索引由数据库报错 |
| sqlserver `DropTable` | `DROP TABLE IF EXISTS ...`——SQL Server 2016+ 上缺表静默跳过；更早版本直接语法报错 | `DROP TABLE ...`——缺表由数据库报错（如错误 3701） |
| mysql / sqlite `DropTable` | 已是严格 `DROP TABLE`（无变化） | 同左 |
| 各方言 `DropIndex` | mysql / sqlite / sqlserver 已是严格 DROP（无变化，仅 postgres 见上） | 同左 |

为什么改：同一 Dialect 方法在四个 adapter 下错误模式分裂（有的静默跳过、有的报错），且与 plan 层 `*IfExists` 探针形成双重幂等。决定幂等只归 plan 层的存在性探针，adapter 层删除一律严格——四个方言同一种失败模式（ADR-0001）。

**受影响写法与改法**：`Down`（或清理类迁移）里依赖「对象不存在也不报错」的基线 drop——典型是反复执行的 `DropTable` / `DropIndex`。改为 `*IfExists` 变体（`ctx` 是第一个参数，供探针使用）：

```go
// 旧写法：postgres/sqlserver 上缺对象时静默跳过，v0.1.0 起会报数据库错误
func (M20260613_120000CreateArticleTable) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		DropTable("article").
		Exec(ctx)
}

// v0.1.0 写法：幂等删除交给存在性探针，缺对象时干净跳过
func (M20260613_120000CreateArticleTable) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		DropTableIfExists(ctx, "article").
		Exec(ctx)
}
```

两点说明：

- 探针语义：非 dry-run 时先查对象存在性，存在才发语句、不存在直接跳过；dry-run 下语句直发不查（完整判定流程见[迁移编写指南 - `*IfExists`](migrations.md#ifexists)）。
- 报错的样子：改法没跟上时，升级后的第一次失败会是数据库自然错误（MySQL `ERROR 1051`、PostgreSQL `42P01`、SQL Server `3701`、SQLite `no such table`），错误文本来自数据库而非本库。

## 2. 未映射列类型：透传拼 SQL 改为统一报错

<a id="unmapped-types"></a>

**结论**：ColumnBuilder 的类型名映射到各方言类型表，映射未命中时**不再透传**，四个方言统一返回 `UnsupportedOperationError`（消息 `"<方言> does not support <类型名>"`，如 `mysql does not support geometry`）。

**旧 / 新对照**：

| | 旧行为（2026-06） | 新行为（v0.1.0） |
| --- | --- | --- |
| 四个 adapter 的类型映射 `default` 分支 | 未识别的类型名**大写后原样拼进列定义**（静默产出垃圾 SQL，到数据库才炸，或更糟——恰好拼出一条合法但错误的语句） | `return unsupported(方言, 类型名)`，生成期立即报错 |
| postgres `AlterColumn` | 同样直接拼映射结果，未映射类型透传进 `ALTER COLUMN ... TYPE` | 映射错误向上传播，`AlterColumn` 报错 |

为什么改：透传违背本库「explicit unsupported」政策——静默产出垃圾语句比报错更危险。四种失败模式归一后，dry-run 扫一遍即可提前发现（ADR-0002）。

**受影响写法与改法**：对现有迁移**没有需要动手的改写**——`MigrationContext` 上的类型构造器全集（`m.String`、`m.Json`……）在四个方言都有映射（postgres / sqlite 的 `enum` / `set` 落为 `TEXT`）或本就显式报错（sqlserver），`default` 分支此前对外不可达。受影响的是**依赖透传预期的代码**：同包内直接构造 ColumnBuilder（如测试、实验代码）传入未映射类型名时，旧版会拿到大写透传的 SQL，v0.1.0 起拿到错误——这类写法改用 `.Append(sql)` 拼原生片段，或 `Raw` 执行整条原生 DDL：

```go
// 需要库未覆盖的原生类型（如 geo 系列）时，不用指望透传，对整条语句负责：
Raw("ALTER TABLE geo ADD COLUMN point POINT NOT NULL")
```

## 3. sqlserver 的 `enum` / `set`：空串哨兵机制废弃

<a id="enum-set"></a>

**结论**：SQL Server 没有 enum / set 等价类型，`m.Enum(...)` / `m.Set(...)` 在 sqlserver 上报 `sqlserver does not support enum`（`set` 同理）——这一点 v0.1.0 前后一致；变化在机制内部。

**旧 / 新对照**（按源码逐行核对）：

| | 旧行为（2026-06） | 新行为（v0.1.0） |
| --- | --- | --- |
| 类型映射 | `mssqlType` 对 `enum` / `set` **返回空串**，再由 `BuildColumn` 的 `if typeSQL == ""` 兜底转成 `UnsupportedOperationError`（自 2026-06 首版即如此，当时的单测已断言该错误类型） | 类型映射直接返回 `unsupported(...)`，空串哨兵机制删除 |
| 对外错误 | `sqlserver does not support enum` / `set` | 同左——错误类型与消息**不变** |

为什么改：旧机制是「空串哨兵 + 兜底翻译」，与其余未映射类型的大写透传混在同一张类型表里，两种失败模式并存；统一后四方言只有一种失败模式（ADR-0002「空串机制废弃」）。

**受影响写法与改法**：无需改写（错误行为前后一致）。在 sqlserver 上建模取值约束的惯用做法不变：`NVARCHAR` + `.Check(...)` 约束，或参考表（见 [SQL Server 注意事项](dialects.md#sqlserver)）。

## 相关修复与检查清单

同一轮变更还包含一处缺陷修复，无需动作、自动受益：sqlserver 把对象名嵌进字符串字面量的语句（`sp_rename`、注释的扩展属性、历史表建表的 `OBJECT_ID` 探针）现在会把名字里的 `'` 翻倍转义——此前对象名含 `'` 会拼出断裂 / 可注入的语句（提交 `d52a0c3`，见 [CHANGELOG](../CHANGELOG.md)）。

升级后建议过一遍：

1. 全部迁移 `MIGRATE_DRY_RUN=1 up` 扫一遍——三处变化相关的生成期问题（严格 drop 之外的两条、类型报错）会在这一步暴露；
2. 检索基线 `DropTable` / `DropIndex` 调用点，缺对象容忍场景改为 `*IfExists` 变体；
3. 真实环境跑一次 `down` / 重放，确认没有依赖旧的静默跳过行为。

更多运行期错误的排查见[排错指南](troubleshooting.md)。
