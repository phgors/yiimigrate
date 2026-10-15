package migrate

import (
	"reflect"
	"testing"
)

// SQL Server-specific divergence only: catalog probes, application locks and
// quoting/identity quirks. Everything the adapters share in structure is
// asserted by the conformance matrix in dialect_conformance_test.go.

func TestSQLServerQuoteColumnBracketEscape(t *testing.T) {
	d := SQLServerDialect{}
	if got := d.QuoteColumn("name]"); got != "[name]]]" {
		t.Fatalf("QuoteColumn with ] = %q, want %q", got, "[name]]]")
	}
}

func TestSQLServerBuildColumnPrimaryKeyNonIdentity(t *testing.T) {
	m := NewMigrationContext(nil, SQLServerDialect{})
	d := SQLServerDialect{}

	got, err := d.BuildColumn(m.String(36).PrimaryKey())
	if err != nil {
		t.Fatalf("PrimaryKey non-identity returned error: %v", err)
	}
	if got != "NVARCHAR(36) NOT NULL PRIMARY KEY" {
		t.Fatalf("PrimaryKey non-identity = %q, want %q", got, "NVARCHAR(36) NOT NULL PRIMARY KEY")
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

func TestSQLServerAcquireLockSQL(t *testing.T) {
	d := SQLServerDialect{}
	stmt, err := d.AcquireLockSQL("migration_lock", 30)
	if err != nil {
		t.Fatalf("AcquireLockSQL error: %v", err)
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
		t.Fatalf("ReleaseLockSQL error: %v", err)
	}
	if !reflect.DeepEqual(stmt.Args, []any{"migration_lock"}) {
		t.Fatalf("ReleaseLockSQL args = %#v", stmt.Args)
	}
	if stmt.Query == "" {
		t.Fatalf("ReleaseLockSQL query is empty")
	}
}
