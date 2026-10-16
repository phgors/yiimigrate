# CLI 参考（cmd/migrate）

`cmd/migrate` 是本仓库自带的迁移命令行工具，八个命令，配置全部通过环境变量传递（除 `create` 的两个旗标外没有旗标）。本文每一项均与 `cmd/migrate` 的实现逐一核对，适用于 v0.1.0。

第一次使用建议先读 [快速上手](getting-started.md)；本页是逐项参考。

## 用法

在仓库根目录下：

```bash
go run ./cmd/migrate <up|down|redo|history|new|mark|to|create> [参数]
```

不带任何参数运行时输出一行用法提示并以 1 退出：

```text
usage: migrate <up|down|redo|history|new|mark|to|create>
```

## 命令一览

<a id="commands"></a>

| 命令 | 形式 | 作用 | 参数默认值 |
| --- | --- | --- | --- |
| `up` | `up [n]` | 应用待执行迁移（旧 -> 新） | `n` 省略或 `<= 0` 时应用全部 |
| `down` | `down [n]` | 回滚已应用迁移（新 -> 旧） | `n` 默认 `1` |
| `redo` | `redo [n]` | 回滚再重放最近 `n` 个迁移 | `n` 默认 `1` |
| `history` | `history [n]` | 列出已应用迁移（新 -> 旧） | `n` 默认 `10` |
| `new` | `new [n]` | 列出待执行迁移（旧 -> 新） | `n` 默认 `10` |
| `mark` | `mark VERSION` | 只改写历史记录，不执行迁移代码 | `VERSION` 必填 |
| `to` | `to VERSION` | 上移或回滚到指定版本 | `VERSION` 必填 |
| `create` | `create [--fields=...] [--dialect=NAME] NAME` | 生成迁移模板文件 | 见 [create](#create) |

`up` / `down` / `redo` / `history` / `new` 是数据库命令：需要 `DB_DSN`（必填）与 `DB_DIALECT`（默认 `mysql`），在执行前都会自动确保迁移历史表存在。数字参数若不是合法整数，按省略处理（回落到该命令的默认值）。

## 数据库命令

### up [n]

计算待执行迁移（已注册但不在历史表中的），按版本名从旧到新逐个应用。每个迁移在自己的事务里执行：先跑 `Up`，再把版本号与 Unix 时间戳写入历史表；中途失败即停止，已完成的迁移保持不变。

```bash
export DB_DIALECT=mysql
export DB_DSN="root:password@tcp(127.0.0.1:3306)/app?parseTime=true"
go run ./cmd/migrate up        # 应用全部
go run ./cmd/migrate up 1      # 只应用最旧的一个待执行迁移
```

输出（每个迁移两行进度，写入 stdout）：

```text
>>> applying m20260613_120000_create_article_table
<<< applied m20260613_120000_create_article_table
```

整个命令执行期间持有迁移锁（见[迁移锁](#lock)），防止并发迁移互相践踏。

### down [n]

从最新开始回滚 `n` 个已应用迁移，每个迁移在自己的事务里执行 `Down` 并从历史表删除对应记录。若历史表里存在未注册的版本（迁移文件被删除），报 `migration not found: <version>` 并停止。

```bash
go run ./cmd/migrate down      # 回滚最近 1 个
go run ./cmd/migrate down 3    # 回滚最近 3 个
```

```text
>>> reverting m20260613_120000_create_article_table
<<< reverted m20260613_120000_create_article_table
```

### redo [n]

等价于先 `down n` 再把刚回滚的迁移重新 `up`（回滚几个就重放几个）。

```bash
go run ./cmd/migrate redo
```

### history [n]

列出最近应用的迁移，每行 `版本号 应用时间戳`（Unix 秒），最新的在前。没有记录时不输出任何内容。

```bash
go run ./cmd/migrate history
```

```text
m20260613_120000_create_article_table 1789012345
```

### new [n]

列出待执行迁移的版本号，最旧的在前，每行一个。没有待执行迁移时不输出任何内容。

```bash
go run ./cmd/migrate new
```

```text
m20260613_120000_create_article_table
```

### mark VERSION

只改写历史记录、不执行任何迁移代码，用于把一个已有 schema 的库纳入迁移管理：清空当前历史记录，然后把从最旧到 `VERSION`（含）的所有已注册迁移写入历史表。`VERSION` 为 `0` 时只清空历史表。`VERSION`（非 `0` 时）必须在注册表中，否则报 `migration not found`。

```bash
go run ./cmd/migrate mark m20260613_120000_create_article_table
go run ./cmd/migrate mark 0   # 清空历史表
```

### to VERSION

迁移到指定版本：`VERSION` 为 `0` 时回滚全部；目标版本未应用则把待执行迁移依次应用到目标为止；目标已应用则把比它新的迁移依次回滚到目标为止。`VERSION`（非 `0` 时）未注册时报 `migration not found`。

```bash
go run ./cmd/migrate to m20260613_120000_create_article_table
go run ./cmd/migrate to 0
```

## create：生成迁移模板

<a id="create"></a>

`create` 不连接数据库，只生成本地文件：

```bash
go run ./cmd/migrate create [--fields=...] [--dialect=NAME] NAME
```

```text
created migrations/m20260613_120000_create_article_table.go
```

生成规则：

- 文件写入当前目录下的 `migrations/`，文件名为 `m` + 当前时刻（`YYYYMMDD_HHMMSS`）+ `_` + 小写蛇形 NAME，包名 `migrations`；已存在的同名文件不会被覆盖（报错退出）。
- 生成后还需把文件里的类型注册进 `migrations/register.go`（见[快速上手 第 4 步](getting-started.md#register)）。

**旗标必须写在 `NAME` 之前**。Go 的 flag 解析在第一个非旗标参数处停止，`go run ./cmd/migrate create create_article_table --fields=...` 不会识别 `--fields`，而是报 usage 错误。正确写法：

```bash
go run ./cmd/migrate create --fields="title:string(128):notNull" create_article_table
```

### --fields 字段语法

<a id="fields"></a>

```text
name:type[:modifier[:modifier...]]
```

多个字段用逗号分隔；类型参数写进圆括号，如 `string(128)`、`decimal(10,2)`。可用 modifier：

| modifier | 生成代码 |
| --- | --- |
| `notNull` | `.NotNull()` |
| `null` | `.Null()` |
| `unsigned` | `.Unsigned()` |
| `default(10)` | `.DefaultValue(10)`（数字与 `true`/`false` 原样，其余按字符串） |
| `defaultExpression(CURRENT_TIMESTAMP)` | `.DefaultExpression("CURRENT_TIMESTAMP")` |

字段未写 `null` / `notNull` 时，生成器自动追加 `.Null()`。

```bash
go run ./cmd/migrate create \
  --fields="user_id:unsignedBigInteger:notNull,title:string(128):notNull,content:longText,metadata:json,status:unsignedTinyInteger:notNull:default(10)" \
  create_article_table
```

### --dialect 按方言生成

`--dialect` 决定模板里的主键写法与表选项，只接受方言注册表的四个规范名：`mysql`、`postgres`、`sqlite`、`sqlserver`（别名 `mssql`、`postgresql`、`sqlite3` 不可用，会报 `unknown dialect`）。默认值取 `DB_DIALECT`，未设置时为 `mysql`——注意若 `DB_DIALECT` 设了别名（如 `postgresql`），`create` 未显式给 `--dialect` 时同样会报 `unknown dialect`，此时请显式传规范名。

| 方言 | 隐式 `id` 主键 | 表选项 |
| --- | --- | --- |
| `mysql` | `m.UnsignedBigPrimaryKey()` | `"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"` |
| `sqlite` | `m.PrimaryKey()` | `""` |
| `postgres` | `m.BigPrimaryKey()` | `""` |
| `sqlserver` | `m.BigPrimaryKey()` | `""` |

```bash
go run ./cmd/migrate create --dialect=postgres --fields="title:string(128):notNull,content:longText" create_article_table
```

带 `--fields` 时，`Up` 生成 `CreateTable`（表名取迁移名去掉 `create_` 前缀与 `_table` 后缀），`Down` 生成 `DropTable`；不带 `--fields` 时，`Up` 是一段占位 `Raw("-- TODO: add migration SQL")`，`Down` 返回 `migrate.ErrIrreversibleMigration`（不可逆）。

## 环境变量

<a id="env"></a>

| 变量 | 必填 | 默认 | 说明 |
| --- | --- | --- | --- |
| `DB_DSN` | 数据库命令必填 | 无 | 连接串，格式由方言驱动定义，示例见下表；`create` 不需要 |
| `DB_DIALECT` | 否 | `mysql` | 数据库方言；不区分大小写、忽略首尾空白；别名见下 |
| `MIGRATE_TABLE` | 否 | `migration` | 迁移历史表表名 |
| `MIGRATE_DRY_RUN` | 否 | 关闭 | 设为 `1`、`true`、`TRUE` 或 `yes` 时开启 dry-run（精确匹配这些值） |

`DB_DIALECT` 的可用值：`mysql`、`postgres`（别名 `postgresql`）、`sqlite`（别名 `sqlite3`）、`sqlserver`（别名 `mssql`）。方言对象始终来自 `migrate` 包的方言注册表，别名只在这层映射生效。

各方言 DSN 示例（由驱动定义，更详细的说明见[快速上手 - 选择方言与 DSN](getting-started.md#dialect)）：

| 方言 | 驱动 | DSN 示例 |
| --- | --- | --- |
| `mysql` | github.com/go-sql-driver/mysql | `root:password@tcp(127.0.0.1:3306)/app?parseTime=true` |
| `postgres` | github.com/lib/pq | `postgres://user:password@127.0.0.1:5432/app?sslmode=disable` |
| `sqlite` | modernc.org/sqlite | `file:app.db`（内存库用 `:memory:`） |
| `sqlserver` | github.com/microsoft/go-mssqldb | `sqlserver://user:password@127.0.0.1:1433?database=app` |

## 迁移锁

<a id="lock"></a>

写操作命令（`up`、`down`、`redo`、`mark`、`to`）默认启用迁移锁（锁名 `yiimigrate`，等待 30 秒），整个命令期间持锁，实现按各方言的锁能力：

| 方言 | 锁实现 |
| --- | --- |
| `mysql` | `SELECT GET_LOCK(?, ?)` / `SELECT RELEASE_LOCK(?)` |
| `postgres` | `SELECT pg_advisory_lock(hashtext($1))` / `SELECT pg_advisory_unlock(hashtext($1))` |
| `sqlserver` | `sp_getapplock` / `sp_releaseapplock` |
| `sqlite` | 不支持，直接报 `sqlite does not support ADVISORY LOCK` |

锁在超时时间内拿不到时报 `migration lock timeout`。dry-run 模式不取锁。内嵌使用时可通过 `Migrator.UseLock = false` 关闭（见 [快速上手 - 集成到自己的项目](getting-started.md#embed)）。

## dry-run

`MIGRATE_DRY_RUN=1` 后，写操作命令变成校验模式：

- 迁移代码照常执行到语句生成这一步，生成期错误（如方言不支持的 `UnsupportedOperationError`）会照常暴露；
- 不执行任何迁移 SQL、不写历史表、不获取迁移锁；
- 不查询当前历史（所有已注册迁移都按待执行处理）；
- 控制台仍输出 `>>> applying` / `<<< applied`（或 `reverting` / `reverted`）进度行。

```bash
MIGRATE_DRY_RUN=1 go run ./cmd/migrate up
```

```text
>>> applying m20260613_120000_create_article_table
<<< applied m20260613_120000_create_article_table
```

## 退出与报错行为

<a id="exit"></a>

- 成功：退出码 `0`。`history` / `new` 无结果时不输出任何内容；`up` / `down` / `redo` 只输出进度行，无额外摘要。
- 失败：错误信息打印一行到 stderr，退出码 `1`。数据库命令先检查 `DB_DSN`，再校验 `mark` / `to` 的参数个数。

常见错误与触发条件（消息原文）：

| 错误消息 | 触发条件 |
| --- | --- |
| `DB_DSN is required` | 数据库命令未设置 `DB_DSN` |
| `unsupported DB_DIALECT "x"; supported values: mysql, postgres, sqlite, sqlserver` | `DB_DIALECT` 不在别名表（含拼写错误） |
| `unknown dialect "x"; supported values: mysql, postgres, sqlite, sqlserver` | `create --dialect` 不是注册表的规范名 |
| `unknown command "x"` | 第一个参数不是八个子命令之一 |
| `usage: migrate mark VERSION` | `mark` 未给 `VERSION`（且 `DB_DSN` 已设置） |
| `usage: migrate to VERSION` | `to` 未给 `VERSION`（且 `DB_DSN` 已设置） |
| `usage: migrate create NAME [--fields=...] [--dialect=...]` | `create` 的 NAME 数量不对（含旗标写在 NAME 之后的情况） |
| `migration not found: <version>` | `down` 遇到未注册的已应用版本；`to` / `mark` 的版本未注册 |
| `sqlite does not support ADVISORY LOCK` | SQLite 上执行写操作命令且未开 dry-run |
| `migration lock timeout` | 迁移锁在等待时间内未获得 |

## up 的执行流程

```mermaid
flowchart TD
    A["读取 DB_DIALECT / DB_DSN"] --> B["打开数据库连接"]
    B --> C["获取迁移锁<br/>（dry-run 跳过）"]
    C --> D["确保历史表存在<br/>（不存在则自动创建）"]
    D --> E["计算待执行迁移<br/>（已注册 - 已应用）"]
    E --> F["逐个迁移：事务内执行 Up<br/>并写入 version + apply_time"]
    F --> G["释放迁移锁"]
```
