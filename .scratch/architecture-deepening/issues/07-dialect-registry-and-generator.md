# 07: 方言注册表 + 生成器按方言出模板

**What to build:** `migrate` 包内建方言注册表：按名字取 Dialect、枚举全部方言名；CLI 的 driver↔dialect 映射改从注册表获取。`migrate create` 在四种方言下都生成能直接 `migrate up` 的迁移模板——postgres/sqlserver 不再用 UnsignedBigPrimaryKey 与 MySQL 表选项，按方言出正确的 PK 形态与表选项（或不出）。

**Blocked by:** 03（adapter 构造形状稳定后再接线）

**Status:** resolved

- [x] migrate 包暴露按名取 Dialect 与列出全部方言名的能力
- [x] CLI 不再自持 driver↔dialect 映射，从注册表取
- [x] 四种方言下生成的迁移模板单测验证：PK 形态正确、表选项按方言取舍、生成的 Up 语句在对应方言的矩阵期望内合法
- [x] generator 不再出现硬编码的 MySQL 专属模板片段
- [x] `go test ./...` 全绿

## Comments

- 2026-10-15 落地提交 `ba59eea`（main，fast-forward 自 `feat/07-dialect-registry-generator`）。
- 注册表：新增 `migrate/dialect_registry.go`——不可变 name→Dialect 表（mysql/sqlite/postgres/sqlserver，与原 CLI 选择一致），`DialectByName(name)`（大小写/空白不敏感，ok=false 报未注册）与 `DialectNames()`（排序枚举）。新增测试：`TestDialectNamesReturnsSortedRegistryNames`、`TestDialectByNameResolvesEveryRegisteredName`、`TestDialectByNameNormalizesInput`、`TestDialectByNameRejectsUnknownName`、`TestDialectRegistryCoversConformanceDialects`（注册表与一致性矩阵不漂移）。
- CLI：`resolveDBConfig` 不再构造任何 Dialect——`driverAliases` 只做拼写→database/sql driver 名（含 sqlite3/postgresql/mssql 别名与空默认 mysql），方言一律 `migrate.DialectByName` 获取；报错列表改为 `migrate.DialectNames()` 拼接（排序后 mysql, postgres, sqlite, sqlserver，`TestResolveDBConfigRejectsUnsupportedDialect` 相应改为逐名断言）。新增 `TestResolveDBConfigSupportsEveryRegisteredDialect`：注册表每增长一个方言，CLI 接线即被测试强制覆盖。
- 生成器：`Options.Dialect`（默认 mysql；`normalizeOptions` 供 `Generate`/`BuildContent` 共用），经注册表校验，未知方言报 `unknown dialect "..."` + 支持列表。模板分歧收敛到 `dialectProfiles`（按 `dialect.Name()` 取，未列出的注册方言回退可移植形态 `m.PrimaryKey()` + 无表选项）：mysql `m.UnsignedBigPrimaryKey()` + `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`；sqlite `m.PrimaryKey()`；postgres/sqlserver `m.BigPrimaryKey()`（BIGSERIAL / BIGINT IDENTITY）；共享模板不再含任何 MySQL 专属字面量。`migrate create` 新增 `--dialect` 旗标（默认取 `DB_DIALECT`，未设为 mysql，`TestDefaultCreateDialectFollowsDBDialectEnv`）。
- 模板单测（`internal/generator/generator_test.go`）：`TestBuildContentTemplateShapePerDialect`（PK 行 + 表选项取舍：mysql 带 ENGINE、其余断言不含 `ENGINE=`）、`TestGeneratedUpSQLMatchesDialectExpectations`（从生成文本提取 id 主键 builder 调用与表选项字面量，喂回对应方言 `CreateTable`，断言整条 SQL 与一致性矩阵期望逐字一致，如 sqlserver `CREATE TABLE [article] ([id] BIGINT IDENTITY(1,1) PRIMARY KEY NOT NULL, ...)`）、`TestBuildContentDefaultsToMySQLDialect`、`TestBuildContentRejectsUnknownDialect`。
- 额外验证：四方言模板经 `Generate` 实际落盘后 `go build ./...` 编译通过（临时目录验证后已清理）；`go clean -testcache` 后 `go test ./...` 全绿（626 passed, 5 packages）；`go vet ./...` 干净；改动文件 `gofmt` 无 diff（`migrate/` 下未改动文件的 gofmt 报告为既有 CRLF 行尾问题，未触碰）。README 增补 `--dialect` 用法与按方言模板表。集成测试因本机未设 `*_TEST_DSN` 自然跳过。
