# Multi-Database Yii2 Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the project from MySQL-hard-coded CLI/documentation toward explicit, tested multi-database support while keeping current MySQL and SQLite dialects real and honest.

**Architecture:** Keep `migrate.Dialect` as the SQL generation boundary. Add a small CLI dialect selector that maps configured dialect names to `database/sql` driver names and `migrate.Dialect` implementations, then document implemented and planned databases separately.

**Tech Stack:** Go, `database/sql`, `github.com/go-sql-driver/mysql`, `modernc.org/sqlite`, existing `migrate` package dialects, standard `testing`.

---

## File Structure

- Modify `cmd/migrate/main.go`: add `DB_DIALECT` handling, choose the correct SQL driver and `migrate.Dialect`, and keep `create` command independent of database configuration.
- Create `cmd/migrate/main_test.go`: test dialect resolution, default behavior, aliases, and unsupported values without opening real databases.
- Modify `go.mod` and `go.sum`: add a pure-Go SQLite driver so the CLI can open SQLite without CGO.
- Modify `README.md`: update the support statement, CLI configuration, MySQL notes, SQLite notes, and planned dialect policy.
- Modify `IMPLEMENTATION.md`: add the CLI dialect selection requirement to the implementation guide.
- Run `gofmt` on changed Go files and `go test ./...`.

---

### Task 1: Add CLI Dialect Resolution Tests

**Files:**
- Create: `cmd/migrate/main_test.go`
- Modify: none
- Test: `cmd/migrate/main_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/migrate/main_test.go` with:

```go
package main

import (
	"strings"
	"testing"

	"github.com/phgors/yiimigrate/migrate"
)

func TestResolveDBConfigDefaultsToMySQL(t *testing.T) {
	config, err := resolveDBConfig("")
	if err != nil {
		t.Fatalf("resolveDBConfig returned error: %v", err)
	}
	if config.driverName != "mysql" {
		t.Fatalf("driverName = %q, want mysql", config.driverName)
	}
	if _, ok := config.dialect.(migrate.MySQLDialect); !ok {
		t.Fatalf("dialect = %T, want migrate.MySQLDialect", config.dialect)
	}
}

func TestResolveDBConfigSupportsSQLiteAliases(t *testing.T) {
	tests := []string{"sqlite", "sqlite3"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			config, err := resolveDBConfig(input)
			if err != nil {
				t.Fatalf("resolveDBConfig returned error: %v", err)
			}
			if config.driverName != "sqlite" {
				t.Fatalf("driverName = %q, want sqlite", config.driverName)
			}
			if _, ok := config.dialect.(migrate.SQLiteDialect); !ok {
				t.Fatalf("dialect = %T, want migrate.SQLiteDialect", config.dialect)
			}
		})
	}
}

func TestResolveDBConfigRejectsUnsupportedDialect(t *testing.T) {
	_, err := resolveDBConfig("postgres")
	if err == nil {
		t.Fatal("resolveDBConfig returned nil error")
	}
	if !strings.Contains(err.Error(), `unsupported DB_DIALECT "postgres"`) {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "mysql, sqlite") {
		t.Fatalf("error does not list supported dialects: %v", err)
	}
}
```

- [ ] **Step 2: Run the tests to verify failure**

Run:

```bash
go test ./cmd/migrate
```

Expected: fail with `undefined: resolveDBConfig`.

- [ ] **Step 3: Commit the failing tests**

```bash
git add cmd/migrate/main_test.go
git commit -m "test: define cli dialect selection behavior"
```

---

### Task 2: Implement CLI Dialect Selection

**Files:**
- Modify: `cmd/migrate/main.go`
- Modify: `go.mod`
- Modify: `go.sum`
- Test: `cmd/migrate/main_test.go`

- [ ] **Step 1: Add the SQLite driver dependency**

Run:

```bash
go get modernc.org/sqlite
```

Expected: `go.mod` gains `modernc.org/sqlite` and `go.sum` gains its checksums.

- [ ] **Step 2: Update CLI imports and add config resolution**

In `cmd/migrate/main.go`, update the import block to include `strings` and the SQLite driver:

```go
import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"

	"github.com/phgors/yiimigrate/internal/generator"
	"github.com/phgors/yiimigrate/migrate"
	"github.com/phgors/yiimigrate/migrations"
)
```

Add this type and function near `runDBCommand`:

```go
type dbConfig struct {
	driverName string
	dialect    migrate.Dialect
}

func resolveDBConfig(name string) (dbConfig, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "mysql":
		return dbConfig{driverName: "mysql", dialect: migrate.MySQLDialect{}}, nil
	case "sqlite", "sqlite3":
		return dbConfig{driverName: "sqlite", dialect: migrate.SQLiteDialect{}}, nil
	default:
		return dbConfig{}, fmt.Errorf("unsupported DB_DIALECT %q; supported values: mysql, sqlite", name)
	}
}
```

- [ ] **Step 3: Use the resolved config in `runDBCommand`**

Replace:

```go
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	migrator := migrate.NewMigrator(db, migrate.MySQLDialect{}, migrations.All())
```

with:

```go
	config, err := resolveDBConfig(os.Getenv("DB_DIALECT"))
	if err != nil {
		return err
	}
	db, err := sql.Open(config.driverName, dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	migrator := migrate.NewMigrator(db, config.dialect, migrations.All())
```

- [ ] **Step 4: Run gofmt**

Run:

```bash
gofmt -w cmd/migrate/main.go cmd/migrate/main_test.go
```

Expected: no output.

- [ ] **Step 5: Run the focused tests**

Run:

```bash
go test ./cmd/migrate
```

Expected: pass.

- [ ] **Step 6: Commit the implementation**

```bash
git add cmd/migrate/main.go cmd/migrate/main_test.go go.mod go.sum
git commit -m "feat: select cli database dialect"
```

---

### Task 3: Document Implemented and Planned Database Support

**Files:**
- Modify: `README.md`
- Modify: `IMPLEMENTATION.md`
- Test: documentation scan and full tests

- [ ] **Step 1: Update the README opening support statement**

Replace the current MySQL-only paragraph near the top of `README.md` with:

```md
当前版本采用多方言架构，已实现并测试 MySQL 和 SQLite。PostgreSQL、SQL Server 等数据库应通过真实 Dialect 逐步加入；README 只宣称已经实现并覆盖测试的数据库。
```

- [ ] **Step 2: Update the README feature list**

Replace:

```md
- MySQL 下使用 `GET_LOCK` / `RELEASE_LOCK` 串行化迁移。
```

with:

```md
- MySQL 下使用 `GET_LOCK` / `RELEASE_LOCK` 串行化迁移；SQLite 等不支持 advisory lock 的方言会返回明确错误，调用方可关闭 `UseLock`。
```

- [ ] **Step 3: Update the CLI environment table**

Replace the CLI environment table rows with:

```md
| 变量 | 说明 |
| --- | --- |
| `DB_DIALECT` | 数据库方言，支持 `mysql`、`sqlite`，默认 `mysql` |
| `DB_DSN` | 数据库 DSN；MySQL 示例：`root:password@tcp(127.0.0.1:3306)/test?parseTime=true`；SQLite 示例：`file:dev.db` |
| `MIGRATE_DRY_RUN` | 设置为 `1`、`true`、`TRUE` 或 `yes` 时启用 dry-run |
| `MIGRATE_TABLE` | 自定义迁移记录表名，默认 `migration` |
```

- [ ] **Step 4: Add SQLite notes after MySQL notes**

After the `## MySQL 注意事项` list, add:

```md
## SQLite 注意事项

- 使用 `SQLiteDialect`，CLI 中设置 `DB_DIALECT=sqlite`。
- 标识符使用双引号 quote。
- 占位符使用 `?`。
- SQLite 支持 `CREATE TABLE`、`DROP TABLE`、`RENAME TABLE`、`ADD COLUMN`、索引和 DML。
- SQLite 不支持的后置 DDL 操作会返回 `UnsupportedOperationError`，例如 `ALTER COLUMN`、`DROP COLUMN`、后置外键和注释。
- SQLite advisory lock 不可用；运行 mutating CLI 命令时需要关闭 `UseLock` 的场景应通过后续配置项处理。
```

- [ ] **Step 5: Update `IMPLEMENTATION.md` CLI task**

In the CLI task section, ensure requirement 13 says:

```md
13. 根据 `DB_DIALECT` 选择已实现的数据库方言，默认 `mysql`，支持 `sqlite`。
```

- [ ] **Step 6: Scan for stale hard limits**

Run:

```bash
rg -n "当前阶段仅支持 MySQL|当前仅实现 `MySQLDialect`|当前只支持 MySQL|SQLite、PostgreSQL、SQL Server 尚未实现|不要实现 PostgresDialect|不要实现 PostgreSQL" README.md IMPLEMENTATION.md AGENTS.md
```

Expected: no output.

- [ ] **Step 7: Commit documentation**

```bash
git add README.md IMPLEMENTATION.md
git commit -m "docs: describe database dialect support"
```

---

### Task 4: Final Verification

**Files:**
- Verify all changed files

- [ ] **Step 1: Run gofmt on changed Go files**

Run:

```bash
gofmt -w cmd/migrate/main.go cmd/migrate/main_test.go
```

Expected: no output.

- [ ] **Step 2: Run all tests**

Run:

```bash
go test ./...
```

Expected: all packages pass. MySQL integration tests skip unless `MYSQL_TEST_DSN` is set.

- [ ] **Step 3: Inspect git status**

Run:

```bash
git status --short
```

Expected: no output.

- [ ] **Step 4: Report completion evidence**

Final report should include:

```txt
- Added DB_DIALECT-based CLI dialect selection for mysql and sqlite.
- Added tests for default, sqlite aliases, and unsupported dialect errors.
- Updated README/IMPLEMENTATION database support wording.
- Verified with gofmt and go test ./...
```

---

## Self-Review

Spec coverage:

- Multi-database direction: covered by CLI dialect resolution and documentation tasks.
- Real tested dialect policy: covered by README and IMPLEMENTATION updates.
- SQLite CLI path: covered by adding `modernc.org/sqlite` and `DB_DIALECT=sqlite`.
- Unsupported values: covered by `resolveDBConfig` test.
- Full verification: covered by `gofmt` and `go test ./...`.

Placeholder scan: no task contains open-ended placeholders.

Type consistency: `dbConfig`, `resolveDBConfig`, `driverName`, and `dialect` are introduced in Task 2 and used consistently by the tests from Task 1.
