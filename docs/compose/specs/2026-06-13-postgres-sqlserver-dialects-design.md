# PostgreSQL & SQL Server Dialect Design

## [S1] Problem

The library currently supports MySQL and SQLite dialects. Users need PostgreSQL and SQL Server support for multi-database scenarios.

## [S2] Solution Overview

Add two new dialect implementations (`PostgreSQLDialect` and `SQLServerDialect`) as independent files, following the existing pattern established by `mysql.go` and `sqlite.go`. Extract additional shared helper functions to `dialect.go` where beneficial.

## [S3] File Structure

```
migrate/
├── postgres.go                     # PostgreSQL dialect (new)
├── sqlserver.go                    # SQL Server dialect (new)
├── dialect.go                      # Add shared DDL helpers
├── postgres_test.go                # PG unit tests (new)
├── sqlserver_test.go               # MSSQL unit tests (new)
├── postgres_integration_test.go    # PG integration tests (new, POSTGRES_TEST_DSN)
├── sqlserver_integration_test.go   # MSSQL integration tests (new, SQLSERVER_TEST_DSN)
```

## [S4] PostgreSQL Dialect

### Identifier Quoting & Placeholders

- **Quote**: double-quote `"name"`
- **Placeholder**: positional `$1, $2, $3, ...` (Placeholder(index) returns `$N`)
- **QuoteTable/QuoteColumn**: reuse shared `quoteName()` with `"` as delimiter

### Type Mapping (typeName → PostgreSQL type)

| typeName | PostgreSQL type | Notes |
|----------|----------------|-------|
| tinyInteger | SMALLINT | PG has no TINYINT |
| smallInteger | SMALLINT | |
| integer | INTEGER / SERIAL | SERIAL when primaryKey+autoIncrement |
| bigInteger | BIGINT / BIGSERIAL | BIGSERIAL when primaryKey+autoIncrement |
| string | VARCHAR(n) | default 255 |
| char | CHAR(n) | |
| text | TEXT | |
| tinyText | TEXT | PG has no TINYTEXT |
| mediumText | TEXT | |
| longText | TEXT | |
| binary | BYTEA | |
| tinyBlob | BYTEA | |
| mediumBlob | BYTEA | |
| longBlob | BYTEA | |
| boolean | BOOLEAN | native |
| float | REAL | |
| double | DOUBLE PRECISION | |
| decimal | DECIMAL(p,s) | |
| money | MONEY | PG has native MONEY type |
| date | DATE | |
| dateTime | TIMESTAMP(p) | |
| time | TIME(p) | |
| timestamp | TIMESTAMP(p) | |
| json | JSONB | prefer JSONB over JSON |
| uuid | UUID | native |
| enum | unsupported | PG uses CREATE TYPE + CHECK |
| set | unsupported | |

### SERIAL/BIGSERIAL Handling

When `primaryKey == true && autoIncrement == true`:
- `integer` → `SERIAL PRIMARY KEY`
- `bigInteger` → `BIGSERIAL PRIMARY KEY`

These combinations bypass normal type rendering.

### Unsupported ColumnBuilder Features

In `BuildColumn()`, return `UnsupportedOperationError` for:
- `charset` — PG uses database-level encoding
- `collation` — per-column collation not commonly used
- `after` — PG doesn't support column ordering
- `first` — PG doesn't support column ordering
- `unsigned` — PG has no unsigned types

### BuildColumn Output Order

1. Type name (or SERIAL/BIGSERIAL)
2. NOT NULL / NULL (omit for SERIAL — it implies NOT NULL)
3. DEFAULT expression/value
4. CHECK constraint
5. UNIQUE

Note: PRIMARY KEY is handled via SERIAL type; for non-autoincrement PK, append PRIMARY KEY.

### Metadata Queries

- **TableExistsSQL**: `SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1`
- **ColumnExistsSQL**: `SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2`
- **IndexExistsSQL**: `SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1 AND indexname = $2`
- **ForeignKeyExistsSQL**: `SELECT 1 FROM information_schema.table_constraints WHERE table_schema = current_schema() AND table_name = $1 AND constraint_name = $2 AND constraint_type = 'FOREIGN KEY'`
- **ConstraintExistsSQL**: `SELECT 1 FROM information_schema.table_constraints WHERE table_schema = current_schema() AND table_name = $1 AND constraint_name = $2`

### DDL Methods

| Method | SQL Template |
|--------|-------------|
| CreateTable | `CREATE TABLE [IF NOT EXISTS] "table" (columns) options` |
| DropTable | `DROP TABLE IF EXISTS "table"` |
| RenameTable | `ALTER TABLE "old" RENAME TO "new"` |
| TruncateTable | `TRUNCATE TABLE "table"` |
| AddColumn | `ALTER TABLE "table" ADD COLUMN "col" definition` |
| AlterColumn | `ALTER TABLE "table" ALTER COLUMN "col" TYPE type [USING col::type]` — single statement, type only |
| DropColumn | `ALTER TABLE "table" DROP COLUMN "col"` |
| RenameColumn | `ALTER TABLE "table" RENAME COLUMN "old" TO "new"` |
| CreateIndex | `CREATE [UNIQUE] INDEX [IF NOT EXISTS] "name" ON "table" (cols)` |
| DropIndex | `DROP INDEX IF EXISTS "name"` — note: no `ON table` clause |
| AddPrimaryKey | `ALTER TABLE "table" ADD CONSTRAINT "name" PRIMARY KEY (cols)` |
| DropPrimaryKey | `ALTER TABLE "table" DROP CONSTRAINT "name"` |
| AddForeignKey | `ALTER TABLE "table" ADD CONSTRAINT "name" FOREIGN KEY (cols) REFERENCES "refTable" (refCols) [ON DELETE action] [ON UPDATE action]` |
| DropForeignKey | `ALTER TABLE "table" DROP CONSTRAINT "name"` |
| AddCommentOnColumn | `COMMENT ON COLUMN "table"."col" IS 'comment'` |
| DropCommentFromColumn | `COMMENT ON COLUMN "table"."col" IS NULL` |
| AddCommentOnTable | `COMMENT ON TABLE "table" IS 'comment'` |
| DropCommentFromTable | `COMMENT ON TABLE "table" IS NULL` |

### Lock Methods

- **AcquireLockSQL**: `SELECT pg_advisory_lock(hashtext($1))` — timeout via `set_config('lock_timeout', ...)` or application-level timeout
- **ReleaseLockSQL**: `SELECT pg_advisory_unlock(hashtext($1))`

Note: `pg_advisory_lock` requires bigint, so `hashtext()` converts the string lock name to an integer hash.

### DML Methods

Delegate to shared functions `buildInsert`, `buildBatchInsert`, `buildUpdate`, `buildDelete`.

## [S5] SQL Server Dialect

### Identifier Quoting & Placeholders

- **Quote**: square brackets `[name]`
- **Placeholder**: `@p1, @p2, @p3, ...` — compatible with `github.com/microsoft/go-mssqldb` driver
- **QuoteTable/QuoteColumn**: custom quoting with `[` and `]`, handle `.` as schema separator

Note: `quoteName()` uses backtick/double-quote, so SQL Server needs its own quoting implementation.

### Type Mapping (typeName → SQL Server type)

| typeName | SQL Server type | Notes |
|----------|----------------|-------|
| tinyInteger | TINYINT | |
| smallInteger | SMALLINT | |
| integer | INT / INT IDENTITY(1,1) | IDENTITY when primaryKey+autoIncrement |
| bigInteger | BIGINT / BIGINT IDENTITY(1,1) | IDENTITY when primaryKey+autoIncrement |
| string | NVARCHAR(n) | default 255, uses NVARCHAR for Unicode |
| char | NCHAR(n) | |
| text | NVARCHAR(MAX) | |
| tinyText | NVARCHAR(MAX) | |
| mediumText | NVARCHAR(MAX) | |
| longText | NVARCHAR(MAX) | |
| binary | VARBINARY(n) | |
| tinyBlob | VARBINARY(MAX) | |
| mediumBlob | VARBINARY(MAX) | |
| longBlob | VARBINARY(MAX) | |
| boolean | BIT | |
| float | FLOAT(n) | |
| double | DOUBLE PRECISION | |
| decimal | DECIMAL(p,s) | |
| money | MONEY | |
| date | DATE | |
| dateTime | DATETIME2(p) | DATETIME2 is preferred over DATETIME |
| time | TIME(p) | |
| timestamp | DATETIME2(p) | |
| json | NVARCHAR(MAX) | no native JSON column type |
| uuid | UNIQUEIDENTIFIER | native |
| enum | unsupported | |
| set | unsupported | |

### IDENTITY Handling

When `primaryKey == true && autoIncrement == true`:
- `integer` → `INT IDENTITY(1,1) PRIMARY KEY`
- `bigInteger` → `BIGINT IDENTITY(1,1) PRIMARY KEY`

### Unsupported ColumnBuilder Features

In `BuildColumn()`, return `UnsupportedOperationError` for:
- `charset` — not applicable
- `collation` — column-level collation is complex, skip for now
- `after` — not supported
- `first` — not supported
- `unsigned` — not supported
- `generatedAs` / `stored` / `virtual` — computed columns syntax differs

### BuildColumn Output Order

1. Type name (or INT/BIGINT IDENTITY)
2. NOT NULL / NULL
3. IDENTITY(1,1) — if autoIncrement and not handled by special type
4. PRIMARY KEY
5. DEFAULT expression/value
6. CHECK constraint
7. UNIQUE

### Metadata Queries

- **TableExistsSQL**: `SELECT 1 FROM sys.tables WHERE schema_name(schema_id) = schema_name() AND name = @p1`
- **ColumnExistsSQL**: `SELECT 1 FROM sys.columns c JOIN sys.tables t ON c.object_id = t.object_id WHERE schema_name(t.schema_id) = schema_name() AND t.name = @p1 AND c.name = @p2`
- **IndexExistsSQL**: `SELECT 1 FROM sys.indexes i JOIN sys.tables t ON i.object_id = t.object_id WHERE schema_name(t.schema_id) = schema_name() AND t.name = @p1 AND i.name = @p2`
- **ForeignKeyExistsSQL**: `SELECT 1 FROM sys.foreign_keys WHERE schema_name(schema_id) = schema_name() AND parent_object_id = object_id(@p1) AND name = @p2`
- **ConstraintExistsSQL**: `SELECT 1 FROM sys.check_constraints cc JOIN sys.tables t ON cc.parent_object_id = t.object_id WHERE schema_name(t.schema_id) = schema_name() AND t.name = @p1 AND cc.name = @p2`

### DDL Methods

| Method | SQL Template |
|--------|-------------|
| CreateTable | `CREATE TABLE [table] (columns)` — no table options |
| DropTable | `DROP TABLE IF EXISTS [table]` |
| RenameTable | `sp_rename 'old', 'new'` |
| TruncateTable | `TRUNCATE TABLE [table]` |
| AddColumn | `ALTER TABLE [table] ADD [col] definition` |
| AlterColumn | `ALTER TABLE [table] ALTER COLUMN [col] type [NULL\|NOT NULL]` |
| DropColumn | `ALTER TABLE [table] DROP COLUMN [col]` |
| RenameColumn | `sp_rename 'table.old', 'new', 'COLUMN'` |
| CreateIndex | `CREATE [UNIQUE] INDEX [name] ON [table] (cols)` |
| DropIndex | `DROP INDEX [name] ON [table]` — MSSQL requires ON table |
| AddPrimaryKey | `ALTER TABLE [table] ADD CONSTRAINT [name] PRIMARY KEY (cols)` |
| DropPrimaryKey | `ALTER TABLE [table] DROP CONSTRAINT [name]` |
| AddForeignKey | `ALTER TABLE [table] ADD CONSTRAINT [name] FOREIGN KEY (cols) REFERENCES [refTable] (refCols) [ON DELETE action] [ON UPDATE action]` |
| DropForeignKey | `ALTER TABLE [table] DROP CONSTRAINT [name]` |
| AddCommentOnColumn | `EXEC sp_addextendedproperty 'MS_Description', 'comment', 'SCHEMA', schema_name(), 'TABLE', 'table', 'COLUMN', 'col'` |
| DropCommentFromColumn | `EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', schema_name(), 'TABLE', 'table', 'COLUMN', 'col'` |
| AddCommentOnTable | `EXEC sp_addextendedproperty 'MS_Description', 'comment', 'SCHEMA', schema_name(), 'TABLE', 'table'` |
| DropCommentFromTable | `EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', schema_name(), 'TABLE', 'table'` |

### Lock Methods

- **AcquireLockSQL**: `EXEC sp_getapplock @Resource = @p1, @LockMode = 'Exclusive', @LockTimeout = @p2` — returns return code as result
- **ReleaseLockSQL**: `EXEC sp_releaseapplock @Resource = @p1`

### DML Methods

Delegate to shared functions. Note: `buildInsert` etc. use `d.Placeholder(index)` which will generate `@p1, @p2, ...`.

## [S6] Shared Function Extraction

### New shared helpers in dialect.go

Extract these common patterns:

1. **`buildGenericCreateIndex`** — `CREATE [UNIQUE] INDEX name ON table (cols)` — shared by PG, SQLite, MySQL
2. **`buildGenericAddPrimaryKey`** — `ALTER TABLE table ADD CONSTRAINT name PRIMARY KEY (cols)` — shared by PG, MSSQL, MySQL
3. **`buildGenericAddForeignKey`** — `ALTER TABLE table ADD CONSTRAINT name FOREIGN KEY ...` — shared by PG, MSSQL, MySQL
4. **`buildGenericDropColumn`** — `ALTER TABLE table DROP COLUMN col` — shared by all

MySQL and SQLite can optionally migrate to these shared functions, but this is not required for the initial implementation.

## [S7] CLI Changes

`cmd/migrate/main.go` `resolveDBConfig()`:
- Add `"postgres"` / `"postgresql"` → `&migrate.PostgreSQLDialect{}`
- Add `"sqlserver"` / `"mssql"` → `&migrate.SQLServerDialect{}`

## [S8] Testing Strategy

### Unit Tests

For each dialect, test SQL generation for:
- All column type mappings via `BuildColumn()`
- SERIAL/IDENTITY primary key generation
- Each DDL method (CreateTable, AddColumn, etc.)
- Each metadata query method
- Unsupported operations return `UnsupportedOperationError`
- DML methods (Insert, Update, Delete) produce correct SQL

### Integration Tests

Same pattern as `mysql_integration_test.go`:
- Skip if `POSTGRES_TEST_DSN` / `SQLSERVER_TEST_DSN` not set
- Create table → insert → query → drop
- Test advisory lock / app lock

## [S9] Key Design Decisions

1. **AlterColumn scope**: Single statement only — ALTER COLUMN TYPE for PG, ALTER COLUMN type [NULL|NOT NULL] for MSSQL. NULL/DEFAULT changes are separate concerns.

2. **PG enum**: Return `UnsupportedOperationError`. Can be extended in the future with `CREATE TYPE ... AS ENUM (...)`.

3. **PG DropIndex ignores table parameter**: PG uses `DROP INDEX name` without `ON table`.

4. **MSSQL comment system**: Uses `sp_addextendedproperty` / `sp_dropextendedproperty` stored procedures, not SQL comments.

5. **No ColumnBuilder changes**: All type mapping happens inside dialect implementations.

6. **Shared DDL helpers**: Extract only when clearly beneficial (3+ dialects share the same SQL). Don't over-extract.
