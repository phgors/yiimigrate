# 快速上手：从零跑通第一次迁移

本页带一个新项目从安装走到完整的迁移闭环：生成迁移文件 -> 注册 -> `up` 应用（自动创建迁移历史表）-> `history` 核对 -> `down` 回滚。命令与输出均与 v0.1.0 的实际行为一致。

- 只想查某个命令或旗标：直接看 [CLI 参考](cli.md)。
- 想了解字段类型与 Schema 操作的完整清单：见 [迁移编写指南](migrations.md) 与 [pkg.go.dev 文档](https://pkg.go.dev/github.com/phgors/yiimigrate/migrate)。

## 前置条件

- Go 1.26 及以上（本库 `go.mod` 声明 `go 1.26.2`）。
- 一个可连接的数据库，四种方言任选其一：MySQL、PostgreSQL、SQL Server、SQLite。
- 环境变量写法：下文示例使用 bash（`export VAR=value`）；PowerShell 用 `$env:VAR = "value"`，cmd 用 `set VAR=value`，效果相同。

## 总览

<a id="overview"></a>

跑通一次迁移的完整路径：

```mermaid
flowchart LR
    A["create 生成迁移文件"] --> B["注册到 migrations.All()"]
    B --> C["设置 DB_DIALECT / DB_DSN"]
    C --> D["new 查看待执行"]
    D --> E["up 应用<br/>（自动创建迁移历史表）"]
    E --> F["history 核对"]
    F --> G["down 回滚"]
    C -. 预览 .-> H["MIGRATE_DRY_RUN=1<br/>不执行只校验"]
```

两条使用路径，按需选择：

| 路径 | 适合 | 说明 |
| --- | --- | --- |
| A. 在本仓库内跑通 | 评估本库、快速上手 | 克隆仓库后用 `go run ./cmd/migrate`，迁移注册到仓库根的 `migrations/register.go` |
| B. 集成到自己的项目 | 生产使用 | `go get` 引入库，迁移写在自家包里，自建一个精简入口 |

路径 A 是最快的完整闭环，下文第 1～5 步以它为主线；路径 B 见[最后一节](#embed)。

## 第 1 步：安装

<a id="install"></a>

路径 A（本仓库内）：

```bash
git clone git@github.com:phgors/yiimigrate.git
cd yiimigrate
go build ./...
```

路径 B（自己的项目）：

```bash
go get github.com/phgors/yiimigrate
```

CLI（`cmd/migrate`）已经内置了四种方言的 `database/sql` 驱动，无需再单独安装：

```go
import (
    _ "github.com/go-sql-driver/mysql"
    _ "github.com/lib/pq"
    _ "github.com/microsoft/go-mssqldb"
    _ "modernc.org/sqlite"
)
```

注意：`cmd/migrate` 固定读取本仓库的 `migrations` 包来获得迁移列表。外部项目通过 `go get` 使用时，数据库命令（`up` 等）找不到你的迁移——路径 B 需要自建入口（见[下文](#embed)）；而 `create` 生成器属于 `internal/` 包，只在仓库内的 CLI 提供，外部项目请手写迁移文件（模板见[下文](#embed)）。

## 第 2 步：选择方言与 DSN

<a id="dialect"></a>

方言知识的唯一入口是 `migrate` 包的方言注册表（`migrate.DialectByName` / `migrate.DialectNames`），CLI 与生成器共用。四个注册名（规范名）：

`mysql`、`postgres`、`sqlite`、`sqlserver`

| DB_DIALECT 可用值 | 数据库 | 驱动（CLI 已内置） | DB_DSN 示例 |
| --- | --- | --- | --- |
| `mysql`（默认） | MySQL | github.com/go-sql-driver/mysql | `root:password@tcp(127.0.0.1:3306)/app?parseTime=true` |
| `postgres`、别名 `postgresql` | PostgreSQL | github.com/lib/pq | `postgres://user:password@127.0.0.1:5432/app?sslmode=disable` |
| `sqlite`、别名 `sqlite3` | SQLite | modernc.org/sqlite | `file:app.db`（内存库用 `:memory:`） |
| `sqlserver`、别名 `mssql` | SQL Server | github.com/microsoft/go-mssqldb | `sqlserver://user:password@127.0.0.1:1433?database=app` |

说明：

- `DB_DIALECT` 不区分大小写、忽略首尾空白；别名（`postgresql`、`sqlite3`、`mssql`）只在数据库命令的 `DB_DIALECT` 中可用，`create --dialect` 只接受规范名。
- DSN 格式由各方言的驱动定义，上表是常用写法；MySQL 迁移里若用 `Raw()` 执行多语句，DSN 需加 `multiStatements=true`。

> **SQLite 用户注意**：写操作命令（`up`、`down`、`redo`、`mark`、`to`）默认启用迁移锁，而 SQLite 没有 advisory lock，会直接报错 `sqlite does not support ADVISORY LOCK`。SQLite 下可以：用 `MIGRATE_DRY_RUN=1` 预览（dry-run 不取锁），或按[路径 B](#embed) 内嵌 `Migrator` 并设置 `UseLock = false`。

## 第 3 步：生成第一个迁移

以 MySQL 为例（其他方言加 `--dialect=...` 即可，差异见 [CLI 参考 -- create](cli.md#create)）：

```bash
go run ./cmd/migrate create \
  --fields="user_id:unsignedBigInteger:notNull,title:string(128):notNull,content:longText" \
  create_article_table
```

输出（`migrations/` 下的文件名时间戳取当前时刻，路径分隔符随操作系统）：

```text
created migrations/m20260613_120000_create_article_table.go
```

生成的文件内容如下（可直接编辑再补充列、索引等）：

```go
package migrations

import (
	"context"

	"github.com/phgors/yiimigrate/migrate"
)

type M20260613_120000CreateArticleTable struct{}

func (M20260613_120000CreateArticleTable) Name() string {
	return "m20260613_120000_create_article_table"
}

func (M20260613_120000CreateArticleTable) Up(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		CreateTable("article", migrate.Columns().
			Add("id", m.UnsignedBigPrimaryKey()).
			Add("user_id", m.UnsignedBigInteger().NotNull()).
			Add("title", m.String(128).NotNull()).
			Add("content", m.LongText().Null()),
			"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		).
		Exec(ctx)
}

func (M20260613_120000CreateArticleTable) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		DropTable("article").
		Exec(ctx)
}
```

两点规则：

- 旗标（`--fields`、`--dialect`）必须写在 `NAME` 之前，放在后面会得到 usage 报错（`--fields` 语法详见 [CLI 参考](cli.md#fields)）。
- 迁移名会被推导为表名：去掉 `create_` 前缀与 `_table` 后缀（`create_article_table` -> `article`），`Down` 生成对应的 `DropTable`。

## 第 4 步：注册迁移

<a id="register"></a>

CLI 通过 `migrations.All()` 获取迁移列表。编辑仓库根的 `migrations/register.go`，把生成文件里的类型加进去：

```go
package migrations

import "github.com/phgors/yiimigrate/migrate"

// All returns the migrations registered with this application.
func All() []migrate.Migration {
	return []migrate.Migration{
		M20260613_120000CreateArticleTable{}, // 类型名以你生成的文件为准
	}
}
```

迁移按 `Name()`（即文件名里的时间戳）排序执行，注册顺序不影响执行顺序。

## 第 5 步：up -> 历史表 -> down

设置环境变量（本例为 MySQL，其他方言见[第 2 步](#dialect)）：

```bash
export DB_DIALECT=mysql
export DB_DSN="root:password@tcp(127.0.0.1:3306)/app?parseTime=true"
```

查看待执行迁移：

```bash
go run ./cmd/migrate new
```

```text
m20260613_120000_create_article_table
```

应用迁移：

```bash
go run ./cmd/migrate up
```

```text
>>> applying m20260613_120000_create_article_table
<<< applied m20260613_120000_create_article_table
```

**迁移历史表自动创建**：所有数据库命令在执行前都会确保历史表存在，不存在则自动创建。默认表名为 `migration`（可用 `MIGRATE_TABLE` 覆盖），两列：`version varchar(180)` 主键 + `apply_time int`（Unix 秒）。MySQL 上的建表语句形如：

```sql
CREATE TABLE IF NOT EXISTS `migration` (`version` varchar(180) NOT NULL PRIMARY KEY, `apply_time` int NOT NULL)
```

（SQL Server 没有 `CREATE TABLE IF NOT EXISTS`，改为先查 `OBJECT_ID` 再建表，效果相同。）

核对历史记录：

```bash
go run ./cmd/migrate history
```

```text
m20260613_120000_create_article_table 1789012345
```

回滚最近一个迁移：

```bash
go run ./cmd/migrate down 1
```

```text
>>> reverting m20260613_120000_create_article_table
<<< reverted m20260613_120000_create_article_table
```

预览而不执行（不取锁、不写历史表、不执行任何迁移 SQL）：

```bash
MIGRATE_DRY_RUN=1 go run ./cmd/migrate up
```

```text
>>> applying m20260613_120000_create_article_table
<<< applied m20260613_120000_create_article_table
```

到这里首次迁移闭环就完成了。所有命令的参数、默认值与报错行为见 [CLI 参考](cli.md)。

## 集成到自己的项目

<a id="embed"></a>

`go get github.com/phgors/yiimigrate` 之后，三件事：

**1. 手写迁移文件**（与 `create` 生成的结构一致，不放本仓库也能写）：

```go
package migrations

import (
	"context"

	"github.com/phgors/yiimigrate/migrate"
)

type M20260613_120000CreateArticleTable struct{}

func (M20260613_120000CreateArticleTable) Name() string {
	return "m20260613_120000_create_article_table"
}

func (M20260613_120000CreateArticleTable) Up(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		CreateTable("article", migrate.Columns().
			Add("id", m.UnsignedBigPrimaryKey()).
			Add("title", m.String(128).NotNull()),
			"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		).
		Exec(ctx)
}

func (M20260613_120000CreateArticleTable) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().DropTable("article").Exec(ctx)
}
```

**2. 在自己的包里注册**（等价于仓库内的 `migrations/register.go`）：

```go
func All() []migrate.Migration {
	return []migrate.Migration{
		M20260613_120000CreateArticleTable{},
	}
}
```

**3. 自建精简入口**（`up` / `down` / `history` 等方法一一对应 CLI 子命令，按需接入自己的命令行框架）：

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"myapp/migrations" // 换成你的迁移包路径

	"github.com/phgors/yiimigrate/migrate"
)

func main() {
	db, err := sql.Open("mysql", os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()

	migrator := migrate.NewMigrator(db, migrate.MySQLDialect{}, migrations.All())
	// SQLite 等没有 advisory lock 的数据库需要关闭迁移锁：
	// migrator.UseLock = false

	if _, err := migrator.Up(context.Background(), 0); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

`NewMigrator` 返回的 `Migrator` 提供几个可调字段：`MigrationTable`（默认 `migration`）、`DryRun`、`UseLock` / `LockName` / `LockTimeoutSeconds`（默认开启，锁名 `yiimigrate`，超时 30 秒）。各字段语义与全部导出 API 见 [pkg.go.dev](https://pkg.go.dev/github.com/phgors/yiimigrate/migrate)。

## 下一步

- [迁移编写指南](migrations.md)：迁移结构、ColumnBuilder、SchemaPlan 与查询 helper 的 API 主参考。
- [CLI 参考](cli.md)：全部子命令、旗标、环境变量与报错行为。
- [examples/migrations/m20260613_120000_create_article_table.go](../examples/migrations/m20260613_120000_create_article_table.go)：一份更完整的迁移范例（幂等 `*IfExists` 变体、`RowExists` 查询分支、带 `migrate.Expr` 的 `Insert`）。
