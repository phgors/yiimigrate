# yiimigrate

`yiimigrate` 是一个受 Yii2 Migration 启发的 Go 数据库迁移库。它只使用标准库 `database/sql`（不引入 ORM），提供接近 Yii2 体验的字段声明、链式 Schema 操作、迁移执行、dry-run、迁移锁和迁移内查询 helper。

当前版本（v0.1.0）已实现四个方言：**MySQL、PostgreSQL、SQL Server、SQLite**。四个方言全部有一致性矩阵单元测试覆盖；其中 MySQL、PostgreSQL、SQL Server 另有集成测试，分别由 `MYSQL_TEST_DSN`、`POSTGRES_TEST_DSN`、`SQLSERVER_TEST_DSN` 环境变量触发（未设置时自动跳过）。

## 特性

- 八个 CLI 命令：`up` / `down` / `redo` / `history` / `new` / `mark` / `to` / `create`。
- 每个迁移默认在自己的事务里执行；迁移历史表（默认表名 `migration`）自动创建。
- 不可变链式 `ColumnBuilder` + 有序 `Columns()`，`CREATE TABLE` 字段顺序稳定。
- 链式 `SchemaPlan` 执行 DDL 与 DML；十个 `*IfExists` 变体用存在性探针提供幂等。
- 查询 helper：`RowExists` / `CountRows` / `QueryValue` / `QueryOne` / `QueryAll`；DML 普通值参数绑定，`migrate.Expr(...)` 作为 SQL 表达式直拼。
- 数据库给不了的操作显式返回 `UnsupportedOperationError`，不静默降级；drop 一律严格。
- 迁移锁默认开启（MySQL `GET_LOCK`、PostgreSQL `pg_advisory_lock`、SQL Server `sp_getapplock`）；SQLite 无 advisory lock，可 dry-run 预览或内嵌时设 `UseLock = false`。
- dry-run 只生成并打印 SQL，不执行迁移 SQL、不写历史表、不取锁。

## 安装

```bash
go get github.com/phgors/yiimigrate
```

CLI（`cmd/migrate`）已内置四个方言的 `database/sql` 驱动，无需单独安装。

## 快速开始

在仓库根目录（完整闭环见[快速上手](docs/getting-started.md)）：

```bash
# 生成迁移文件（旗标必须写在 NAME 之前）
go run ./cmd/migrate create --fields="title:string(128):notNull" create_article_table

# 应用迁移（以 MySQL 为例）
export DB_DSN="root:password@tcp(127.0.0.1:3306)/app?parseTime=true"
go run ./cmd/migrate up

# 回滚最近一个迁移
go run ./cmd/migrate down 1

# 预览 SQL 而不执行
MIGRATE_DRY_RUN=1 go run ./cmd/migrate up
```

生成的迁移就是一个实现 `Name()` / `Up()` / `Down()` 的普通 Go 类型，迁移风格与 Yii2 相近：

```go
func (M20260613_120000CreateArticleTable) Up(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		CreateTable("article", migrate.Columns().
			Add("id", m.UnsignedBigPrimaryKey()).
			Add("title", m.String(128).NotNull()),
			"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		).
		Exec(ctx)
}
```

## 文档

| 文档 | 内容 |
| --- | --- |
| [快速上手](docs/getting-started.md) | 从零跑通第一次迁移：安装、方言与 DSN、生成、注册、`up` -> `history` -> `down`，集成到自己的项目与跑测试 |
| [迁移编写指南](docs/migrations.md) | API 主参考：迁移结构、ColumnBuilder、SchemaPlan 与 `*IfExists` 探针、查询 helper、迁移锁与 dry-run、Yii2 对照表 |
| [方言参考](docs/dialects.md) | 四方言能力矩阵、各方言注意事项、新增方言的扩展指南 |
| [CLI 参考](docs/cli.md) | 八个子命令、`create` 旗标、环境变量、迁移锁与报错行为 |
| [架构](docs/architecture.md) | 模块图（Dialect seam）、一次 `migrate up` 的执行时序、CLI 调用路径 |
| [排错指南](docs/troubleshooting.md) | 常见错误「现象 -> 原因 -> 处置」目录与已知限制 |
| [升级注意](docs/upgrade-notes.md) | v0.1.0 行为变化的旧 / 新对照与迁移建议 |
| [CHANGELOG](CHANGELOG.md) | 按版本归档的变更历史 |

导出 API 的逐项明细见 [pkg.go.dev 文档](https://pkg.go.dev/github.com/phgors/yiimigrate/migrate)。
