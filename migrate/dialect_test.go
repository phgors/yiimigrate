package migrate

import (
	"errors"
	"reflect"
	"testing"
)

func TestSQLiteCreateTableSQL(t *testing.T) {
	m := NewMigrationContext(nil, SQLiteDialect{})
	sql, err := SQLiteDialect{}.CreateTable("article", Columns().
		Add("id", m.PrimaryKey()).
		Add("title", m.String(128).NotNull()).
		Add("metadata", m.Json().Null()).
		Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")),
		"",
	)
	if err != nil {
		t.Fatalf("CreateTable returned error: %v", err)
	}

	want := `CREATE TABLE "article" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "title" VARCHAR(128) NOT NULL, "metadata" TEXT NULL, "created_at" TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`
	if sql != want {
		t.Fatalf("CreateTable SQL:\n got: %s\nwant: %s", sql, want)
	}
}

func TestSQLiteMetadataAndDMLSQL(t *testing.T) {
	d := SQLiteDialect{}

	tableSQL := d.TableExistsSQL("article")
	if tableSQL.Query != `SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table', 'view') AND name = ?` {
		t.Fatalf("TableExistsSQL = %s", tableSQL.Query)
	}
	if !reflect.DeepEqual(tableSQL.Args, []any{"article"}) {
		t.Fatalf("TableExistsSQL args = %#v", tableSQL.Args)
	}

	columnSQL := d.ColumnExistsSQL("article", "title")
	if columnSQL.Query != `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?` {
		t.Fatalf("ColumnExistsSQL = %s", columnSQL.Query)
	}
	if !reflect.DeepEqual(columnSQL.Args, []any{"article", "title"}) {
		t.Fatalf("ColumnExistsSQL args = %#v", columnSQL.Args)
	}

	stmt, err := d.Insert("article", Row{
		"title":      "Hello",
		"created_at": Expr("CURRENT_TIMESTAMP"),
	})
	if err != nil {
		t.Fatalf("Insert returned error: %v", err)
	}
	wantSQL := `INSERT INTO "article" ("created_at", "title") VALUES (CURRENT_TIMESTAMP, ?)`
	if stmt.Query != wantSQL {
		t.Fatalf("Insert SQL:\n got: %s\nwant: %s", stmt.Query, wantSQL)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"Hello"}) {
		t.Fatalf("Insert args = %#v", stmt.Args)
	}
}

func TestSQLiteUnsupportedOperationsReturnErrors(t *testing.T) {
	_, err := SQLiteDialect{}.AlterColumn("article", "title", NewMigrationContext(nil, SQLiteDialect{}).String(128))
	var unsupported *UnsupportedOperationError
	if !errors.As(err, &unsupported) {
		t.Fatalf("AlterColumn error = %v, want UnsupportedOperationError", err)
	}
	if unsupported.Dialect != "sqlite" || unsupported.Operation != "ALTER COLUMN" {
		t.Fatalf("unsupported error = %#v", unsupported)
	}
}

func TestMySQLCreateTableSQL(t *testing.T) {
	m := NewMigrationContext(nil, MySQLDialect{})
	sql, err := MySQLDialect{}.CreateTable("article", Columns().
		Add("id", m.UnsignedBigPrimaryKey()).
		Add("title", m.String(128).NotNull()).
		Add("metadata", m.Json().Null()),
		"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	)
	if err != nil {
		t.Fatalf("CreateTable returned error: %v", err)
	}

	want := "CREATE TABLE `article` (`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, `title` VARCHAR(128) NOT NULL, `metadata` JSON NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"
	if sql != want {
		t.Fatalf("CreateTable SQL:\n got: %s\nwant: %s", sql, want)
	}
}

func TestQuoteIndexColumnLeavesExpressions(t *testing.T) {
	d := MySQLDialect{}
	if got := d.QuoteIndexColumn("LOWER(email)"); got != "LOWER(email)" {
		t.Fatalf("QuoteIndexColumn expression = %s", got)
	}
	if got := d.QuoteIndexColumn("user_id"); got != "`user_id`" {
		t.Fatalf("QuoteIndexColumn column = %s", got)
	}
}
