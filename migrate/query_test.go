package migrate

import (
	"context"
	"reflect"
	"testing"
)

func TestMigrationContextQueryHelpers(t *testing.T) {
	state := newFakeSQLState()
	state.rows["SELECT COUNT(*) FROM user"] = fakeQueryResult{
		columns: []string{"count"},
		values:  [][]any{{int64(2)}},
	}
	state.rows["SELECT * FROM user WHERE id = ?"] = fakeQueryResult{
		columns: []string{"id", "username", "nickname", "raw"},
		values:  [][]any{{int64(1), []byte("admin"), nil, []byte("bytes")}},
	}
	state.rows["SELECT * FROM user WHERE status = ?"] = fakeQueryResult{
		columns: []string{"id", "username"},
		values:  [][]any{{int64(1), []byte("admin")}, {int64(2), []byte("member")}},
	}
	state.rowExists["user.username = ?"] = true
	state.countRows["user.status = ?"] = 2
	db := openFakeSQLDB(t, state)
	ctx := NewMigrationContext(db, MySQLDialect{})

	value, err := ctx.QueryValue(context.Background(), "SELECT COUNT(*) FROM user")
	if err != nil {
		t.Fatalf("QueryValue() error = %v", err)
	}
	if value != int64(2) {
		t.Fatalf("QueryValue() = %#v, want 2", value)
	}

	row, err := ctx.QueryOne(context.Background(), "SELECT * FROM user WHERE id = ?", 1)
	if err != nil {
		t.Fatalf("QueryOne() error = %v", err)
	}
	wantRow := Row{"id": int64(1), "username": "admin", "nickname": nil, "raw": "bytes"}
	if !reflect.DeepEqual(row, wantRow) {
		t.Fatalf("QueryOne() = %#v, want %#v", row, wantRow)
	}

	rows, err := ctx.QueryAll(context.Background(), "SELECT * FROM user WHERE status = ?", 10)
	if err != nil {
		t.Fatalf("QueryAll() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(QueryAll()) = %d, want 2", len(rows))
	}

	exists, err := ctx.RowExists(context.Background(), "user", "username = ?", "admin")
	if err != nil {
		t.Fatalf("RowExists() error = %v", err)
	}
	if !exists {
		t.Fatalf("RowExists() = false, want true")
	}

	count, err := ctx.CountRows(context.Background(), "user", "status = ?", 10)
	if err != nil {
		t.Fatalf("CountRows() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("CountRows() = %d, want 2", count)
	}
}

func TestMigrationContextQueryEmptyResults(t *testing.T) {
	state := newFakeSQLState()
	db := openFakeSQLDB(t, state)
	ctx := NewMigrationContext(db, MySQLDialect{})

	value, err := ctx.QueryValue(context.Background(), "SELECT missing")
	if err != nil {
		t.Fatalf("QueryValue() error = %v", err)
	}
	if value != nil {
		t.Fatalf("QueryValue() = %#v, want nil", value)
	}

	row, err := ctx.QueryOne(context.Background(), "SELECT missing")
	if err != nil {
		t.Fatalf("QueryOne() error = %v", err)
	}
	if row != nil {
		t.Fatalf("QueryOne() = %#v, want nil", row)
	}

	rows, err := ctx.QueryAll(context.Background(), "SELECT missing")
	if err != nil {
		t.Fatalf("QueryAll() error = %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("len(QueryAll()) = %d, want 0", len(rows))
	}
}
