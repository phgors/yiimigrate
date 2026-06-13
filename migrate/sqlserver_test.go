package migrate

import (
	"errors"
	"reflect"
	"testing"
)

func newSQLServerCtx() *MigrationContext {
	return NewMigrationContext(nil, SQLServerDialect{})
}

func TestSQLServerName(t *testing.T) {
	d := SQLServerDialect{}
	if got := d.Name(); got != "sqlserver" {
		t.Fatalf("Name() = %q, want %q", got, "sqlserver")
	}
}

func TestSQLServerQuoteTable(t *testing.T) {
	d := SQLServerDialect{}
	tests := []struct{ input, want string }{
		{"article", "[article]"},
		{"dbo.article", "[dbo].[article]"},
		{"*", "*"},
	}
	for _, tt := range tests {
		if got := d.QuoteTable(tt.input); got != tt.want {
			t.Fatalf("QuoteTable(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSQLServerQuoteColumn(t *testing.T) {
	d := SQLServerDialect{}
	if got := d.QuoteColumn("title"); got != "[title]" {
		t.Fatalf("QuoteColumn(\"title\") = %q, want %q", got, "[title]")
	}
	if got := d.QuoteColumn("name]"); got != "[name]]]" {
		t.Fatalf("QuoteColumn with ] = %q, want %q", got, "[name]]]")
	}
}

func TestSQLServerPlaceholder(t *testing.T) {
	d := SQLServerDialect{}
	if got := d.Placeholder(1); got != "@p1" {
		t.Fatalf("Placeholder(1) = %q, want %q", got, "@p1")
	}
	if got := d.Placeholder(2); got != "@p2" {
		t.Fatalf("Placeholder(2) = %q, want %q", got, "@p2")
	}
	if got := d.Placeholder(10); got != "@p10" {
		t.Fatalf("Placeholder(10) = %q, want %q", got, "@p10")
	}
}

func TestSQLServerBuildColumnTypeMappings(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	tests := []struct {
		name string
		col  *ColumnBuilder
		want string
	}{
		{"tinyInteger", m.TinyInteger(), "TINYINT"},
		{"smallInteger", m.SmallInteger(), "SMALLINT"},
		{"integer", m.Integer(), "INT"},
		{"bigInteger", m.BigInteger(), "BIGINT"},
		{"boolean", m.Boolean(), "BIT"},
		{"uuid", m.UUID(), "UNIQUEIDENTIFIER"},
		{"json", m.Json(), "NVARCHAR(MAX)"},
		{"text", m.Text(), "NVARCHAR(MAX)"},
		{"tinyText", m.TinyText(), "NVARCHAR(MAX)"},
		{"mediumText", m.MediumText(), "NVARCHAR(MAX)"},
		{"longText", m.LongText(), "NVARCHAR(MAX)"},
		{"string default", m.String(), "NVARCHAR(255)"},
		{"string with size", m.String(128), "NVARCHAR(128)"},
		{"char", m.Char(10), "NCHAR(10)"},
		{"binary with size", m.Binary(255), "VARBINARY(255)"},
		{"float default", m.Float(), "FLOAT"},
		{"float with precision", m.Float(53), "FLOAT(53)"},
		{"double", m.Double(), "DOUBLE PRECISION"},
		{"decimal", m.Decimal(10, 2), "DECIMAL(10, 2)"},
		{"money", m.Money(10, 2), "MONEY"},
		{"date", m.Date(), "DATE"},
		{"dateTime default", m.DateTime(), "DATETIME2"},
		{"dateTime with precision", m.DateTime(3), "DATETIME2(3)"},
		{"time default", m.Time(), "TIME"},
		{"time with precision", m.Time(6), "TIME(6)"},
		{"timestamp default", m.Timestamp(), "DATETIME2"},
		{"timestamp with precision", m.Timestamp(0), "DATETIME2(0)"},
		{"tinyBlob", m.TinyBlob(), "VARBINARY(MAX)"},
		{"mediumBlob", m.MediumBlob(), "VARBINARY(MAX)"},
		{"longBlob", m.LongBlob(), "VARBINARY(MAX)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.BuildColumn(tt.col)
			if err != nil {
				t.Fatalf("BuildColumn(%s) returned error: %v", tt.name, err)
			}
			if got != tt.want {
				t.Fatalf("BuildColumn(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestSQLServerBuildColumnIdentity(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	got, err := d.BuildColumn(m.PrimaryKey())
	if err != nil {
		t.Fatalf("BuildColumn integer identity returned error: %v", err)
	}
	if got != "INT IDENTITY(1,1) PRIMARY KEY NOT NULL" {
		t.Fatalf("integer identity = %q, want %q", got, "INT IDENTITY(1,1) PRIMARY KEY NOT NULL")
	}

	got, err = d.BuildColumn(m.BigPrimaryKey())
	if err != nil {
		t.Fatalf("BuildColumn bigInteger identity returned error: %v", err)
	}
	if got != "BIGINT IDENTITY(1,1) PRIMARY KEY NOT NULL" {
		t.Fatalf("bigInteger identity = %q, want %q", got, "BIGINT IDENTITY(1,1) PRIMARY KEY NOT NULL")
	}
}

func TestSQLServerBuildColumnNullNotNull(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	got, err := d.BuildColumn(m.String(128).NotNull())
	if err != nil {
		t.Fatalf("NotNull returned error: %v", err)
	}
	if got != "NVARCHAR(128) NOT NULL" {
		t.Fatalf("NotNull = %q, want %q", got, "NVARCHAR(128) NOT NULL")
	}

	got, err = d.BuildColumn(m.String(128).Null())
	if err != nil {
		t.Fatalf("Null returned error: %v", err)
	}
	if got != "NVARCHAR(128) NULL" {
		t.Fatalf("Null = %q, want %q", got, "NVARCHAR(128) NULL")
	}
}

func TestSQLServerBuildColumnPrimaryKeyNonIdentity(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	got, err := d.BuildColumn(m.String(36).PrimaryKey())
	if err != nil {
		t.Fatalf("PrimaryKey non-identity returned error: %v", err)
	}
	if got != "NVARCHAR(36) NOT NULL PRIMARY KEY" {
		t.Fatalf("PrimaryKey non-identity = %q, want %q", got, "NVARCHAR(36) NOT NULL PRIMARY KEY")
	}
}

func TestSQLServerBuildColumnDefaults(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	got, err := d.BuildColumn(m.Integer().DefaultExpression("CURRENT_TIMESTAMP"))
	if err != nil {
		t.Fatalf("DefaultExpression returned error: %v", err)
	}
	if got != "INT DEFAULT CURRENT_TIMESTAMP" {
		t.Fatalf("DefaultExpression = %q, want %q", got, "INT DEFAULT CURRENT_TIMESTAMP")
	}

	got, err = d.BuildColumn(m.Integer().DefaultValue(42))
	if err != nil {
		t.Fatalf("DefaultValue returned error: %v", err)
	}
	if got != "INT DEFAULT 42" {
		t.Fatalf("DefaultValue = %q, want %q", got, "INT DEFAULT 42")
	}
}

func TestSQLServerBuildColumnCheck(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	got, err := d.BuildColumn(m.Integer().Check("value > 0"))
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if got != "INT CHECK (value > 0)" {
		t.Fatalf("Check = %q, want %q", got, "INT CHECK (value > 0)")
	}
}

func TestSQLServerBuildColumnUnique(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	got, err := d.BuildColumn(m.String(128).Unique())
	if err != nil {
		t.Fatalf("Unique returned error: %v", err)
	}
	if got != "NVARCHAR(128) UNIQUE" {
		t.Fatalf("Unique = %q, want %q", got, "NVARCHAR(128) UNIQUE")
	}
}

func TestSQLServerBuildColumnAppend(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	got, err := d.BuildColumn(m.String(128).Append("COLLATE Latin1_General_CI_AS"))
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if got != "NVARCHAR(128) COLLATE Latin1_General_CI_AS" {
		t.Fatalf("Append = %q, want %q", got, "NVARCHAR(128) COLLATE Latin1_General_CI_AS")
	}
}

func TestSQLServerBuildColumnUnsupported(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}
	mustUnsupported := func(name string, builder *ColumnBuilder) {
		t.Helper()
		_, err := d.BuildColumn(builder)
		var ue *UnsupportedOperationError
		if !errors.As(err, &ue) {
			t.Fatalf("%s: expected UnsupportedOperationError, got %v", name, err)
		}
		if ue.Dialect != "sqlserver" {
			t.Fatalf("%s: dialect = %q, want %q", name, ue.Dialect, "sqlserver")
		}
	}

	mustUnsupported("unsigned", m.Integer().Unsigned())
	mustUnsupported("charset", m.String(128).Charset("utf8"))
	mustUnsupported("collation", m.String(128).Collate("utf8_general_ci"))
	mustUnsupported("after", m.String(128).After("other_col"))
	mustUnsupported("first", m.String(128).First())
	mustUnsupported("generatedAs", m.Integer().GeneratedAs("id * 2"))
}

func TestSQLServerBuildColumnEnumUnsupported(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	_, err := d.BuildColumn(m.Enum("active", "inactive"))
	var ue *UnsupportedOperationError
	if !errors.As(err, &ue) {
		t.Fatalf("enum: expected UnsupportedOperationError, got %v", err)
	}

	_, err = d.BuildColumn(m.Set("a", "b"))
	if !errors.As(err, &ue) {
		t.Fatalf("set: expected UnsupportedOperationError, got %v", err)
	}
}

func TestSQLServerBuildColumnNil(t *testing.T) {
	d := SQLServerDialect{}
	_, err := d.BuildColumn(nil)
	if err == nil {
		t.Fatalf("nil ColumnBuilder should return error")
	}
}

func TestSQLServerTableExistsSQL(t *testing.T) {
	d := SQLServerDialect{}
	stmt := d.TableExistsSQL("article")
	want := "SELECT COUNT(*) FROM sys.tables WHERE SCHEMA_NAME(schema_id) = SCHEMA_NAME() AND name = @p1"
	if stmt.Query != want {
		t.Fatalf("TableExistsSQL query = %q, want %q", stmt.Query, want)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"article"}) {
		t.Fatalf("TableExistsSQL args = %#v", stmt.Args)
	}
}

func TestSQLServerColumnExistsSQL(t *testing.T) {
	d := SQLServerDialect{}
	stmt := d.ColumnExistsSQL("article", "title")
	if stmt.Query == "" {
		t.Fatalf("ColumnExistsSQL query is empty")
	}
	if !reflect.DeepEqual(stmt.Args, []any{"article", "title"}) {
		t.Fatalf("ColumnExistsSQL args = %#v", stmt.Args)
	}
}

func TestSQLServerIndexExistsSQL(t *testing.T) {
	d := SQLServerDialect{}
	stmt := d.IndexExistsSQL("article", "idx_title")
	if !reflect.DeepEqual(stmt.Args, []any{"article", "idx_title"}) {
		t.Fatalf("IndexExistsSQL args = %#v", stmt.Args)
	}
}

func TestSQLServerForeignKeyExistsSQL(t *testing.T) {
	d := SQLServerDialect{}
	stmt := d.ForeignKeyExistsSQL("article", "fk_user")
	if !reflect.DeepEqual(stmt.Args, []any{"article", "fk_user"}) {
		t.Fatalf("ForeignKeyExistsSQL args = %#v", stmt.Args)
	}
}

func TestSQLServerConstraintExistsSQL(t *testing.T) {
	d := SQLServerDialect{}
	stmt := d.ConstraintExistsSQL("article", "chk_status")
	if !reflect.DeepEqual(stmt.Args, []any{"article", "chk_status"}) {
		t.Fatalf("ConstraintExistsSQL args = %#v", stmt.Args)
	}
}

func TestSQLServerBuildRowExistsSQL(t *testing.T) {
	d := SQLServerDialect{}
	got := d.BuildRowExistsSQL("user", "username = @p1")
	want := "SELECT CASE WHEN EXISTS(SELECT 1 FROM [user] WHERE username = @p1) THEN 1 ELSE 0 END"
	if got != want {
		t.Fatalf("BuildRowExistsSQL = %q, want %q", got, want)
	}
}

func TestSQLServerBuildRowExistsSQLNoCondition(t *testing.T) {
	d := SQLServerDialect{}
	got := d.BuildRowExistsSQL("user", "")
	want := "SELECT CASE WHEN EXISTS(SELECT 1 FROM [user]) THEN 1 ELSE 0 END"
	if got != want {
		t.Fatalf("BuildRowExistsSQL no condition = %q, want %q", got, want)
	}
}

func TestSQLServerBuildCountRowsSQL(t *testing.T) {
	d := SQLServerDialect{}
	got := d.BuildCountRowsSQL("article", "status = @p1")
	want := "SELECT COUNT(*) FROM [article] WHERE status = @p1"
	if got != want {
		t.Fatalf("BuildCountRowsSQL = %q, want %q", got, want)
	}
}

func TestSQLServerCreateTable(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	sql, err := d.CreateTable("article", Columns().
		Add("id", m.PrimaryKey()).
		Add("title", m.String(128).NotNull()).
		Add("content", m.LongText().Null()),
		"",
	)
	if err != nil {
		t.Fatalf("CreateTable returned error: %v", err)
	}
	want := "CREATE TABLE [article] ([id] INT IDENTITY(1,1) PRIMARY KEY NOT NULL, [title] NVARCHAR(128) NOT NULL, [content] NVARCHAR(MAX) NULL)"
	if sql != want {
		t.Fatalf("CreateTable SQL:\n got: %s\nwant: %s", sql, want)
	}
}

func TestSQLServerCreateTableWithOptionsUnsupported(t *testing.T) {
	d := SQLServerDialect{}
	_, err := d.CreateTable("article", Columns().Add("id", newSQLServerCtx().PrimaryKey()), "ENGINE=InnoDB")
	var ue *UnsupportedOperationError
	if !errors.As(err, &ue) {
		t.Fatalf("expected UnsupportedOperationError, got %v", err)
	}
	if ue.Operation != "TABLE OPTIONS" {
		t.Fatalf("operation = %q, want %q", ue.Operation, "TABLE OPTIONS")
	}
}

func TestSQLServerDropTable(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.DropTable("article")
	if err != nil {
		t.Fatalf("DropTable returned error: %v", err)
	}
	want := "DROP TABLE IF EXISTS [article]"
	if sql != want {
		t.Fatalf("DropTable = %q, want %q", sql, want)
	}
}

func TestSQLServerRenameTable(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.RenameTable("old_name", "new_name")
	if err != nil {
		t.Fatalf("RenameTable returned error: %v", err)
	}
	want := "EXEC sp_rename 'old_name', 'new_name'"
	if sql != want {
		t.Fatalf("RenameTable = %q, want %q", sql, want)
	}
}

func TestSQLServerTruncateTable(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.TruncateTable("article")
	if err != nil {
		t.Fatalf("TruncateTable returned error: %v", err)
	}
	want := "TRUNCATE TABLE [article]"
	if sql != want {
		t.Fatalf("TruncateTable = %q, want %q", sql, want)
	}
}

func TestSQLServerAddColumn(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	sql, err := d.AddColumn("article", "summary", m.String(255).NotNull())
	if err != nil {
		t.Fatalf("AddColumn returned error: %v", err)
	}
	want := "ALTER TABLE [article] ADD [summary] NVARCHAR(255) NOT NULL"
	if sql != want {
		t.Fatalf("AddColumn = %q, want %q", sql, want)
	}
}

func TestSQLServerAlterColumn(t *testing.T) {
	m := newSQLServerCtx()
	d := SQLServerDialect{}

	sql, err := d.AlterColumn("article", "title", m.String(256).NotNull())
	if err != nil {
		t.Fatalf("AlterColumn returned error: %v", err)
	}
	want := "ALTER TABLE [article] ALTER COLUMN [title] NVARCHAR(256) NOT NULL"
	if sql != want {
		t.Fatalf("AlterColumn = %q, want %q", sql, want)
	}
}

func TestSQLServerDropColumn(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.DropColumn("article", "summary")
	if err != nil {
		t.Fatalf("DropColumn returned error: %v", err)
	}
	want := "ALTER TABLE [article] DROP COLUMN [summary]"
	if sql != want {
		t.Fatalf("DropColumn = %q, want %q", sql, want)
	}
}

func TestSQLServerRenameColumn(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.RenameColumn("article", "old_col", "new_col")
	if err != nil {
		t.Fatalf("RenameColumn returned error: %v", err)
	}
	want := "EXEC sp_rename 'article.old_col', 'new_col', 'COLUMN'"
	if sql != want {
		t.Fatalf("RenameColumn = %q, want %q", sql, want)
	}
}

func TestSQLServerCreateIndex(t *testing.T) {
	d := SQLServerDialect{}

	sql, err := d.CreateIndex("idx_title", "article", []string{"title"}, false)
	if err != nil {
		t.Fatalf("CreateIndex returned error: %v", err)
	}
	want := "CREATE INDEX [idx_title] ON [article] ([title])"
	if sql != want {
		t.Fatalf("CreateIndex = %q, want %q", sql, want)
	}

	sql, err = d.CreateIndex("idx_title", "article", []string{"title"}, true)
	if err != nil {
		t.Fatalf("CreateIndex unique returned error: %v", err)
	}
	want = "CREATE UNIQUE INDEX [idx_title] ON [article] ([title])"
	if sql != want {
		t.Fatalf("CreateIndex unique = %q, want %q", sql, want)
	}
}

func TestSQLServerDropIndex(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.DropIndex("idx_title", "article")
	if err != nil {
		t.Fatalf("DropIndex returned error: %v", err)
	}
	want := "DROP INDEX [idx_title] ON [article]"
	if sql != want {
		t.Fatalf("DropIndex = %q, want %q", sql, want)
	}
}

func TestSQLServerAddPrimaryKey(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.AddPrimaryKey("pk_article", "article", []string{"id"})
	if err != nil {
		t.Fatalf("AddPrimaryKey returned error: %v", err)
	}
	want := "ALTER TABLE [article] ADD CONSTRAINT [pk_article] PRIMARY KEY ([id])"
	if sql != want {
		t.Fatalf("AddPrimaryKey = %q, want %q", sql, want)
	}
}

func TestSQLServerDropPrimaryKey(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.DropPrimaryKey("pk_article", "article")
	if err != nil {
		t.Fatalf("DropPrimaryKey returned error: %v", err)
	}
	want := "ALTER TABLE [article] DROP CONSTRAINT [pk_article]"
	if sql != want {
		t.Fatalf("DropPrimaryKey = %q, want %q", sql, want)
	}
}

func TestSQLServerAddForeignKey(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.AddForeignKey("fk_user", "article", []string{"user_id"}, "user", []string{"id"}, Cascade, NoAction)
	if err != nil {
		t.Fatalf("AddForeignKey returned error: %v", err)
	}
	want := "ALTER TABLE [article] ADD CONSTRAINT [fk_user] FOREIGN KEY ([user_id]) REFERENCES [user] ([id]) ON DELETE CASCADE ON UPDATE NO ACTION"
	if sql != want {
		t.Fatalf("AddForeignKey = %q, want %q", sql, want)
	}
}

func TestSQLServerAddForeignKeyNoActions(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.AddForeignKey("fk_user", "article", []string{"user_id"}, "user", []string{"id"}, "", "")
	if err != nil {
		t.Fatalf("AddForeignKey returned error: %v", err)
	}
	want := "ALTER TABLE [article] ADD CONSTRAINT [fk_user] FOREIGN KEY ([user_id]) REFERENCES [user] ([id])"
	if sql != want {
		t.Fatalf("AddForeignKey no actions = %q, want %q", sql, want)
	}
}

func TestSQLServerDropForeignKey(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.DropForeignKey("fk_user", "article")
	if err != nil {
		t.Fatalf("DropForeignKey returned error: %v", err)
	}
	want := "ALTER TABLE [article] DROP CONSTRAINT [fk_user]"
	if sql != want {
		t.Fatalf("DropForeignKey = %q, want %q", sql, want)
	}
}

func TestSQLServerAddCommentOnColumn(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.AddCommentOnColumn("article", "title", "The title")
	if err != nil {
		t.Fatalf("AddCommentOnColumn returned error: %v", err)
	}
	want := "EXEC sp_addextendedproperty 'MS_Description', 'The title', 'SCHEMA', SCHEMA_NAME(), 'TABLE', 'article', 'COLUMN', 'title'"
	if sql != want {
		t.Fatalf("AddCommentOnColumn = %q, want %q", sql, want)
	}
}

func TestSQLServerDropCommentFromColumn(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.DropCommentFromColumn("article", "title")
	if err != nil {
		t.Fatalf("DropCommentFromColumn returned error: %v", err)
	}
	want := "EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', SCHEMA_NAME(), 'TABLE', 'article', 'COLUMN', 'title'"
	if sql != want {
		t.Fatalf("DropCommentFromColumn = %q, want %q", sql, want)
	}
}

func TestSQLServerAddCommentOnTable(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.AddCommentOnTable("article", "Articles table")
	if err != nil {
		t.Fatalf("AddCommentOnTable returned error: %v", err)
	}
	want := "EXEC sp_addextendedproperty 'MS_Description', 'Articles table', 'SCHEMA', SCHEMA_NAME(), 'TABLE', 'article'"
	if sql != want {
		t.Fatalf("AddCommentOnTable = %q, want %q", sql, want)
	}
}

func TestSQLServerDropCommentFromTable(t *testing.T) {
	d := SQLServerDialect{}
	sql, err := d.DropCommentFromTable("article")
	if err != nil {
		t.Fatalf("DropCommentFromTable returned error: %v", err)
	}
	want := "EXEC sp_dropextendedproperty 'MS_Description', 'SCHEMA', SCHEMA_NAME(), 'TABLE', 'article'"
	if sql != want {
		t.Fatalf("DropCommentFromTable = %q, want %q", sql, want)
	}
}

func TestSQLServerInsert(t *testing.T) {
	d := SQLServerDialect{}
	stmt, err := d.Insert("article", Row{
		"title":      "Hello",
		"created_at": Expr("GETDATE()"),
	})
	if err != nil {
		t.Fatalf("Insert returned error: %v", err)
	}
	wantSQL := "INSERT INTO [article] ([created_at], [title]) VALUES (GETDATE(), @p1)"
	if stmt.Query != wantSQL {
		t.Fatalf("Insert SQL:\n got: %s\nwant: %s", stmt.Query, wantSQL)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"Hello"}) {
		t.Fatalf("Insert args = %#v", stmt.Args)
	}
}

func TestSQLServerBatchInsert(t *testing.T) {
	d := SQLServerDialect{}
	stmt, err := d.BatchInsert("article", []string{"title", "status"}, [][]any{
		{"Hello", "draft"},
		{"World", "published"},
	})
	if err != nil {
		t.Fatalf("BatchInsert returned error: %v", err)
	}
	wantSQL := "INSERT INTO [article] ([title], [status]) VALUES (@p1, @p2), (@p3, @p4)"
	if stmt.Query != wantSQL {
		t.Fatalf("BatchInsert SQL:\n got: %s\nwant: %s", stmt.Query, wantSQL)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"Hello", "draft", "World", "published"}) {
		t.Fatalf("BatchInsert args = %#v", stmt.Args)
	}
}

func TestSQLServerUpdate(t *testing.T) {
	d := SQLServerDialect{}
	stmt, err := d.Update("article", Row{"title": "Updated"}, "id = @p1", 42)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	wantSQL := "UPDATE [article] SET [title] = @p1 WHERE id = @p1"
	if stmt.Query != wantSQL {
		t.Fatalf("Update SQL:\n got: %s\nwant: %s", stmt.Query, wantSQL)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"Updated", 42}) {
		t.Fatalf("Update args = %#v", stmt.Args)
	}
}

func TestSQLServerDelete(t *testing.T) {
	d := SQLServerDialect{}
	stmt, err := d.Delete("article", "id = @p1", 42)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	wantSQL := "DELETE FROM [article] WHERE id = @p1"
	if stmt.Query != wantSQL {
		t.Fatalf("Delete SQL:\n got: %s\nwant: %s", stmt.Query, wantSQL)
	}
	if !reflect.DeepEqual(stmt.Args, []any{42}) {
		t.Fatalf("Delete args = %#v", stmt.Args)
	}
}

func TestSQLServerAcquireLockSQL(t *testing.T) {
	d := SQLServerDialect{}
	stmt, err := d.AcquireLockSQL("migration_lock", 30)
	if err != nil {
		t.Fatalf("AcquireLockSQL returned error: %v", err)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"migration_lock", 30}) {
		t.Fatalf("AcquireLockSQL args = %#v", stmt.Args)
	}
	if stmt.Query == "" {
		t.Fatalf("AcquireLockSQL query is empty")
	}
}

func TestSQLServerReleaseLockSQL(t *testing.T) {
	d := SQLServerDialect{}
	stmt, err := d.ReleaseLockSQL("migration_lock")
	if err != nil {
		t.Fatalf("ReleaseLockSQL returned error: %v", err)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"migration_lock"}) {
		t.Fatalf("ReleaseLockSQL args = %#v", stmt.Args)
	}
	if stmt.Query == "" {
		t.Fatalf("ReleaseLockSQL query is empty")
	}
}

func TestSQLServerQuoteIndexColumn(t *testing.T) {
	d := SQLServerDialect{}
	if got := d.QuoteIndexColumn("LOWER(email)"); got != "LOWER(email)" {
		t.Fatalf("QuoteIndexColumn expression = %q, want %q", got, "LOWER(email)")
	}
	if got := d.QuoteIndexColumn("user_id"); got != "[user_id]" {
		t.Fatalf("QuoteIndexColumn column = %q, want %q", got, "[user_id]")
	}
}
