# iphubtest 迁移示例设计

将 `E:\project\iphub\dev\web\php\site\app\migrations` 下的 18 个 Yii2 PHP 迁移,使用 `github.com/phgors/yiimigrate` 库重写为 Go 版本,作为完整可运行的示例保存到 `examples/iphubtest`。

原项目数据库名 `www_chaofanip_test`,示例使用 `go_migration_test`。

## 目标

- 提供一个 `cd examples/iphubtest && go run . up` 即可运行的端到端示例。
- 忠实翻译 18 个迁移的 `safeUp()` 逻辑,字段、索引、外键、注释、默认值与原 PHP 等价。
- `Down()` 与原 PHP 行为一致:原 PHP `safeDown()` 返回 `false` 的迁移,Go 版本返回 `migrate.ErrIrreversibleMigration`;原 PHP 实现了 `safeDown()` 的迁移(`m250907_000000_add_pay_method_to_user_recharge`、`m251025_150000_user_realname_verification`、`m251125_125129_add_setting`),按其逻辑翻译。
- 为每个迁移提供基于 `MySQLDialect` 的 SQL 生成单元测试,不依赖 MySQL。

## 目录结构

```
examples/iphubtest/
├── go.mod                  # 独立 module,replace 指向 ../..
├── main.go                 # CLI 入口,封装 up/down/redo/history/new/mark/to
├── README.md               # 使用说明
├── tablenames.go           # 对应 PHP TableNames.php,集中表名常量
├── setting.go              # 对应 PHP Setting 类,分类常量 + helper
├── register.go             # All() 返回全部迁移
├── migrations/
│   ├── m250718_142816_system.go
│   ├── m250718_142831_user.go
│   ├── m250803_073503_user_realname.go
│   ├── m250803_142638_product.go
│   ├── m250812_013259_product_line.go
│   ├── m250812_125424_cms.go
│   ├── m250816_073403_order.go
│   ├── m250907_000000_add_pay_method_to_user_recharge.go
│   ├── m250926_143220_user_auth_token.go
│   ├── m250927_100201_product_service.go
│   ├── m251020_122403_user_realname.go
│   ├── m251024_205915_user_realname.go
│   ├── m251025_150000_user_realname_verification.go
│   ├── m251103_084127_user_totp.go
│   ├── m251125_125129_add_setting.go
│   ├── m251219_145854_user_add_is_agent.go
│   ├── m260104_145509_user_add_qqwx.go
│   └── m260323_072857_product_order_intro.go
└── migrations/*_test.go    # SQL 生成单元测试(每文件一组)
```

## 关键技术决策

### 1. Go module 组织

`examples/iphubtest/go.mod`:

```
module github.com/phgors/yiimigrate/examples/iphubtest

go 1.26.2

require (
    github.com/phgors/yiimigrate v0.0.0-00010101000000-000000000000
    golang.org/x/crypto v0.50.0
)

replace github.com/phgors/yiimigrate => ../..
```

不污染根 `go.mod`,独立测试。

### 2. 表名常量(tablenames.go)

对照 `TableNames.php`,定义全部 `TN_*` 字符串常量。例:

```go
const (
    TNSystemSetting            = "system_setting"
    TNSystemMobileSms          = "system_mobile_sms"
    TNUser                     = "user"
    // ...
)
```

### 3. Setting 分类常量与 helper(setting.go)

原 PHP `Setting::CATEGORY_GENERAL` 等值无法从迁移脚本直接看出,采用直观的英文 slug:

```go
const (
    CategoryGeneral = "general"
    CategoryUser    = "user"
    CategorySms     = "sms"
    CategoryOrder   = "order"
)
```

提供两个 helper(基于 `MigrationContext`),覆盖原迁移中 `Setting::has` / `Setting::set` 用法:

```go
func SettingExists(ctx context.Context, m *migrate.MigrationContext, key, category string) (bool, error)
func InsertSetting(ctx context.Context, m *migrate.MigrationContext, category, name, key, value, description string) error
```

### 4. 密码哈希

引入 `golang.org/x/crypto/bcrypt`,与 PHP `password_hash($passwd, PASSWORD_DEFAULT)` 行为一致(PHP 默认就是 bcrypt cost 10)。

PHP 项目里 `USER_PASSWORD_HASH_METHOD == PASSWORD_HASH_PRE_MD5` 这条预处理分支无法从迁移代码看到取值,采用最直接的 bcrypt(对原文哈希),并在注释中说明。

```go
func hashPassword(plain string) (string, error) {
    b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
    return string(b), err
}
```

### 5. 时间函数

PHP `time()` 在迁移中插入数据库时,统一替换为 `migrate.Expr("UNIX_TIMESTAMP()")`,在 SQL 层执行,避免 Go 进程时区/时钟与 DB 不一致。

### 6. getLastInsertID

PHP 通过 `$this->db->getLastInsertID()` 取自增 ID 后用于后续 insert。Go 用查询 helper:

```go
v, err := m.QueryValue(ctx, "SELECT LAST_INSERT_ID()")
id, _ := v.(int64)
```

### 7. Yii2 字段类型映射

| Yii2 PHP | yiimigrate Go |
| --- | --- |
| `$this->primaryKey()->unsigned()` | `m.UnsignedPrimaryKey()` |
| `$this->bigPrimaryKey()->unsigned()` | `m.UnsignedBigPrimaryKey()` |
| `$this->integer()->unsigned()` | `m.UnsignedInteger()` |
| `$this->bigInteger()->unsigned()` | `m.UnsignedBigInteger()` |
| `$this->tinyInteger(1)->unsigned()` | `m.UnsignedTinyInteger(1)` |
| `$this->string(N)` | `m.String(N)` |
| `$this->text()` | `m.Text()` |
| `$this->longtext` (createColumnSchemaBuilder) | `m.LongText()` |
| `$this->json()` | `m.Json()` |
| `$this->boolean()` | `m.Boolean()` |
| `$this->decimal(p, s)` | `m.Decimal(p, s)` |
| `$this->decimal(p, s)->unsigned()` | `m.Decimal(p, s).Unsigned()` |
| `->notNull()` | `.NotNull()` |
| `->null()` | `.Null()` |
| `->defaultValue(v)` | `.DefaultValue(v)` |
| `->comment('xxx')` | `.Comment("xxx")` |
| `->unique()` | `.Unique()` |
| `->after('col')` | `.After("col")` |

### 8. 表选项与注释

原 PHP `createTable(..., 'comment="xxx"')` 是非标准语法,实际 Yii2 在 MySQL 下拼到表 option。Go 版本使用更标准的 `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='xxx'` 作为表 option 字符串。

### 9. createIndex / addForeignKey / addPrimaryKey

直接对应 `m.Schema().CreateIndex|AddForeignKey|AddPrimaryKey` 等。`{{%table}}` 语法糖剥除,直接使用纯表名常量。

### 10. batchInsert 设置项

`m.Schema().BatchInsert(table, columns, rows)`,普通值参数绑定,布尔/数字/字符串直接传 Go 字面量。

### 11. update() / Setting 条件插入

PHP 中条件插入 setting 的模式:

```php
$exists = (new Query())->from(...)->where(['key' => 'xxx', 'category' => 'yyy'])->exists();
if (!$exists) { $this->insert(...); }
```

翻译为:

```go
exists, err := SettingExists(ctx, m, "xxx", "yyy")
if err != nil { return err }
if !exists {
    if err := InsertSetting(ctx, m, CategoryUser, "中文名", "xxx", "0", "描述"); err != nil {
        return err
    }
}
```

`update()` 调用翻译为 `m.Schema().Update(table, row, condition, args...)`。

### 12. Down 行为

- 原 PHP `safeDown()` 返回 `false` 的迁移(大多数):Go 版 `Down` 返回 `migrate.ErrIrreversibleMigration`。
- 原 PHP `safeDown()` 有真实逻辑的 3 个迁移,翻译为对应的 `DropIndex/DropColumn/DropForeignKey/DropTable` 链。
- 原 PHP `safeDown()` 中既写了回滚逻辑又 `return false` 的(`m251025_150000_user_realname_verification`),按字面翻译:执行回滚链,最后返回 `ErrIrreversibleMigration`,与原 PHP 一致。

### 13. CLI(main.go)

复用根 `cmd/migrate` 的命令结构(up/down/redo/history/new/mark/to),环境变量:

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `DB_DSN` | `root:@tcp(127.0.0.1:3306)/go_migration_test?parseTime=true&multiStatements=true` | 数据库 DSN |
| `DB_DIALECT` | `mysql` | 固定 mysql |
| `MIGRATE_DRY_RUN` | 空 | `1` 启用 dry-run |
| `MIGRATE_TABLE` | `migration` | 迁移记录表 |

`UseLock = true`(MySQL 默认),`LockName = "iphubtest"`。

### 14. 单元测试

每个迁移文件配 `<file>_test.go`,纯 SQL 生成测试,不依赖 MySQL:

- 用 `migrate.NewMigrationContext(nil, migrate.MySQLDialect{}, migrate.WithDryRun(true))` 构造 ctx。
- 调用 `Up(ctx, m)`,然后通过 `m.Schema()...` 链尾的 `Statements()`(需库新增,见下)断言生成的 SQL 关键片段。
- 不依赖 `MYSQL_TEST_DSN`,可在 CI 直接跑。

**对库的小幅扩展(单独提交)**:给 `migrate.SchemaPlan` 增加只读访问器:

```go
// Statements returns the accumulated SQL statements in the plan.
// Intended for inspection in tests; the slice is a defensive copy.
func (p *SchemaPlan) Statements() []SQLStatement
```

该访问器对生产代码无害(只读),让外部包能在 dry-run 模式下断言 SQL 生成结果。

### 15. README.md

包含:
- 项目来源说明(从 Yii2 PHP 迁移翻译)
- 准备 MySQL 数据库 `go_migration_test` 的 SQL
- 环境变量说明
- 运行命令示例(up/down/redo/history/new)
- 与原 PHP 对照的迁移清单表

## 不在范围内

- 不实现原 PHP 的业务逻辑(只翻译 DDL/DML)。
- 不修改根 `cmd/migrate`、根 `migrations/register.go`、根 `examples/migrations`。
- 不引入除 `golang.org/x/crypto/bcrypt` 之外的新依赖。
- 不为示例编写 MySQL 集成测试(用户可不设置 `MYSQL_TEST_DSN` 直接跑 `go test ./...`)。

## 验证清单

- [ ] `cd examples/iphubtest && go build ./...`
- [ ] `cd examples/iphubtest && go test ./...`(无 MySQL)
- [ ] `cd examples/iphubtest && go vet ./...`
- [ ] `gofmt -l examples/iphubtest` 无输出
- [ ] 根目录 `go test ./...` 不受影响
