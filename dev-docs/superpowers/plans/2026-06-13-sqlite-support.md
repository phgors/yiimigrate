# SQLite Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a usable `database/sql` migration core with real SQLite dialect support.

**Architecture:** Add small focused files under `migrate/`: immutable column builders, ordered column lists, dialect interfaces, MySQL and SQLite SQL generators, schema plans, and query helpers. SQLite unsupported DDL is represented by explicit errors captured by `SchemaPlan`.

**Tech Stack:** Go 1.26, `database/sql`, standard library tests, no ORM dependencies, no required database drivers for unit tests.

---

### Task 1: Core Types And Builders

**Files:**
- Create: `migrate/types.go`
- Create: `migrate/column_builder.go`
- Create: `migrate/columns.go`
- Test: `migrate/column_builder_test.go`

- [ ] **Step 1: Write tests for immutable builders and ordered columns**

```go
func TestColumnBuilderImmutable(t *testing.T) {
	base := NewMigrationContext(nil, SQLiteDialect{}).String(64)
	notNull := base.NotNull()
	nullable := base.Null()
	if notNull == base || nullable == base {
		t.Fatal("builder methods must return clones")
	}
}

func TestColumnsKeepOrder(t *testing.T) {
	cols := Columns().
		Add("id", NewMigrationContext(nil, SQLiteDialect{}).PrimaryKey()).
		Add("title", NewMigrationContext(nil, SQLiteDialect{}).String(128))
	got := []string{cols.Items()[0].Name, cols.Items()[1].Name}
	want := []string{"id", "title"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("columns order = %#v, want %#v", got, want)
	}
}
```

- [ ] **Step 2: Implement core types and immutable builders**

Create `Row`, `Expression`, `Expr`, `DBTX`, `SQLStatement`, `MigrationContext`, `ColumnBuilder`, `ColumnList`, and Yii-like builder methods. Every public type and method gets a Go doc comment.

- [ ] **Step 3: Verify task tests**

Run: `go test ./migrate -run 'TestColumnBuilderImmutable|TestColumnsKeepOrder'`

Expected: tests pass.

### Task 2: Dialect Interface, MySQL, And SQLite SQL Generation

**Files:**
- Create: `migrate/dialect.go`
- Create: `migrate/mysql.go`
- Create: `migrate/sqlite.go`
- Test: `migrate/dialect_test.go`

- [ ] **Step 1: Write SQL generation tests**

Cover SQLite identifier quoting, `CREATE TABLE`, `CREATE INDEX`, metadata SQL, DML SQL, unsupported operations, MySQL `CREATE TABLE`, and expression index quoting.

- [ ] **Step 2: Implement the dialect interface and both dialects**

`SQLiteDialect` supports create/drop/rename table, add column, indexes, DML, existence SQL, and query helper SQL. Unsupported SQLite DDL returns `UnsupportedOperationError`.

- [ ] **Step 3: Verify dialect tests**

Run: `go test ./migrate -run 'TestSQLite|TestMySQL|TestUnsupported'`

Expected: tests pass.

### Task 3: SchemaPlan And Query Helpers

**Files:**
- Create: `migrate/schema_plan.go`
- Create: `migrate/query.go`
- Test: `migrate/schema_plan_test.go`

- [ ] **Step 1: Write tests for plan accumulation and dry-run behavior**

Use a lightweight fake `DBTX` for `ExecContext` tests. Validate that accumulated errors prevent execution and dry-run skips execution.

- [ ] **Step 2: Implement schema plan and query helpers**

Add `Raw`, DDL methods, IfExists/IfNotExists helpers, DML methods, `Exec`, `QueryValue`, `QueryOne`, `QueryAll`, `RowExists`, and `CountRows`.

- [ ] **Step 3: Verify plan tests**

Run: `go test ./migrate -run 'TestSchemaPlan|TestDryRun|TestQuerySQL'`

Expected: tests pass.

### Task 4: Project Phase Documentation

**Files:**
- Modify: `AGENTS.md`
- Modify: `IMPLEMENTATION.md`

- [ ] **Step 1: Update support policy wording**

Replace MySQL-only phase language with MySQL plus SQLite support. Keep PostgreSQL and SQL Server out of scope.

- [ ] **Step 2: Verify policy text**

Run: `rg -n "MySQL is the only supported|Do not implement SQLite|不要实现.*SQLite|仅支持 MySQL|只实现 MySQL" AGENTS.md IMPLEMENTATION.md`

Expected: no matches.

### Task 5: Full Verification

**Files:**
- All changed Go files

- [ ] **Step 1: Format Go files**

Run: `gofmt -w migrate/*.go`

- [ ] **Step 2: Run the full test suite**

Run: `go test ./...`

Expected: all packages pass.

- [ ] **Step 3: Review final diff**

Run: `git diff --stat` and `git diff --check`

Expected: no whitespace errors and changes match the SQLite support scope.
