package migrate

import (
	"errors"
	"reflect"
	"testing"
)

func newPg() *MigrationContext {
	return NewMigrationContext(nil, PostgreSQLDialect{})
}

func TestPostgreSQLName(t *testing.T) {
	d := PostgreSQLDialect{}
	if got := d.Name(); got != "postgres" {
		t.Fatalf("Name() = %q, want %q", got, "postgres")
	}
}

func TestPostgreSQLQuoteTable(t *testing.T) {
	d := PostgreSQLDialect{}
	if got := d.QuoteTable("article"); got != `"article"` {
		t.Fatalf("QuoteTable = %q, want %q", got, `"article"`)
	}
	if got := d.QuoteTable("*"); got != "*" {
		t.Fatalf("QuoteTable(*) = %q, want %q", got, "*")
	}
}

func TestPostgreSQLQuoteColumn(t *testing.T) {
	d := PostgreSQLDialect{}
	if got := d.QuoteColumn("title"); got != `"title"` {
		t.Fatalf("QuoteColumn = %q, want %q", got, `"title"`)
	}
}

func TestPostgreSQLQuoteIndexColumn(t *testing.T) {
	d := PostgreSQLDialect{}
	if got := d.QuoteIndexColumn("user_id"); got != `"user_id"` {
		t.Fatalf("QuoteIndexColumn plain = %q", got)
	}
	if got := d.QuoteIndexColumn("LOWER(email)"); got != "LOWER(email)" {
		t.Fatalf("QuoteIndexColumn expression = %q", got)
	}
}

func TestPostgreSQLPlaceholder(t *testing.T) {
	d := PostgreSQLDialect{}
	if got := d.Placeholder(1); got != "$1" {
		t.Fatalf("Placeholder(1) = %q, want %q", got, "$1")
	}
	if got := d.Placeholder(2); got != "$2" {
		t.Fatalf("Placeholder(2) = %q, want %q", got, "$2")
	}
	if got := d.Placeholder(10); got != "$10" {
		t.Fatalf("Placeholder(10) = %q, want %q", got, "$10")
	}
}

func TestPostgreSQLBuildColumnNil(t *testing.T) {
	d := PostgreSQLDialect{}
	_, err := d.BuildColumn(nil)
	if err == nil {
		t.Fatal("BuildColumn(nil) should return error")
	}
}

func TestPostgreSQLBuildColumnUnsupported(t *testing.T) {
	m := newPg()
	d := PostgreSQLDialect{}
	cases := []struct {
		name string
		col  *ColumnBuilder
		op   string
	}{
		{"unsigned", m.Integer().Unsigned(), "UNSIGNED"},
		{"charset", m.String(64).Charset("utf8mb4"), "CHARACTER SET"},
		{"collation", m.String(64).Collate("utf8mb4_unicode_ci"), "COLLATE"},
		{"after", m.String(64).After("id"), "AFTER"},
		{"first", m.String(64).First(), "FIRST"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := d.BuildColumn(tc.col)
			var uoe *UnsupportedOperationError
			if !errors.As(err, &uoe) {
				t.Fatalf("got error %v, want UnsupportedOperationError", err)
			}
			if uoe.Dialect != "postgres" {
				t.Fatalf("dialect = %q, want %q", uoe.Dialect, "postgres")
			}
			if uoe.Operation != tc.op {
				t.Fatalf("operation = %q, want %q", uoe.Operation, tc.op)
			}
		})
	}
}

func TestPostgreSQLBuildColumnSerialPrimaryKey(t *testing.T) {
	m := newPg()
	d := PostgreSQLDialect{}

	col, err := d.BuildColumn(m.PrimaryKey())
	if err != nil {
		t.Fatalf("BuildColumn error: %v", err)
	}
	if col != "SERIAL PRIMARY KEY" {
		t.Fatalf("integer serial PK = %q, want %q", col, "SERIAL PRIMARY KEY")
	}

	col, err = d.BuildColumn(m.BigPrimaryKey())
	if err != nil {
		t.Fatalf("BuildColumn error: %v", err)
	}
	if col != "BIGSERIAL PRIMARY KEY" {
		t.Fatalf("bigInteger serial PK = %q, want %q", col, "BIGSERIAL PRIMARY KEY")
	}
}

func TestPostgreSQLBuildColumnTypeMappings(t *testing.T) {
	m := newPg()
	d := PostgreSQLDialect{}
	cases := []struct {
		name string
		col  *ColumnBuilder
		want string
	}{
		{"tinyInteger", m.TinyInteger(), "SMALLINT"},
		{"smallInteger", m.SmallInteger(), "SMALLINT"},
		{"integer", m.Integer(), "INTEGER"},
		{"bigInteger", m.BigInteger(), "BIGINT"},
		{"string default", m.String(), "VARCHAR(255)"},
		{"string sized", m.String(128), "VARCHAR(128)"},
		{"char", m.Char(10), "CHAR(10)"},
		{"text", m.Text(), "TEXT"},
		{"tinyText", m.TinyText(), "TEXT"},
		{"mediumText", m.MediumText(), "TEXT"},
		{"longText", m.LongText(), "TEXT"},
		{"binary", m.Binary(), "BYTEA"},
		{"tinyBlob", m.TinyBlob(), "BYTEA"},
		{"mediumBlob", m.MediumBlob(), "BYTEA"},
		{"longBlob", m.LongBlob(), "BYTEA"},
		{"boolean", m.Boolean(), "BOOLEAN"},
		{"float", m.Float(), "REAL"},
		{"double", m.Double(), "DOUBLE PRECISION"},
		{"decimal", m.Decimal(10, 2), "DECIMAL(10, 2)"},
		{"money", m.Money(10, 2), "MONEY"},
		{"date", m.Date(), "DATE"},
		{"dateTime", m.DateTime(), "TIMESTAMP"},
		{"dateTime precision", m.DateTime(3), "TIMESTAMP(3)"},
		{"time", m.Time(), "TIME"},
		{"time precision", m.Time(6), "TIME(6)"},
		{"timestamp", m.Timestamp(), "TIMESTAMP"},
		{"timestamp precision", m.Timestamp(0), "TIMESTAMP(0)"},
		{"json", m.Json(), "JSONB"},
		{"uuid", m.UUID(), "UUID"},
		{"enum", m.Enum("a", "b"), "TEXT"},
		{"set", m.Set("x", "y"), "TEXT"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := d.BuildColumn(tc.col)
			if err != nil {
				t.Fatalf("BuildColumn error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPostgreSQLBuildColumnModifiers(t *testing.T) {
	m := newPg()
	d := PostgreSQLDialect{}
	cases := []struct {
		name string
		col  *ColumnBuilder
		want string
	}{
		{"not null", m.String(64).NotNull(), "VARCHAR(64) NOT NULL"},
		{"null explicit", m.String(64).Null(), "VARCHAR(64) NULL"},
		{"default value", m.Integer().DefaultValue(42), "INTEGER DEFAULT 42"},
		{"default expression", m.Timestamp(0).DefaultExpression("CURRENT_TIMESTAMP"), "TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP"},
		{"unique", m.String(64).Unique(), "VARCHAR(64) UNIQUE"},
		{"check", m.Integer().Check("id > 0"), "INTEGER CHECK (id > 0)"},
		{"append", m.Integer().Append("USING id::integer"), "INTEGER USING id::integer"},
		{"generated as", m.Integer().GeneratedAs("id * 2"), "INTEGER GENERATED ALWAYS AS (id * 2)"},
		{"generated stored", m.Integer().GeneratedAs("id * 2").Stored(), "INTEGER GENERATED ALWAYS AS (id * 2) STORED"},
		{"generated virtual", m.Integer().GeneratedAs("id * 2").Virtual(), "INTEGER GENERATED ALWAYS AS (id * 2) VIRTUAL"},
		{"combined", m.String(128).NotNull().DefaultExpression("''"), "VARCHAR(128) NOT NULL DEFAULT ''"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := d.BuildColumn(tc.col)
			if err != nil {
				t.Fatalf("BuildColumn error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPostgreSQLTableExistsSQL(t *testing.T) {
	d := PostgreSQLDialect{}
	s := d.TableExistsSQL("article")
	want := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1"
	if s.Query != want {
		t.Fatalf("TableExistsSQL query = %q, want %q", s.Query, want)
	}
	if !reflect.DeepEqual(s.Args, []any{"article"}) {
		t.Fatalf("TableExistsSQL args = %#v", s.Args)
	}
}

func TestPostgreSQLColumnExistsSQL(t *testing.T) {
	d := PostgreSQLDialect{}
	s := d.ColumnExistsSQL("article", "title")
	want := "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2"
	if s.Query != want {
		t.Fatalf("ColumnExistsSQL query = %q, want %q", s.Query, want)
	}
	if !reflect.DeepEqual(s.Args, []any{"article", "title"}) {
		t.Fatalf("ColumnExistsSQL args = %#v", s.Args)
	}
}

func TestPostgreSQLIndexExistsSQL(t *testing.T) {
	d := PostgreSQLDialect{}
	s := d.IndexExistsSQL("article", "idx-title")
	want := "SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND tablename = $1 AND indexname = $2"
	if s.Query != want {
		t.Fatalf("IndexExistsSQL query = %q, want %q", s.Query, want)
	}
	if !reflect.DeepEqual(s.Args, []any{"article", "idx-title"}) {
		t.Fatalf("IndexExistsSQL args = %#v", s.Args)
	}
}

func TestPostgreSQLForeignKeyExistsSQL(t *testing.T) {
	d := PostgreSQLDialect{}
	s := d.ForeignKeyExistsSQL("article", "fk_user")
	want := "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = current_schema() AND table_name = $1 AND constraint_name = $2 AND constraint_type = 'FOREIGN KEY'"
	if s.Query != want {
		t.Fatalf("ForeignKeyExistsSQL query = %q, want %q", s.Query, want)
	}
	if !reflect.DeepEqual(s.Args, []any{"article", "fk_user"}) {
		t.Fatalf("ForeignKeyExistsSQL args = %#v", s.Args)
	}
}

func TestPostgreSQLConstraintExistsSQL(t *testing.T) {
	d := PostgreSQLDialect{}
	s := d.ConstraintExistsSQL("article", "chk_id")
	want := "SELECT COUNT(*) FROM information_schema.table_constraints WHERE table_schema = current_schema() AND table_name = $1 AND constraint_name = $2"
	if s.Query != want {
		t.Fatalf("ConstraintExistsSQL query = %q, want %q", s.Query, want)
	}
	if !reflect.DeepEqual(s.Args, []any{"article", "chk_id"}) {
		t.Fatalf("ConstraintExistsSQL args = %#v", s.Args)
	}
}

func TestPostgreSQLBuildRowExistsSQL(t *testing.T) {
	d := PostgreSQLDialect{}
	got := d.BuildRowExistsSQL("user", "username = $1")
	want := `SELECT EXISTS(SELECT 1 FROM "user" WHERE username = $1)`
	if got != want {
		t.Fatalf("BuildRowExistsSQL = %q, want %q", got, want)
	}
}

func TestPostgreSQLBuildRowExistsSQLNoCondition(t *testing.T) {
	d := PostgreSQLDialect{}
	got := d.BuildRowExistsSQL("user", "")
	want := `SELECT EXISTS(SELECT 1 FROM "user")`
	if got != want {
		t.Fatalf("BuildRowExistsSQL empty = %q, want %q", got, want)
	}
}

func TestPostgreSQLBuildCountRowsSQL(t *testing.T) {
	d := PostgreSQLDialect{}
	got := d.BuildCountRowsSQL("article", "status = $1")
	want := `SELECT COUNT(*) FROM "article" WHERE status = $1`
	if got != want {
		t.Fatalf("BuildCountRowsSQL = %q, want %q", got, want)
	}
}

func TestPostgreSQLCreateTable(t *testing.T) {
	m := newPg()
	d := PostgreSQLDialect{}
	sql, err := d.CreateTable("article", Columns().
		Add("id", m.PrimaryKey()).
		Add("title", m.String(128).NotNull()).
		Add("metadata", m.Json().Null()).
		Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")),
		"",
	)
	if err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}
	want := `CREATE TABLE "article" ("id" SERIAL PRIMARY KEY, "title" VARCHAR(128) NOT NULL, "metadata" JSONB NULL, "created_at" TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP)`
	if sql != want {
		t.Fatalf("CreateTable SQL:\n got: %s\nwant: %s", sql, want)
	}
}

func TestPostgreSQLCreateTableWithOptions(t *testing.T) {
	m := newPg()
	d := PostgreSQLDialect{}
	sql, err := d.CreateTable("article", Columns().
		Add("id", m.PrimaryKey()).
		Add("title", m.String(128).NotNull()),
		"WITH (fillfactor = 90)",
	)
	if err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}
	want := `CREATE TABLE "article" ("id" SERIAL PRIMARY KEY, "title" VARCHAR(128) NOT NULL) WITH (fillfactor = 90)`
	if sql != want {
		t.Fatalf("CreateTable SQL:\n got: %s\nwant: %s", sql, want)
	}
}

func TestPostgreSQLDropTable(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.DropTable("article")
	if err != nil {
		t.Fatalf("DropTable error: %v", err)
	}
	want := `DROP TABLE IF EXISTS "article"`
	if sql != want {
		t.Fatalf("DropTable = %q, want %q", sql, want)
	}
}

func TestPostgreSQLRenameTable(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.RenameTable("article", "post")
	if err != nil {
		t.Fatalf("RenameTable error: %v", err)
	}
	want := `ALTER TABLE "article" RENAME TO "post"`
	if sql != want {
		t.Fatalf("RenameTable = %q, want %q", sql, want)
	}
}

func TestPostgreSQLTruncateTable(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.TruncateTable("article")
	if err != nil {
		t.Fatalf("TruncateTable error: %v", err)
	}
	want := `TRUNCATE TABLE "article"`
	if sql != want {
		t.Fatalf("TruncateTable = %q, want %q", sql, want)
	}
}

func TestPostgreSQLAddColumn(t *testing.T) {
	m := newPg()
	d := PostgreSQLDialect{}
	sql, err := d.AddColumn("article", "summary", m.Text().Null())
	if err != nil {
		t.Fatalf("AddColumn error: %v", err)
	}
	want := `ALTER TABLE "article" ADD COLUMN "summary" TEXT NULL`
	if sql != want {
		t.Fatalf("AddColumn = %q, want %q", sql, want)
	}
}

func TestPostgreSQLAlterColumn(t *testing.T) {
	m := newPg()
	d := PostgreSQLDialect{}
	sql, err := d.AlterColumn("article", "title", m.String(256))
	if err != nil {
		t.Fatalf("AlterColumn error: %v", err)
	}
	want := `ALTER TABLE "article" ALTER COLUMN "title" TYPE VARCHAR(256)`
	if sql != want {
		t.Fatalf("AlterColumn = %q, want %q", sql, want)
	}
}

func TestPostgreSQLDropColumn(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.DropColumn("article", "summary")
	if err != nil {
		t.Fatalf("DropColumn error: %v", err)
	}
	want := `ALTER TABLE "article" DROP COLUMN "summary"`
	if sql != want {
		t.Fatalf("DropColumn = %q, want %q", sql, want)
	}
}

func TestPostgreSQLRenameColumn(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.RenameColumn("article", "title", "heading")
	if err != nil {
		t.Fatalf("RenameColumn error: %v", err)
	}
	want := `ALTER TABLE "article" RENAME COLUMN "title" TO "heading"`
	if sql != want {
		t.Fatalf("RenameColumn = %q, want %q", sql, want)
	}
}

func TestPostgreSQLCreateIndex(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.CreateIndex("idx_article_title", "article", []string{"title"}, false)
	if err != nil {
		t.Fatalf("CreateIndex error: %v", err)
	}
	want := `CREATE INDEX "idx_article_title" ON "article" ("title")`
	if sql != want {
		t.Fatalf("CreateIndex = %q, want %q", sql, want)
	}
}

func TestPostgreSQLCreateUniqueIndex(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.CreateIndex("idx_article_slug", "article", []string{"slug"}, true)
	if err != nil {
		t.Fatalf("CreateIndex unique error: %v", err)
	}
	want := `CREATE UNIQUE INDEX "idx_article_slug" ON "article" ("slug")`
	if sql != want {
		t.Fatalf("CreateIndex unique = %q, want %q", sql, want)
	}
}

func TestPostgreSQLCreateIndexExpression(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.CreateIndex("idx_article_lower_title", "article", []string{"LOWER(title)"}, false)
	if err != nil {
		t.Fatalf("CreateIndex expression error: %v", err)
	}
	want := `CREATE INDEX "idx_article_lower_title" ON "article" (LOWER(title))`
	if sql != want {
		t.Fatalf("CreateIndex expression = %q, want %q", sql, want)
	}
}

func TestPostgreSQLDropIndex(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.DropIndex("idx_article_title", "article")
	if err != nil {
		t.Fatalf("DropIndex error: %v", err)
	}
	want := `DROP INDEX IF EXISTS "idx_article_title"`
	if sql != want {
		t.Fatalf("DropIndex = %q, want %q", sql, want)
	}
}

func TestPostgreSQLAddPrimaryKey(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.AddPrimaryKey("pk_article", "article", []string{"id"})
	if err != nil {
		t.Fatalf("AddPrimaryKey error: %v", err)
	}
	want := `ALTER TABLE "article" ADD CONSTRAINT "pk_article" PRIMARY KEY ("id")`
	if sql != want {
		t.Fatalf("AddPrimaryKey = %q, want %q", sql, want)
	}
}

func TestPostgreSQLDropPrimaryKey(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.DropPrimaryKey("pk_article", "article")
	if err != nil {
		t.Fatalf("DropPrimaryKey error: %v", err)
	}
	want := `ALTER TABLE "article" DROP CONSTRAINT "pk_article"`
	if sql != want {
		t.Fatalf("DropPrimaryKey = %q, want %q", sql, want)
	}
}

func TestPostgreSQLAddForeignKey(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.AddForeignKey("fk_article_user", "article", []string{"user_id"}, "user", []string{"id"}, Cascade, NoAction)
	if err != nil {
		t.Fatalf("AddForeignKey error: %v", err)
	}
	want := `ALTER TABLE "article" ADD CONSTRAINT "fk_article_user" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON DELETE CASCADE ON UPDATE NO ACTION`
	if sql != want {
		t.Fatalf("AddForeignKey = %q, want %q", sql, want)
	}
}

func TestPostgreSQLAddForeignKeyNoActions(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.AddForeignKey("fk_article_user", "article", []string{"user_id"}, "user", []string{"id"}, "", "")
	if err != nil {
		t.Fatalf("AddForeignKey error: %v", err)
	}
	want := `ALTER TABLE "article" ADD CONSTRAINT "fk_article_user" FOREIGN KEY ("user_id") REFERENCES "user" ("id")`
	if sql != want {
		t.Fatalf("AddForeignKey no actions = %q, want %q", sql, want)
	}
}

func TestPostgreSQLDropForeignKey(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.DropForeignKey("fk_article_user", "article")
	if err != nil {
		t.Fatalf("DropForeignKey error: %v", err)
	}
	want := `ALTER TABLE "article" DROP CONSTRAINT "fk_article_user"`
	if sql != want {
		t.Fatalf("DropForeignKey = %q, want %q", sql, want)
	}
}

func TestPostgreSQLAddCommentOnColumn(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.AddCommentOnColumn("article", "title", "The title")
	if err != nil {
		t.Fatalf("AddCommentOnColumn error: %v", err)
	}
	want := `COMMENT ON COLUMN "article"."title" IS 'The title'`
	if sql != want {
		t.Fatalf("AddCommentOnColumn = %q, want %q", sql, want)
	}
}

func TestPostgreSQLDropCommentFromColumn(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.DropCommentFromColumn("article", "title")
	if err != nil {
		t.Fatalf("DropCommentFromColumn error: %v", err)
	}
	want := `COMMENT ON COLUMN "article"."title" IS NULL`
	if sql != want {
		t.Fatalf("DropCommentFromColumn = %q, want %q", sql, want)
	}
}

func TestPostgreSQLAddCommentOnTable(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.AddCommentOnTable("article", "Articles table")
	if err != nil {
		t.Fatalf("AddCommentOnTable error: %v", err)
	}
	want := `COMMENT ON TABLE "article" IS 'Articles table'`
	if sql != want {
		t.Fatalf("AddCommentOnTable = %q, want %q", sql, want)
	}
}

func TestPostgreSQLDropCommentFromTable(t *testing.T) {
	d := PostgreSQLDialect{}
	sql, err := d.DropCommentFromTable("article")
	if err != nil {
		t.Fatalf("DropCommentFromTable error: %v", err)
	}
	want := `COMMENT ON TABLE "article" IS NULL`
	if sql != want {
		t.Fatalf("DropCommentFromTable = %q, want %q", sql, want)
	}
}

func TestPostgreSQLInsert(t *testing.T) {
	d := PostgreSQLDialect{}
	stmt, err := d.Insert("article", Row{
		"title":      "Hello",
		"created_at": Expr("CURRENT_TIMESTAMP"),
	})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}
	wantQuery := `INSERT INTO "article" ("created_at", "title") VALUES (CURRENT_TIMESTAMP, $1)`
	if stmt.Query != wantQuery {
		t.Fatalf("Insert query:\n got: %s\nwant: %s", stmt.Query, wantQuery)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"Hello"}) {
		t.Fatalf("Insert args = %#v", stmt.Args)
	}
}

func TestPostgreSQLBatchInsert(t *testing.T) {
	d := PostgreSQLDialect{}
	stmt, err := d.BatchInsert("article", []string{"title", "status"}, [][]any{
		{"Hello", "draft"},
		{"World", "published"},
	})
	if err != nil {
		t.Fatalf("BatchInsert error: %v", err)
	}
	wantQuery := `INSERT INTO "article" ("title", "status") VALUES ($1, $2), ($3, $4)`
	if stmt.Query != wantQuery {
		t.Fatalf("BatchInsert query:\n got: %s\nwant: %s", stmt.Query, wantQuery)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"Hello", "draft", "World", "published"}) {
		t.Fatalf("BatchInsert args = %#v", stmt.Args)
	}
}

func TestPostgreSQLUpdate(t *testing.T) {
	d := PostgreSQLDialect{}
	stmt, err := d.Update("article", Row{"title": "Updated"}, `"id" = $1`, 42)
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	wantQuery := `UPDATE "article" SET "title" = $1 WHERE "id" = $1`
	if stmt.Query != wantQuery {
		t.Fatalf("Update query:\n got: %s\nwant: %s", stmt.Query, wantQuery)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"Updated", 42}) {
		t.Fatalf("Update args = %#v", stmt.Args)
	}
}

func TestPostgreSQLUpdateNoCondition(t *testing.T) {
	d := PostgreSQLDialect{}
	stmt, err := d.Update("article", Row{"title": "All"}, "")
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	wantQuery := `UPDATE "article" SET "title" = $1`
	if stmt.Query != wantQuery {
		t.Fatalf("Update no cond query:\n got: %s\nwant: %s", stmt.Query, wantQuery)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"All"}) {
		t.Fatalf("Update no cond args = %#v", stmt.Args)
	}
}

func TestPostgreSQLDelete(t *testing.T) {
	d := PostgreSQLDialect{}
	stmt, err := d.Delete("article", `"id" = $1`, 42)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	wantQuery := `DELETE FROM "article" WHERE "id" = $1`
	if stmt.Query != wantQuery {
		t.Fatalf("Delete query:\n got: %s\nwant: %s", stmt.Query, wantQuery)
	}
	if !reflect.DeepEqual(stmt.Args, []any{42}) {
		t.Fatalf("Delete args = %#v", stmt.Args)
	}
}

func TestPostgreSQLDeleteNoCondition(t *testing.T) {
	d := PostgreSQLDialect{}
	stmt, err := d.Delete("article", "")
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	wantQuery := `DELETE FROM "article"`
	if stmt.Query != wantQuery {
		t.Fatalf("Delete no cond query:\n got: %s\nwant: %s", stmt.Query, wantQuery)
	}
	if len(stmt.Args) != 0 {
		t.Fatalf("Delete no cond args = %#v, want empty", stmt.Args)
	}
}

func TestPostgreSQLAcquireLockSQL(t *testing.T) {
	d := PostgreSQLDialect{}
	stmt, err := d.AcquireLockSQL("my_migration", 10)
	if err != nil {
		t.Fatalf("AcquireLockSQL error: %v", err)
	}
	wantQuery := `SELECT pg_advisory_lock(hashtext($1))`
	if stmt.Query != wantQuery {
		t.Fatalf("AcquireLockSQL query = %q, want %q", stmt.Query, wantQuery)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"my_migration"}) {
		t.Fatalf("AcquireLockSQL args = %#v", stmt.Args)
	}
}

func TestPostgreSQLReleaseLockSQL(t *testing.T) {
	d := PostgreSQLDialect{}
	stmt, err := d.ReleaseLockSQL("my_migration")
	if err != nil {
		t.Fatalf("ReleaseLockSQL error: %v", err)
	}
	wantQuery := `SELECT pg_advisory_unlock(hashtext($1))`
	if stmt.Query != wantQuery {
		t.Fatalf("ReleaseLockSQL query = %q, want %q", stmt.Query, wantQuery)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"my_migration"}) {
		t.Fatalf("ReleaseLockSQL args = %#v", stmt.Args)
	}
}
