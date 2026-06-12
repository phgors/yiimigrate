package migrate

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

type execCall struct {
	query string
	args  []any
}

type fakeDBTX struct {
	execCalls []execCall
}

func (f *fakeDBTX) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	f.execCalls = append(f.execCalls, execCall{query: query, args: append([]any(nil), args...)})
	return nil, nil
}

func (f *fakeDBTX) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("query not implemented")
}

func (f *fakeDBTX) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

func TestSchemaPlanExecutesStatements(t *testing.T) {
	db := &fakeDBTX{}
	m := NewMigrationContext(db, SQLiteDialect{})

	err := m.Schema().
		Raw(`CREATE TABLE "article" ("id" INTEGER)`).
		Insert("article", Row{"id": 1}).
		Exec(context.Background())
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if len(db.execCalls) != 2 {
		t.Fatalf("exec call count = %d, want 2", len(db.execCalls))
	}
	if db.execCalls[1].query != `INSERT INTO "article" ("id") VALUES (?)` {
		t.Fatalf("insert query = %s", db.execCalls[1].query)
	}
}

func TestSchemaPlanDryRunSkipsExecution(t *testing.T) {
	db := &fakeDBTX{}
	m := NewMigrationContext(db, SQLiteDialect{}, WithDryRun(true))

	err := m.Schema().
		Raw(`CREATE TABLE "article" ("id" INTEGER)`).
		Exec(context.Background())
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if len(db.execCalls) != 0 {
		t.Fatalf("dry run executed %d statements", len(db.execCalls))
	}
}

func TestSchemaPlanDryRunIfNotExistsDoesNotQueryMetadata(t *testing.T) {
	db := &fakeDBTX{}
	m := NewMigrationContext(db, SQLiteDialect{}, WithDryRun(true))

	err := m.Schema().
		CreateTableIfNotExists(context.Background(), "article", Columns().
			Add("id", m.PrimaryKey()),
		).
		Exec(context.Background())
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if len(db.execCalls) != 0 {
		t.Fatalf("dry run executed %d statements", len(db.execCalls))
	}
}

func TestSchemaPlanStopsOnAccumulatedError(t *testing.T) {
	db := &fakeDBTX{}
	m := NewMigrationContext(db, SQLiteDialect{})

	err := m.Schema().
		AlterColumn("article", "title", m.String(128)).
		Raw(`CREATE TABLE "article" ("id" INTEGER)`).
		Exec(context.Background())
	if err == nil {
		t.Fatal("Exec returned nil error")
	}
	if !strings.Contains(err.Error(), "sqlite does not support ALTER COLUMN") {
		t.Fatalf("Exec error = %v", err)
	}
	if len(db.execCalls) != 0 {
		t.Fatalf("plan with accumulated error executed %d statements", len(db.execCalls))
	}
}
