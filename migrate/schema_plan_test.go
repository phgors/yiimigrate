package migrate

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestMigrationContextExistsHelpers(t *testing.T) {
	state := newFakeSQLState()
	state.tables["user"] = true
	state.columns["user.email"] = true
	state.indexes["user.idx-user-email"] = true
	state.foreignKeys["profile.fk-profile-user_id"] = true
	state.constraints["profile.fk-profile-user_id"] = true
	db := openFakeSQLDB(t, state)
	ctx := NewMigrationContext(db, MySQLDialect{})

	got, err := ctx.TableExists(context.Background(), "user")
	assertExists(t, got, err, true)
	got, err = ctx.ColumnExists(context.Background(), "user", "email")
	assertExists(t, got, err, true)
	got, err = ctx.IndexExists(context.Background(), "user", "idx-user-email")
	assertExists(t, got, err, true)
	got, err = ctx.ForeignKeyExists(context.Background(), "profile", "fk-profile-user_id")
	assertExists(t, got, err, true)
	got, err = ctx.ConstraintExists(context.Background(), "profile", "fk-profile-user_id")
	assertExists(t, got, err, true)
	got, err = ctx.TableExists(context.Background(), "missing")
	assertExists(t, got, err, false)
}

func TestSchemaPlanExecutesDDLAndDMLInOrder(t *testing.T) {
	state := newFakeSQLState()
	db := openFakeSQLDB(t, state)
	ctx := NewMigrationContext(db, MySQLDialect{})

	err := ctx.Schema().
		CreateTable("user", Columns().
			Add("id", ctx.UnsignedBigPrimaryKey()).
			Add("username", ctx.String(64).NotNull()),
			"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		).
		CreateIndex("idx-user-username", "user", []string{"username"}, true).
		Insert("user", Row{
			"username":   "admin",
			"created_at": Expr("UNIX_TIMESTAMP()"),
		}).
		Update("user", Row{"updated_at": Expr("UNIX_TIMESTAMP()")}, "username = ?", "admin").
		Delete("user", "username = ?", "old").
		Exec(context.Background())
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	got := state.execRecords()
	want := []fakeExecRecord{
		{query: "CREATE TABLE `user` (`id` bigint UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, `username` varchar(64) NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", args: []any{}},
		{query: "CREATE UNIQUE INDEX `idx-user-username` ON `user` (`username`)", args: []any{}},
		{query: "INSERT INTO `user` (`created_at`, `username`) VALUES (UNIX_TIMESTAMP(), ?)", args: []any{"admin"}},
		{query: "UPDATE `user` SET `updated_at` = UNIX_TIMESTAMP() WHERE username = ?", args: []any{"admin"}},
		{query: "DELETE FROM `user` WHERE username = ?", args: []any{"old"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exec records = %#v, want %#v", got, want)
	}
}

func TestSchemaPlanIfExistsMethods(t *testing.T) {
	state := newFakeSQLState()
	state.tables["user"] = true
	state.columns["user.email"] = true
	state.indexes["user.idx-user-email"] = true
	db := openFakeSQLDB(t, state)
	ctx := NewMigrationContext(db, MySQLDialect{})

	err := ctx.Schema().
		CreateTableIfNotExists(context.Background(), "user", Columns().Add("id", ctx.Integer())).
		DropTableIfExists(context.Background(), "user").
		AddColumnIfNotExists(context.Background(), "user", "email", ctx.String(128)).
		DropColumnIfExists(context.Background(), "user", "email").
		CreateIndexIfNotExists(context.Background(), "idx-user-email", "user", []string{"email"}, false).
		DropIndexIfExists(context.Background(), "idx-user-email", "user").
		Exec(context.Background())
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	got := state.execQueries()
	want := []string{
		"DROP TABLE `user`",
		"ALTER TABLE `user` DROP COLUMN `email`",
		"DROP INDEX `idx-user-email` ON `user`",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exec queries = %#v, want %#v", got, want)
	}
}

func TestSchemaPlanDryRunDoesNotExecuteSQL(t *testing.T) {
	state := newFakeSQLState()
	db := openFakeSQLDB(t, state)
	ctx := NewMigrationContext(db, MySQLDialect{})
	ctx.SetDryRun(true)

	err := ctx.Schema().
		Raw("DROP TABLE `user`").
		Insert("user", Row{"username": "admin"}).
		Exec(context.Background())
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if got := state.execQueries(); len(got) != 0 {
		t.Fatalf("dry-run executed SQL: %#v", got)
	}
}

func TestSchemaPlanReturnsStoredExistenceError(t *testing.T) {
	state := newFakeSQLState()
	state.queryErr = errors.New("metadata unavailable")
	db := openFakeSQLDB(t, state)
	ctx := NewMigrationContext(db, MySQLDialect{})

	err := ctx.Schema().
		CreateTableIfNotExists(context.Background(), "user", Columns().Add("id", ctx.Integer())).
		Exec(context.Background())
	if !errors.Is(err, state.queryErr) {
		t.Fatalf("Exec() error = %v, want %v", err, state.queryErr)
	}
}

func TestDMLBuildsDeterministicSQLAndArgs(t *testing.T) {
	dialect := MySQLDialect{}

	sql, args := dialect.Insert("user", Row{"username": "admin", "created_at": Expr("UNIX_TIMESTAMP()")})
	if sql != "INSERT INTO `user` (`created_at`, `username`) VALUES (UNIX_TIMESTAMP(), ?)" {
		t.Fatalf("Insert SQL = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"admin"}) {
		t.Fatalf("Insert args = %#v", args)
	}

	sql, args = dialect.BatchInsert("role", []string{"name", "created_at"}, [][]any{
		{"admin", Expr("UNIX_TIMESTAMP()")},
		{"member", Expr("UNIX_TIMESTAMP()")},
	})
	if sql != "INSERT INTO `role` (`name`, `created_at`) VALUES (?, UNIX_TIMESTAMP()), (?, UNIX_TIMESTAMP())" {
		t.Fatalf("BatchInsert SQL = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"admin", "member"}) {
		t.Fatalf("BatchInsert args = %#v", args)
	}

	sql, args = dialect.Update("user", Row{"updated_at": Expr("UNIX_TIMESTAMP()"), "status": 10}, "id = ?", 1)
	if sql != "UPDATE `user` SET `status` = ?, `updated_at` = UNIX_TIMESTAMP() WHERE id = ?" {
		t.Fatalf("Update SQL = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{10, 1}) {
		t.Fatalf("Update args = %#v", args)
	}

	sql, args = dialect.Delete("user", "status = ?", 0)
	if sql != "DELETE FROM `user` WHERE status = ?" {
		t.Fatalf("Delete SQL = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{0}) {
		t.Fatalf("Delete args = %#v", args)
	}
}

func assertExists(t *testing.T, got bool, err error, want bool) {
	t.Helper()
	if err != nil {
		t.Fatalf("exists helper error = %v", err)
	}
	if got != want {
		t.Fatalf("exists = %t, want %t", got, want)
	}
}
