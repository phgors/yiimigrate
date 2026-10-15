# PostgreSQL & SQL Server Dialect Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use compose:subagent (recommended) or compose:execute to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add complete PostgreSQL and SQL Server dialect implementations to the migration library.

**Architecture:** Two independent dialect files (`postgres.go`, `sqlserver.go`) implementing the existing `Dialect` interface (38 methods), following the patterns established by `mysql.go` and `sqlite.go`. Each dialect has its own unit test file and integration test file.

**Tech Stack:** Go, `database/sql`, `github.com/lib/pq` (PG driver), `github.com/microsoft/go-mssqldb` (MSSQL driver)

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `migrate/postgres.go` | Create | PostgreSQL dialect implementation |
| `migrate/sqlserver.go` | Create | SQL Server dialect implementation |
| `migrate/postgres_test.go` | Create | PG SQL generation unit tests |
| `migrate/sqlserver_test.go` | Create | MSSQL SQL generation unit tests |
| `migrate/postgres_integration_test.go` | Create | PG integration tests (POSTGRES_TEST_DSN) |
| `migrate/sqlserver_integration_test.go` | Create | MSSQL integration tests (SQLSERVER_TEST_DSN) |
| `cmd/migrate/main.go` | Modify | Add PG/MSSQL to resolveDBConfig + driver imports |

---

### Task 1: PostgreSQL Dialect — Core Structure & BuildColumn

**Covers:** [S4]

**Files:**
- Create: `migrate/postgres.go`

- [ ] **Step 1: Write the PostgreSQL dialect file with all 38 methods**

Create `migrate/postgres.go`:

```go
package migrate

import (
	"fmt"
	"strings"
)

type PostgreSQLDialect struct{}

func (d PostgreSQLDialect) Name() string { return "postgres" }

func (d PostgreSQLDialect) QuoteTable(name string) string { return quoteName(name, `"`) }

func (d PostgreSQLDialect) QuoteColumn(name string) string { return quoteName(name, `"`) }

func (d PostgreSQLDialect) QuoteIndexColumn(name string) string { return quoteIndexColumn(d, name) }

func (d PostgreSQLDialect) Placeholder(index int) string { return fmt.Sprintf("$%d", index) }

func (d PostgreSQLDialect) BuildColumn(c *ColumnBuilder) (string, error) {
	if c == nil {
		return "", fmt.Errorf("migrate: column builder is nil")
	}
	if c.unsigned {
		return "", unsupported(d.Name(), "UNSIGNED")
	}
	if c.charset != "" {
		return "", unsupported(d.Name(), "CHARACTER SET")
	}
	if c.collation != "" {
		return "", unsupported(d.Name(), "COLLATE")
	}
	if c.after != "" {
		return "", unsupported(d.Name(), "AFTER")
	}
	if c.first {
		return "", unsupported(d.Name(), "FIRST")
	}

	if c.primaryKey && c.autoIncrement {
		parts := []string{pgSerialType(c)}
		if c.appendSQL != "" {
			parts = append(parts, c.appendSQL)
		}
		return strings.Join(parts, " "), nil
	}

	parts := []string{pgType(c)}
	if c.nullSet {
		if c.nullable {
			parts = append(parts, "NULL")
		} else {
			parts = append(parts, "NOT NULL")
		}
	}
	if c.primaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if c.unique {
		parts = append(parts, "UNIQUE")
	}
	if c.defaultExpr != "" {
		parts = append(parts, "DEFAULT", c.defaultExpr)
	}
	if c.defaultSet {
		parts = append(parts, "DEFAULT", sqlLiteral(c.defaultValue))
	}
	if c.check != "" {
		parts = append(parts, "CHECK ("+c.check+")")
	}
	if c.generatedAs != "" {
		parts = append(parts, "GENERATED ALWAYS AS ("+c.generatedAs+")")
		if c.generatedKind != "" {
			parts = append(parts, c.generatedKind)
		}
	}
	if c.appendSQL != "" {
		parts = append(parts, c.appendSQL)
	}
	return strings.Join(parts, " "), nil
}

func pgSerialType(c *ColumnBuilder) string {
	switch c.typeName {
	case "integer":
		return "SERIAL PRIMARY KEY"
	case "bigInteger":
		return "BIGSERIAL PRIMARY KEY"
	default:
		return "SERIAL PRIMARY KEY"
	}
}

func pgType(c *ColumnBuilder) string {
	switch c.typeName {
	case "tinyInteger":
		return "SMALLINT"
	case "smallInteger":
		return "SMALLINT"
	case "integer":
		return "INTEGER"
	case "bigInteger":
		return "BIGINT"
	case "string":
		return sizedType("VARCHAR", c.size, 255)
	case "char":
		return sizedType("CHAR", c.size)
	case "text", "tinyText", "mediumText", "longText":
		return "TEXT"
	case "binary", "tinyBlob", "mediumBlob", "longBlob":
		return "BYTEA"
	case "boolean":
		return "BOOLEAN"
	case "float":
		return "REAL"
	case "double":
		return "DOUBLE PRECISION"
	case "decimal", "money":
		return sizedType("DECIMAL", c.size)
	case "date":
		return "DATE"
	case "dateTime", "timestamp":
		return precisionType("TIMESTAMP", c.size)
	case "time":
		return precisionType("TIME", c.size)
	case "json":
		return "JSONB"
	case "uuid":
		return "UUID"
	case "enum":
		return sizedType("VARCHAR", c.size, 255)
	case "set":
		return sizedType("VARCHAR", c.size, 255)
	default:
		return strings.ToUpper(c.typeName)
	}
}
```

- [ ] **Step 2: Add remaining DDL methods to postgres.go**

Append these methods to `migrate/postgres.go`:

```go
func (d PostgreSQLDialect) TableExistsSQL(table string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1",
		Args:  []any{table},
	}
}

func (d PostgreSQLDialect) ColumnExistsSQL(table, column string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2",
		Args:  []any{table, column},
	}
}

func (d PostgreSQLDialect) IndexExistsSQL(table, index string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1 AND indexname = $2",
		Args:  []any{table, index},
	}
}

func (d PostgreSQLDialect) ForeignKeyExistsSQL(table, name string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = current_schema() AND table_name = $1 AND constraint_name = $2 AND constraint_type = 'FOREIGN KEY'",
		Args:  []any{table, name},
	}
}

func (d PostgreSQLDialect) ConstraintExistsSQL(table, name string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = current_schema() AND table_name = $1 AND constraint_name = $2",
		Args:  []any{table, name},
	}
}

func (d PostgreSQLDialect) BuildRowExistsSQL(table string, condition string) string {
	query := "SELECT EXISTS(SELECT 1 FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return query + ")"
}

func (d PostgreSQLDialect) BuildCountRowsSQL(table string, condition string) string {
	query := "SELECT COUNT(*) FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return query
}

func (d PostgreSQLDialect) CreateTable(table string, columns *ColumnList, options string) (string, error) {
	defs := make([]string, 0, len(columns.Items()))
	for _, item := range columns.Items() {
		columnSQL, err := d.BuildColumn(item.Column)
		if err != nil {
			return "", err
		}
		defs = append(defs, d.QuoteColumn(item.Name)+" "+columnSQL)
	}
	return fmt.Sprintf("CREATE TABLE %s (%s)%s", d.QuoteTable(table), strings.Join(defs, ", "), joinOptions(options)), nil
}

func (d PostgreSQLDialect) DropTable(table string) (string, error) {
	return "DROP TABLE IF EXISTS " + d.QuoteTable(table), nil
}

func (d PostgreSQLDialect) RenameTable(oldName, newName string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s RENAME TO %s", d.QuoteTable(oldName), d.QuoteTable(newName)), nil
}

func (d PostgreSQLDialect) TruncateTable(table string) (string, error) {
	return "TRUNCATE TABLE " + d.QuoteTable(table), nil
}

func (d PostgreSQLDialect) AddColumn(table, column string, builder *ColumnBuilder) (string, error) {
	columnSQL, err := d.BuildColumn(builder)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", d.QuoteTable(table), d.QuoteColumn(column), columnSQL), nil
}

func (d PostgreSQLDialect) AlterColumn(table, column string, builder *ColumnBuilder) (string, error) {
	columnSQL, err := d.BuildColumn(builder)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", d.QuoteTable(table), d.QuoteColumn(column), columnSQL), nil
}

func (d PostgreSQLDialect) DropColumn(table, column string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", d.QuoteTable(table), d.QuoteColumn(column)), nil
}

func (d PostgreSQLDialect) RenameColumn(table, oldName, newName string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", d.QuoteTable(table), d.QuoteColumn(oldName), d.QuoteColumn(newName)), nil
}

func (d PostgreSQLDialect) CreateIndex(name, table string, columns []string, unique bool) (string, error) {
	prefix := "CREATE INDEX"
	if unique {
		prefix = "CREATE UNIQUE INDEX"
	}
	return fmt.Sprintf("%s %s ON %s (%s)", prefix, d.QuoteTable(name), d.QuoteTable(table), columnList(d, columns)), nil
}

func (d PostgreSQLDialect) DropIndex(name, table string) (string, error) {
	return "DROP INDEX IF EXISTS " + d.QuoteTable(name), nil
}

func (d PostgreSQLDialect) AddPrimaryKey(name, table string, columns []string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s)", d.QuoteTable(table), d.QuoteTable(name), columnList(d, columns)), nil
}

func (d PostgreSQLDialect) DropPrimaryKey(name, table string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", d.QuoteTable(table), d.QuoteTable(name)), nil
}

func (d PostgreSQLDialect) AddForeignKey(name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) (string, error) {
	query := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)", d.QuoteTable(table), d.QuoteTable(name), columnList(d, columns), d.QuoteTable(refTable), columnList(d, refColumns))
	if onDelete != "" {
		query += " ON DELETE " + string(onDelete)
	}
	if onUpdate != "" {
		query += " ON UPDATE " + string(onUpdate)
	}
	return query, nil
}

func (d PostgreSQLDialect) DropForeignKey(name, table string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", d.QuoteTable(table), d.QuoteTable(name)), nil
}

func (d PostgreSQLDialect) AddCommentOnColumn(table, column, comment string) (string, error) {
	return fmt.Sprintf("COMMENT ON COLUMN %s.%s IS %s", d.QuoteTable(table), d.QuoteColumn(column), sqlLiteral(comment)), nil
}

func (d PostgreSQLDialect) DropCommentFromColumn(table, column string) (string, error) {
	return fmt.Sprintf("COMMENT ON COLUMN %s.%s IS NULL", d.QuoteTable(table), d.QuoteColumn(column)), nil
}

func (d PostgreSQLDialect) AddCommentOnTable(table, comment string) (string, error) {
	return fmt.Sprintf("COMMENT ON TABLE %s IS %s", d.QuoteTable(table), sqlLiteral(comment)), nil
}

func (d PostgreSQLDialect) DropCommentFromTable(table string) (string, error) {
	return fmt.Sprintf("COMMENT ON TABLE %s IS NULL", d.QuoteTable(table)), nil
}

func (d PostgreSQLDialect) Insert(table string, row Row) (SQLStatement, error) {
	return buildInsert(d, table, row)
}

func (d PostgreSQLDialect) BatchInsert(table string, columns []string, rows [][]any) (SQLStatement, error) {
	return buildBatchInsert(d, table, columns, rows)
}

func (d PostgreSQLDialect) Update(table string, row Row, condition string, args ...any) (SQLStatement, error) {
	return buildUpdate(d, table, row, condition, args...)
}

func (d PostgreSQLDialect) Delete(table string, condition string, args ...any) (SQLStatement, error) {
	return buildDelete(d, table, condition, args...)
}

func (d PostgreSQLDialect) AcquireLockSQL(lockName string, timeoutSeconds int) (SQLStatement, error) {
	return SQLStatement{
		Query: "SELECT pg_advisory_lock(hashtext($1))",
		Args:  []any{lockName},
	}, nil
}

func (d PostgreSQLDialect) ReleaseLockSQL(lockName string) (SQLStatement, error) {
	return SQLStatement{
		Query: "SELECT pg_advisory_unlock(hashtext($1))",
		Args:  []any{lockName},
	}, nil
}
```

- [ ] **Step 3: Verify the file compiles**

Run: `go build ./migrate/...`
Expected: Success (no compile errors)

- [ ] **Step 4: Commit**

```bash
git add migrate/postgres.go
git commit -m "feat: 添加 PostgreSQL 方言实现"
```

---

### Task 2: PostgreSQL Unit Tests

**Covers:** [S8]

**Files:**
- Create: `migrate/postgres_test.go`

- [ ] **Step 1: Write unit tests for PostgreSQL dialect**

Create `migrate/postgres_test.go`:

```go
package migrate

import (
	"errors"
	"reflect"
	"testing"
)

func TestPostgreSQLCreateTableSQL(t *testing.T) {
	m := NewMigrationContext(nil, PostgreSQLDialect{})
	sql, err := PostgreSQLDialect{}.CreateTable("article", Columns().
		Add("id", m.BigPrimaryKey()).
		Add("title", m.String(128).NotNull()).
		Add("metadata", m.Json().Null()).
		Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")),
		"",
	)
	if err != nil {
		t.Fatalf("CreateTable returned error: %v", err)
	}
	want := `CREATE TABLE "article" ("id" BIGSERIAL PRIMARY KEY, "title" VARCHAR(128) NOT NULL, "metadata" JSONB NULL, "created_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`
	if sql != want {
		t.Fatalf("CreateTable SQL:\n got: %s\nwant: %s", sql, want)
	}
}

func TestPostgreSQLPrimaryKey(t *testing.T) {
	m := NewMigrationContext(nil, PostgreSQLDialect{})
	col, err := PostgreSQLDialect{}.BuildColumn(m.PrimaryKey())
	if err != nil {
		t.Fatalf("BuildColumn error: %v", err)
	}
	if col != "SERIAL PRIMARY KEY" {
		t.Fatalf("got %q, want %q", col, "SERIAL PRIMARY KEY")
	}
}

func TestPostgreSQLMetadataSQL(t *testing.T) {
	d := PostgreSQLDialect{}

	tableSQL := d.TableExistsSQL("article")
	wantQuery := `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1`
	if tableSQL.Query != wantQuery {
		t.Fatalf("TableExistsSQL:\n got: %s\nwant: %s", tableSQL.Query, wantQuery)
	}
	if !reflect.DeepEqual(tableSQL.Args, []any{"article"}) {
		t.Fatalf("TableExistsSQL args = %#v", tableSQL.Args)
	}

	colSQL := d.ColumnExistsSQL("article", "title")
	wantColQuery := `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2`
	if colSQL.Query != wantColQuery {
		t.Fatalf("ColumnExistsSQL:\n got: %s\nwant: %s", colSQL.Query, wantColQuery)
	}

	idxSQL := d.IndexExistsSQL("article", "idx_title")
	wantIdxQuery := `SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1 AND indexname = $2`
	if idxSQL.Query != wantIdxQuery {
		t.Fatalf("IndexExistsSQL:\n got: %s\nwant: %s", idxSQL.Query, wantIdxQuery)
	}
}

func TestPostgreSQLDML(t *testing.T) {
	d := PostgreSQLDialect{}
	stmt, err := d.Insert("article", Row{
		"title":      "Hello",
		"created_at": Expr("CURRENT_TIMESTAMP"),
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}
	wantSQL := `INSERT INTO "article" ("created_at", "title") VALUES (CURRENT_TIMESTAMP, $1)`
	if stmt.Query != wantSQL {
		t.Fatalf("Insert SQL:\n got: %s\nwant: %s", stmt.Query, wantSQL)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"Hello"}) {
		t.Fatalf("Insert args = %#v", stmt.Args)
	}
}

func TestPostgreSQLDDLMethods(t *testing.T) {
	d := PostgreSQLDialect{}

	sql, err := d.DropTable("article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `DROP TABLE IF EXISTS "article"` {
		t.Fatalf("DropTable: %s", sql)
	}

	sql, err = d.RenameTable("old", "new")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER TABLE "old" RENAME TO "new"` {
		t.Fatalf("RenameTable: %s", sql)
	}

	sql, err = d.TruncateTable("article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `TRUNCATE TABLE "article"` {
		t.Fatalf("TruncateTable: %s", sql)
	}

	sql, err = d.DropColumn("article", "title")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER TABLE "article" DROP COLUMN "title"` {
		t.Fatalf("DropColumn: %s", sql)
	}

	sql, err = d.RenameColumn("article", "old_col", "new_col")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER TABLE "article" RENAME COLUMN "old_col" TO "new_col"` {
		t.Fatalf("RenameColumn: %s", sql)
	}

	sql, err = d.DropIndex("idx_title", "article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `DROP INDEX IF EXISTS "idx_title"` {
		t.Fatalf("DropIndex: %s", sql)
	}

	sql, err = d.DropPrimaryKey("pk_article", "article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER TABLE "article" DROP CONSTRAINT "pk_article"` {
		t.Fatalf("DropPrimaryKey: %s", sql)
	}

	sql, err = d.DropForeignKey("fk_user", "article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER TABLE "article" DROP CONSTRAINT "fk_user"` {
		t.Fatalf("DropForeignKey: %s", sql)
	}
}

func TestPostgreSQLComments(t *testing.T) {
	d := PostgreSQLDialect{}

	sql, err := d.AddCommentOnColumn("article", "title", "The title")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `COMMENT ON COLUMN "article"."title" IS 'The title'` {
		t.Fatalf("AddCommentOnColumn: %s", sql)
	}

	sql, err = d.DropCommentFromColumn("article", "title")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `COMMENT ON COLUMN "article"."title" IS NULL` {
		t.Fatalf("DropCommentFromColumn: %s", sql)
	}

	sql, err = d.AddCommentOnTable("article", "Articles table")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `COMMENT ON TABLE "article" IS 'Articles table'` {
		t.Fatalf("AddCommentOnTable: %s", sql)
	}

	sql, err = d.DropCommentFromTable("article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `COMMENT ON TABLE "article" IS NULL` {
		t.Fatalf("DropCommentFromTable: %s", sql)
	}
}

func TestPostgreSQLLocks(t *testing.T) {
	d := PostgreSQLDialect{}

	stmt, err := d.AcquireLockSQL("migration", 30)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Query != `SELECT pg_advisory_lock(hashtext($1))` {
		t.Fatalf("AcquireLockSQL: %s", stmt.Query)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"migration"}) {
		t.Fatalf("AcquireLockSQL args = %#v", stmt.Args)
	}

	stmt, err = d.ReleaseLockSQL("migration")
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Query != `SELECT pg_advisory_unlock(hashtext($1))` {
		t.Fatalf("ReleaseLockSQL: %s", stmt.Query)
	}
}

func TestPostgreSQLUnsupportedOperations(t *testing.T) {
	m := NewMigrationContext(nil, PostgreSQLDialect{})
	_, err := PostgreSQLDialect{}.BuildColumn(m.Integer().Unsigned())
	var unsupported *UnsupportedOperationError
	if !errors.As(err, &unsupported) {
		t.Fatalf("Unsigned error = %v, want UnsupportedOperationError", err)
	}
	if unsupported.Dialect != "postgres" || unsupported.Operation != "UNSIGNED" {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func TestPostgreSQLTypeMappings(t *testing.T) {
	m := NewMigrationContext(nil, PostgreSQLDialect{})
	d := PostgreSQLDialect{}

	tests := []struct {
		name string
		col  *ColumnBuilder
		want string
	}{
		{"boolean", m.Boolean(), "BOOLEAN"},
		{"uuid", m.UUID(), "UUID"},
		{"json", m.Json(), "JSONB"},
		{"text", m.Text(), "TEXT"},
		{"binary", m.Binary(100), "BYTEA"},
		{"float", m.Float(), "REAL"},
		{"double", m.Double(), "DOUBLE PRECISION"},
		{"date", m.Date(), "DATE"},
		{"time", m.Time(), "TIME"},
		{"timestamp", m.Timestamp(), "TIMESTAMP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.BuildColumn(tt.col)
			if err != nil {
				t.Fatalf("BuildColumn error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./migrate/ -run TestPostgreSQL -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add migrate/postgres_test.go
git commit -m "test: 添加 PostgreSQL 方言单元测试"
```

---

### Task 3: SQL Server Dialect — Core Structure & BuildColumn

**Covers:** [S5]

**Files:**
- Create: `migrate/sqlserver.go`

- [ ] **Step 1: Write the SQL Server dialect file with all 38 methods**

Create `migrate/sqlserver.go`:

```go
package migrate

import (
	"fmt"
	"strings"
)

type SQLServerDialect struct{}

func (d SQLServerDialect) Name() string { return "sqlserver" }

func (d SQLServerDialect) QuoteTable(name string) string { return sqlserverQuoteName(name) }

func (d SQLServerDialect) QuoteColumn(name string) string { return sqlserverQuoteName(name) }

func (d SQLServerDialect) QuoteIndexColumn(name string) string {
	if strings.ContainsAny(name, "() ") || strings.Contains(name, "->") {
		return name
	}
	return sqlserverQuoteName(name)
}

func (d SQLServerDialect) Placeholder(index int) string { return fmt.Sprintf("@p%d", index) }

func sqlserverQuoteName(name string) string {
	if name == "*" {
		return name
	}
	parts := strings.Split(name, ".")
	for i, part := range parts {
		parts[i] = "[" + strings.ReplaceAll(part, "]", "]]") + "]"
	}
	return strings.Join(parts, ".")
}

func (d SQLServerDialect) BuildColumn(c *ColumnBuilder) (string, error) {
	if c == nil {
		return "", fmt.Errorf("migrate: column builder is nil")
	}
	if c.unsigned {
		return "", unsupported(d.Name(), "UNSIGNED")
	}
	if c.charset != "" {
		return "", unsupported(d.Name(), "CHARACTER SET")
	}
	if c.collation != "" {
		return "", unsupported(d.Name(), "COLLATE")
	}
	if c.after != "" {
		return "", unsupported(d.Name(), "AFTER")
	}
	if c.first {
		return "", unsupported(d.Name(), "FIRST")
	}
	if c.generatedAs != "" {
		return "", unsupported(d.Name(), "GENERATED COLUMN")
	}

	if c.primaryKey && c.autoIncrement {
		parts := []string{mssqlIdentityType(c)}
		if c.nullSet && !c.nullable {
			parts = append(parts, "NOT NULL")
		}
		if c.appendSQL != "" {
			parts = append(parts, c.appendSQL)
		}
		return strings.Join(parts, " "), nil
	}

	parts := []string{mssqlType(c)}
	if c.nullSet {
		if c.nullable {
			parts = append(parts, "NULL")
		} else {
			parts = append(parts, "NOT NULL")
		}
	}
	if c.primaryKey {
		parts = append(parts, "PRIMARY KEY")
	}
	if c.unique {
		parts = append(parts, "UNIQUE")
	}
	if c.defaultExpr != "" {
		parts = append(parts, "DEFAULT", c.defaultExpr)
	}
	if c.defaultSet {
		parts = append(parts, "DEFAULT", sqlLiteral(c.defaultValue))
	}
	if c.check != "" {
		parts = append(parts, "CHECK ("+c.check+")")
	}
	if c.appendSQL != "" {
		parts = append(parts, c.appendSQL)
	}
	return strings.Join(parts, " "), nil
}

func mssqlIdentityType(c *ColumnBuilder) string {
	switch c.typeName {
	case "integer":
		return "INT IDENTITY(1,1) PRIMARY KEY"
	case "bigInteger":
		return "BIGINT IDENTITY(1,1) PRIMARY KEY"
	default:
		return "INT IDENTITY(1,1) PRIMARY KEY"
	}
}

func mssqlType(c *ColumnBuilder) string {
	switch c.typeName {
	case "tinyInteger":
		return "TINYINT"
	case "smallInteger":
		return "SMALLINT"
	case "integer":
		return "INT"
	case "bigInteger":
		return "BIGINT"
	case "string":
		return sizedType("NVARCHAR", c.size, 255)
	case "char":
		return sizedType("NCHAR", c.size)
	case "text", "tinyText", "mediumText", "longText":
		return "NVARCHAR(MAX)"
	case "binary":
		return sizedType("VARBINARY", c.size)
	case "tinyBlob", "mediumBlob", "longBlob":
		return "VARBINARY(MAX)"
	case "boolean":
		return "BIT"
	case "float":
		return sizedType("FLOAT", c.size)
	case "double":
		return "DOUBLE PRECISION"
	case "decimal", "money":
		return sizedType("DECIMAL", c.size)
	case "date":
		return "DATE"
	case "dateTime", "timestamp":
		return precisionType("DATETIME2", c.size)
	case "time":
		return precisionType("TIME", c.size)
	case "json":
		return "NVARCHAR(MAX)"
	case "uuid":
		return "UNIQUEIDENTIFIER"
	case "enum", "set":
		return sizedType("NVARCHAR", c.size, 255)
	default:
		return strings.ToUpper(c.typeName)
	}
}
```

- [ ] **Step 2: Add remaining DDL, DML, and lock methods to sqlserver.go**

Append these methods to `migrate/sqlserver.go`:

```go
func (d SQLServerDialect) TableExistsSQL(table string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM sys.tables WHERE SCHEMA_NAME(schema_id) = SCHEMA_NAME() AND name = @p1",
		Args:  []any{table},
	}
}

func (d SQLServerDialect) ColumnExistsSQL(table, column string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM sys.columns c JOIN sys.tables t ON c.object_id = t.object_id WHERE SCHEMA_NAME(t.schema_id) = SCHEMA_NAME() AND t.name = @p1 AND c.name = @p2",
		Args:  []any{table, column},
	}
}

func (d SQLServerDialect) IndexExistsSQL(table, index string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM sys.indexes i JOIN sys.tables t ON i.object_id = t.object_id WHERE SCHEMA_NAME(t.schema_id) = SCHEMA_NAME() AND t.name = @p1 AND i.name = @p2",
		Args:  []any{table, index},
	}
}

func (d SQLServerDialect) ForeignKeyExistsSQL(table, name string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM sys.foreign_keys WHERE SCHEMA_NAME(schema_id) = SCHEMA_NAME() AND OBJECT_NAME(parent_object_id) = @p1 AND name = @p2",
		Args:  []any{table, name},
	}
}

func (d SQLServerDialect) ConstraintExistsSQL(table, name string) SQLStatement {
	return SQLStatement{
		Query: "SELECT COUNT(*) FROM sys.check_constraints cc JOIN sys.tables t ON cc.parent_object_id = t.object_id WHERE SCHEMA_NAME(t.schema_id) = SCHEMA_NAME() AND t.name = @p1 AND cc.name = @p2",
		Args:  []any{table, name},
	}
}

func (d SQLServerDialect) BuildRowExistsSQL(table string, condition string) string {
	query := "SELECT CASE WHEN EXISTS(SELECT 1 FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return query + ") THEN 1 ELSE 0 END"
}

func (d SQLServerDialect) BuildCountRowsSQL(table string, condition string) string {
	query := "SELECT COUNT(*) FROM " + d.QuoteTable(table)
	if strings.TrimSpace(condition) != "" {
		query += " WHERE " + condition
	}
	return query
}

func (d SQLServerDialect) CreateTable(table string, columns *ColumnList, options string) (string, error) {
	if strings.TrimSpace(options) != "" {
		return "", unsupported(d.Name(), "TABLE OPTIONS")
	}
	defs := make([]string, 0, len(columns.Items()))
	for _, item := range columns.Items() {
		columnSQL, err := d.BuildColumn(item.Column)
		if err != nil {
			return "", err
		}
		defs = append(defs, d.QuoteColumn(item.Name)+" "+columnSQL)
	}
	return fmt.Sprintf("CREATE TABLE %s (%s)", d.QuoteTable(table), strings.Join(defs, ", ")), nil
}

func (d SQLServerDialect) DropTable(table string) (string, error) {
	return "DROP TABLE IF EXISTS " + d.QuoteTable(table), nil
}

func (d SQLServerDialect) RenameTable(oldName, newName string) (string, error) {
	return fmt.Sprintf("EXEC sp_rename %s, %s", sqlLiteral(oldName), sqlLiteral(newName)), nil
}

func (d SQLServerDialect) TruncateTable(table string) (string, error) {
	return "TRUNCATE TABLE " + d.QuoteTable(table), nil
}

func (d SQLServerDialect) AddColumn(table, column string, builder *ColumnBuilder) (string, error) {
	columnSQL, err := d.BuildColumn(builder)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ALTER TABLE %s ADD %s %s", d.QuoteTable(table), d.QuoteColumn(column), columnSQL), nil
}

func (d SQLServerDialect) AlterColumn(table, column string, builder *ColumnBuilder) (string, error) {
	columnSQL, err := d.BuildColumn(builder)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s", d.QuoteTable(table), d.QuoteColumn(column), columnSQL), nil
}

func (d SQLServerDialect) DropColumn(table, column string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", d.QuoteTable(table), d.QuoteColumn(column)), nil
}

func (d SQLServerDialect) RenameColumn(table, oldName, newName string) (string, error) {
	return fmt.Sprintf("EXEC sp_rename %s, %s, %s", sqlLiteral(table+"."+oldName), sqlLiteral(newName), sqlLiteral("COLUMN")), nil
}

func (d SQLServerDialect) CreateIndex(name, table string, columns []string, unique bool) (string, error) {
	prefix := "CREATE INDEX"
	if unique {
		prefix = "CREATE UNIQUE INDEX"
	}
	return fmt.Sprintf("%s %s ON %s (%s)", prefix, d.QuoteTable(name), d.QuoteTable(table), columnList(d, columns)), nil
}

func (d SQLServerDialect) DropIndex(name, table string) (string, error) {
	return fmt.Sprintf("DROP INDEX %s ON %s", d.QuoteTable(name), d.QuoteTable(table)), nil
}

func (d SQLServerDialect) AddPrimaryKey(name, table string, columns []string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s)", d.QuoteTable(table), d.QuoteTable(name), columnList(d, columns)), nil
}

func (d SQLServerDialect) DropPrimaryKey(name, table string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", d.QuoteTable(table), d.QuoteTable(name)), nil
}

func (d SQLServerDialect) AddForeignKey(name, table string, columns []string, refTable string, refColumns []string, onDelete ForeignKeyAction, onUpdate ForeignKeyAction) (string, error) {
	query := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)", d.QuoteTable(table), d.QuoteTable(name), columnList(d, columns), d.QuoteTable(refTable), columnList(d, refColumns))
	if onDelete != "" {
		query += " ON DELETE " + string(onDelete)
	}
	if onUpdate != "" {
		query += " ON UPDATE " + string(onUpdate)
	}
	return query, nil
}

func (d SQLServerDialect) DropForeignKey(name, table string) (string, error) {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", d.QuoteTable(table), d.QuoteTable(name)), nil
}

func (d SQLServerDialect) AddCommentOnColumn(table, column, comment string) (string, error) {
	return fmt.Sprintf("EXEC sp_addextendedproperty 'MS_Description', %s, 'SCHEMA', SCHEMA_NAME(), 'TABLE', %s, 'COLUMN', %s", sqlLiteral(comment), sqlLiteral(table), sqlLiteral(column)), nil
}

func (d SQLServerDialect) DropCommentFromColumn(table, column string) (string, error) {
	return fmt.Sprintf("EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', SCHEMA_NAME(), 'TABLE', %s, 'COLUMN', %s", sqlLiteral(table), sqlLiteral(column)), nil
}

func (d SQLServerDialect) AddCommentOnTable(table, comment string) (string, error) {
	return fmt.Sprintf("EXEC sp_addextendedproperty 'MS_Description', %s, 'SCHEMA', SCHEMA_NAME(), 'TABLE', %s", sqlLiteral(comment), sqlLiteral(table)), nil
}

func (d SQLServerDialect) DropCommentFromTable(table string) (string, error) {
	return fmt.Sprintf("EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', SCHEMA_NAME(), 'TABLE', %s", sqlLiteral(table)), nil
}

func (d SQLServerDialect) Insert(table string, row Row) (SQLStatement, error) {
	return buildInsert(d, table, row)
}

func (d SQLServerDialect) BatchInsert(table string, columns []string, rows [][]any) (SQLStatement, error) {
	return buildBatchInsert(d, table, columns, rows)
}

func (d SQLServerDialect) Update(table string, row Row, condition string, args ...any) (SQLStatement, error) {
	return buildUpdate(d, table, row, condition, args...)
}

func (d SQLServerDialect) Delete(table string, condition string, args ...any) (SQLStatement, error) {
	return buildDelete(d, table, condition, args...)
}

func (d SQLServerDialect) AcquireLockSQL(lockName string, timeoutSeconds int) (SQLStatement, error) {
	return SQLStatement{
		Query: "DECLARE @result INT; EXEC @result = sp_getapplock @Resource = @p1, @LockMode = 'Exclusive', @LockTimeout = @p2; SELECT @result",
		Args:  []any{lockName, timeoutSeconds},
	}, nil
}

func (d SQLServerDialect) ReleaseLockSQL(lockName string) (SQLStatement, error) {
	return SQLStatement{
		Query: "DECLARE @result INT; EXEC @result = sp_releaseapplock @Resource = @p1; SELECT @result",
		Args:  []any{lockName},
	}, nil
}
```

- [ ] **Step 3: Verify the file compiles**

Run: `go build ./migrate/...`
Expected: Success

- [ ] **Step 4: Commit**

```bash
git add migrate/sqlserver.go
git commit -m "feat: 添加 SQL Server 方言实现"
```

---

### Task 4: SQL Server Unit Tests

**Covers:** [S8]

**Files:**
- Create: `migrate/sqlserver_test.go`

- [ ] **Step 1: Write unit tests for SQL Server dialect**

Create `migrate/sqlserver_test.go`:

```go
package migrate

import (
	"errors"
	"reflect"
	"testing"
)

func TestSQLServerCreateTableSQL(t *testing.T) {
	m := NewMigrationContext(nil, SQLServerDialect{})
	sql, err := SQLServerDialect{}.CreateTable("article", Columns().
		Add("id", m.BigPrimaryKey()).
		Add("title", m.String(128).NotNull()).
		Add("metadata", m.Json().Null()),
		"",
	)
	if err != nil {
		t.Fatalf("CreateTable returned error: %v", err)
	}
	want := `CREATE TABLE [article] ([id] BIGINT IDENTITY(1,1) PRIMARY KEY, [title] NVARCHAR(128) NOT NULL, [metadata] NVARCHAR(MAX) NULL)`
	if sql != want {
		t.Fatalf("CreateTable SQL:\n got: %s\nwant: %s", sql, want)
	}
}

func TestSQLServerPrimaryKey(t *testing.T) {
	m := NewMigrationContext(nil, SQLServerDialect{})
	col, err := SQLServerDialect{}.BuildColumn(m.PrimaryKey())
	if err != nil {
		t.Fatalf("BuildColumn error: %v", err)
	}
	if col != "INT IDENTITY(1,1) PRIMARY KEY" {
		t.Fatalf("got %q, want %q", col, "INT IDENTITY(1,1) PRIMARY KEY")
	}
}

func TestSQLServerMetadataSQL(t *testing.T) {
	d := SQLServerDialect{}

	tableSQL := d.TableExistsSQL("article")
	if tableSQL.Query != "SELECT COUNT(*) FROM sys.tables WHERE SCHEMA_NAME(schema_id) = SCHEMA_NAME() AND name = @p1" {
		t.Fatalf("TableExistsSQL: %s", tableSQL.Query)
	}
	if !reflect.DeepEqual(tableSQL.Args, []any{"article"}) {
		t.Fatalf("TableExistsSQL args = %#v", tableSQL.Args)
	}
}

func TestSQLServerDML(t *testing.T) {
	d := SQLServerDialect{}
	stmt, err := d.Insert("article", Row{
		"title":      "Hello",
		"created_at": Expr("GETDATE()"),
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}
	wantSQL := `INSERT INTO [article] ([created_at], [title]) VALUES (GETDATE(), @p1)`
	if stmt.Query != wantSQL {
		t.Fatalf("Insert SQL:\n got: %s\nwant: %s", stmt.Query, wantSQL)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"Hello"}) {
		t.Fatalf("Insert args = %#v", stmt.Args)
	}
}

func TestSQLServerDDLMethods(t *testing.T) {
	d := SQLServerDialect{}

	sql, err := d.DropTable("article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `DROP TABLE IF EXISTS [article]` {
		t.Fatalf("DropTable: %s", sql)
	}

	sql, err = d.RenameTable("old", "new")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `EXEC sp_rename 'old', 'new'` {
		t.Fatalf("RenameTable: %s", sql)
	}

	sql, err = d.TruncateTable("article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `TRUNCATE TABLE [article]` {
		t.Fatalf("TruncateTable: %s", sql)
	}

	sql, err = d.DropColumn("article", "title")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER TABLE [article] DROP COLUMN [title]` {
		t.Fatalf("DropColumn: %s", sql)
	}

	sql, err = d.DropIndex("idx_title", "article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `DROP INDEX [idx_title] ON [article]` {
		t.Fatalf("DropIndex: %s", sql)
	}

	sql, err = d.DropPrimaryKey("pk_article", "article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER TABLE [article] DROP CONSTRAINT [pk_article]` {
		t.Fatalf("DropPrimaryKey: %s", sql)
	}

	sql, err = d.DropForeignKey("fk_user", "article")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `ALTER TABLE [article] DROP CONSTRAINT [fk_user]` {
		t.Fatalf("DropForeignKey: %s", sql)
	}
}

func TestSQLServerComments(t *testing.T) {
	d := SQLServerDialect{}

	sql, err := d.AddCommentOnColumn("article", "title", "The title")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `EXEC sp_addextendedproperty 'MS_Description', 'The title', 'SCHEMA', SCHEMA_NAME(), 'TABLE', 'article', 'COLUMN', 'title'` {
		t.Fatalf("AddCommentOnColumn: %s", sql)
	}

	sql, err = d.DropCommentFromColumn("article", "title")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', SCHEMA_NAME(), 'TABLE', 'article', 'COLUMN', 'title'` {
		t.Fatalf("DropCommentFromColumn: %s", sql)
	}

	sql, err = d.AddCommentOnTable("article", "Articles table")
	if err != nil {
		t.Fatal(err)
	}
	if sql != `EXEC sp_addextendedproperty 'MS_Description', 'Articles table', 'SCHEMA', SCHEMA_NAME(), 'TABLE', 'article'` {
		t.Fatalf("AddCommentOnTable: %s", sql)
	}
}

func TestSQLServerLocks(t *testing.T) {
	d := SQLServerDialect{}

	stmt, err := d.AcquireLockSQL("migration", 30)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stmt.Query, "sp_getapplock") {
		t.Fatalf("AcquireLockSQL should contain sp_getapplock: %s", stmt.Query)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"migration", 30}) {
		t.Fatalf("AcquireLockSQL args = %#v", stmt.Args)
	}

	stmt, err = d.ReleaseLockSQL("migration")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stmt.Query, "sp_releaseapplock") {
		t.Fatalf("ReleaseLockSQL should contain sp_releaseapplock: %s", stmt.Query)
	}
}

func TestSQLServerUnsupportedOperations(t *testing.T) {
	m := NewMigrationContext(nil, SQLServerDialect{})
	_, err := SQLServerDialect{}.BuildColumn(m.Integer().Unsigned())
	var unsupported *UnsupportedOperationError
	if !errors.As(err, &unsupported) {
		t.Fatalf("Unsigned error = %v, want UnsupportedOperationError", err)
	}
	if unsupported.Dialect != "sqlserver" || unsupported.Operation != "UNSIGNED" {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func TestSQLServerTypeMappings(t *testing.T) {
	m := NewMigrationContext(nil, SQLServerDialect{})
	d := SQLServerDialect{}

	tests := []struct {
		name string
		col  *ColumnBuilder
		want string
	}{
		{"boolean", m.Boolean(), "BIT"},
		{"uuid", m.UUID(), "UNIQUEIDENTIFIER"},
		{"json", m.Json(), "NVARCHAR(MAX)"},
		{"text", m.Text(), "NVARCHAR(MAX)"},
		{"binary", m.Binary(100), "VARBINARY(100)"},
		{"date", m.Date(), "DATE"},
		{"time", m.Time(), "TIME"},
		{"timestamp", m.Timestamp(), "DATETIME2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.BuildColumn(tt.col)
			if err != nil {
				t.Fatalf("BuildColumn error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
```

Note: The `TestSQLServerLocks` and `TestSQLServerTypeMappings` tests use `strings.Contains`, so add `"strings"` to the import list.

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./migrate/ -run TestSQLServer -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add migrate/sqlserver_test.go
git commit -m "test: 添加 SQL Server 方言单元测试"
```

---

### Task 5: CLI Integration

**Covers:** [S7]

**Files:**
- Modify: `cmd/migrate/main.go`

- [ ] **Step 1: Add PostgreSQL and SQL Server dialect options to resolveDBConfig**

Edit `cmd/migrate/main.go`. First add the driver imports:

```go
import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	_ "modernc.org/sqlite"
	// ... existing imports
)
```

Then update `resolveDBConfig`:

```go
func resolveDBConfig(name string) (dbConfig, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "mysql":
		return dbConfig{driverName: "mysql", dialect: migrate.MySQLDialect{}}, nil
	case "sqlite", "sqlite3":
		return dbConfig{driverName: "sqlite", dialect: migrate.SQLiteDialect{}}, nil
	case "postgres", "postgresql":
		return dbConfig{driverName: "postgres", dialect: migrate.PostgreSQLDialect{}}, nil
	case "sqlserver", "mssql":
		return dbConfig{driverName: "sqlserver", dialect: migrate.SQLServerDialect{}}, nil
	default:
		return dbConfig{}, fmt.Errorf("unsupported DB_DIALECT %q; supported values: mysql, sqlite, postgres, sqlserver", name)
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/migrate/...`
Expected: May fail due to missing driver dependencies — this is expected. The code is correct; drivers will be installed with `go mod tidy`.

Run: `go get github.com/lib/pq github.com/microsoft/go-mssqldb && go mod tidy`
Then: `go build ./cmd/migrate/...`
Expected: Success

- [ ] **Step 3: Run all existing tests**

Run: `go test ./...`
Expected: All existing tests pass. New dialect tests pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/migrate/main.go go.mod go.sum
git commit -m "feat: CLI 支持 PostgreSQL 和 SQL Server 方言"
```

---

### Task 6: Integration Test Stubs

**Covers:** [S8]

**Files:**
- Create: `migrate/postgres_integration_test.go`
- Create: `migrate/sqlserver_integration_test.go`

- [ ] **Step 1: Create PostgreSQL integration test file**

Create `migrate/postgres_integration_test.go`:

```go
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func pgDSN() string { return os.Getenv("POSTGRES_TEST_DSN") }

func TestPGIntegrationCreateTableAndCRUD(t *testing.T) {
	dsn := pgDSN()
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	table := fmt.Sprintf("test_%d", time.Now().UnixNano())
	m := NewMigrationContext(db, PostgreSQLDialect{})

	err = m.Schema().
		CreateTableIfNotExists(ctx, table, Columns().
			Add("id", m.BigPrimaryKey()).
			Add("name", m.String(100).NotNull()).
			Add("data", m.Json().Null()).
			Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")),
			"",
		).Exec(ctx)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = m.Schema().Insert(table, Row{
		"name": "test",
	}).Exec(ctx)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	exists, err := m.RowExists(ctx, table, "name = $1", "test")
	if err != nil {
		t.Fatalf("row exists: %v", err)
	}
	if !exists {
		t.Fatal("expected row to exist")
	}

	count, err := m.CountRows(ctx, table, "")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}

	_, err = m.Schema().DropTableIfExists(ctx, table).Exec(ctx)
	if err != nil {
		t.Fatalf("drop table: %v", err)
	}
}
```

- [ ] **Step 2: Create SQL Server integration test file**

Create `migrate/sqlserver_integration_test.go`:

```go
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/microsoft/go-mssqldb"
)

func mssqlDSN() string { return os.Getenv("SQLSERVER_TEST_DSN") }

func TestMSSQLIntegrationCreateTableAndCRUD(t *testing.T) {
	dsn := mssqlDSN()
	if dsn == "" {
		t.Skip("SQLSERVER_TEST_DSN not set")
	}

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	table := fmt.Sprintf("test_%d", time.Now().UnixNano())
	m := NewMigrationContext(db, SQLServerDialect{})

	err = m.Schema().
		CreateTableIfNotExists(ctx, table, Columns().
			Add("id", m.BigPrimaryKey()).
			Add("name", m.String(100).NotNull()).
			Add("data", m.Json().Null()).
			Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("GETDATE()")),
			"",
		).Exec(ctx)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = m.Schema().Insert(table, Row{
		"name": "test",
	}).Exec(ctx)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	exists, err := m.RowExists(ctx, table, "name = @p1", "test")
	if err != nil {
		t.Fatalf("row exists: %v", err)
	}
	if !exists {
		t.Fatal("expected row to exist")
	}

	count, err := m.CountRows(ctx, table, "")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}

	_, err = m.Schema().DropTableIfExists(ctx, table).Exec(ctx)
	if err != nil {
		t.Fatalf("drop table: %v", err)
	}
}
```

- [ ] **Step 3: Run all tests (integration tests will skip without DSN)**

Run: `go test ./...`
Expected: All tests pass. Integration tests skip with "POSTGRES_TEST_DSN not set" / "SQLSERVER_TEST_DSN not set".

- [ ] **Step 4: Commit**

```bash
git add migrate/postgres_integration_test.go migrate/sqlserver_integration_test.go
git commit -m "test: 添加 PostgreSQL 和 SQL Server 集成测试"
```

---

### Task 7: Final Verification

**Covers:** [S8]

**Files:** None (verification only)

- [ ] **Step 1: Run gofmt on all changed files**

Run: `gofmt -w migrate/postgres.go migrate/sqlserver.go migrate/postgres_test.go migrate/sqlserver_test.go migrate/postgres_integration_test.go migrate/sqlserver_integration_test.go cmd/migrate/main.go`
Expected: No output (files formatted correctly)

- [ ] **Step 2: Run full test suite**

Run: `go test ./...`
Expected: All tests pass

- [ ] **Step 3: Run go vet**

Run: `go vet ./...`
Expected: No issues
