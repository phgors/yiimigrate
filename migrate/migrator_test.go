package migrate

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestMySQLDialectCreatesMigrationTableSQL(t *testing.T) {
	dialect := MySQLDialect{}

	got := dialect.CreateMigrationTableSQL("migration")
	want := "CREATE TABLE IF NOT EXISTS `migration` (`version` varchar(180) NOT NULL PRIMARY KEY, `apply_time` int NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"

	if got != want {
		t.Fatalf("CreateMigrationTableSQL() = %q, want %q", got, want)
	}
}

func TestMySQLDialectQuotesIdentifiers(t *testing.T) {
	dialect := MySQLDialect{}

	if got := dialect.QuoteTable("app.user"); got != "`app`.`user`" {
		t.Fatalf("QuoteTable() = %q", got)
	}
	if got := dialect.QuoteColumn("user"); got != "`user`" {
		t.Fatalf("QuoteColumn() = %q", got)
	}
}

func TestMigratorAppliedPendingAndHistory(t *testing.T) {
	state := newFakeSQLState()
	state.applied["m20260613_100000_create_user"] = 100
	state.applied["m20260613_110000_create_post"] = 110
	db := openFakeSQLDB(t, state)
	migrator := NewMigrator(db, MySQLDialect{}, []Migration{
		testMigration{name: "m20260613_110000_create_post"},
		testMigration{name: "m20260613_100000_create_user"},
		testMigration{name: "m20260613_120000_create_comment"},
	})

	applied, err := migrator.Applied(context.Background())
	if err != nil {
		t.Fatalf("Applied() error = %v", err)
	}
	if got, want := appliedVersions(applied), []string{"m20260613_100000_create_user", "m20260613_110000_create_post"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Applied() versions = %#v, want %#v", got, want)
	}

	pending, err := migrator.Pending(context.Background())
	if err != nil {
		t.Fatalf("Pending() error = %v", err)
	}
	if got, want := migrationNames(pending), []string{"m20260613_120000_create_comment"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Pending() names = %#v, want %#v", got, want)
	}

	history, err := migrator.History(context.Background(), 1)
	if err != nil {
		t.Fatalf("History() error = %v", err)
	}
	if got, want := appliedVersions(history), []string{"m20260613_110000_create_post"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("History() versions = %#v, want %#v", got, want)
	}
}

func TestMigratorUpAppliesPendingMigrationInTransaction(t *testing.T) {
	state := newFakeSQLState()
	db := openFakeSQLDB(t, state)
	first := testMigration{
		name: "m20260613_100000_create_user",
		up: func(ctx context.Context, m *MigrationContext) error {
			return m.Execute(ctx, "CREATE TABLE `user` (`id` int)")
		},
	}
	second := testMigration{name: "m20260613_110000_create_post"}
	migrator := NewMigrator(db, MySQLDialect{}, []Migration{second, first})

	applied, err := migrator.Up(context.Background(), 1)
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	if got, want := applied, []string{"m20260613_100000_create_user"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Up() applied = %#v, want %#v", got, want)
	}
	if _, ok := state.appliedVersion("m20260613_100000_create_user"); !ok {
		t.Fatalf("migration was not recorded as applied")
	}
	if got := state.commitCount(); got != 1 {
		t.Fatalf("commits = %d, want 1", got)
	}
	if got := state.rollbackCount(); got != 0 {
		t.Fatalf("rollbacks = %d, want 0", got)
	}
}

func TestMigratorUpRollsBackWhenMigrationFails(t *testing.T) {
	state := newFakeSQLState()
	db := openFakeSQLDB(t, state)
	boom := errors.New("boom")
	migrator := NewMigrator(db, MySQLDialect{}, []Migration{
		testMigration{
			name: "m20260613_100000_create_user",
			up: func(ctx context.Context, m *MigrationContext) error {
				if err := m.Execute(ctx, "CREATE TABLE `user` (`id` int)"); err != nil {
					return err
				}
				return boom
			},
		},
	})

	_, err := migrator.Up(context.Background(), 0)
	if !errors.Is(err, boom) {
		t.Fatalf("Up() error = %v, want %v", err, boom)
	}
	if _, ok := state.appliedVersion("m20260613_100000_create_user"); ok {
		t.Fatalf("failed migration was recorded as applied")
	}
	if got := state.commitCount(); got != 0 {
		t.Fatalf("commits = %d, want 0", got)
	}
	if got := state.rollbackCount(); got != 1 {
		t.Fatalf("rollbacks = %d, want 1", got)
	}
}

func TestMigratorDownRevertsLatestMigrationInTransaction(t *testing.T) {
	state := newFakeSQLState()
	state.applied["m20260613_100000_create_user"] = 100
	state.applied["m20260613_110000_create_post"] = 110
	db := openFakeSQLDB(t, state)
	migrator := NewMigrator(db, MySQLDialect{}, []Migration{
		testMigration{name: "m20260613_100000_create_user"},
		testMigration{
			name: "m20260613_110000_create_post",
			down: func(ctx context.Context, m *MigrationContext) error {
				return m.Execute(ctx, "DROP TABLE `post`")
			},
		},
	})

	reverted, err := migrator.Down(context.Background(), 1)
	if err != nil {
		t.Fatalf("Down() error = %v", err)
	}
	if got, want := reverted, []string{"m20260613_110000_create_post"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Down() reverted = %#v, want %#v", got, want)
	}
	if _, ok := state.appliedVersion("m20260613_110000_create_post"); ok {
		t.Fatalf("migration was not removed from applied records")
	}
	if got := state.commitCount(); got != 1 {
		t.Fatalf("commits = %d, want 1", got)
	}
}

func TestMigratorDownReturnsMigrationNotFound(t *testing.T) {
	state := newFakeSQLState()
	state.applied["m20260613_100000_missing"] = 100
	db := openFakeSQLDB(t, state)
	migrator := NewMigrator(db, MySQLDialect{}, nil)

	_, err := migrator.Down(context.Background(), 1)
	if !errors.Is(err, ErrMigrationNotFound) {
		t.Fatalf("Down() error = %v, want ErrMigrationNotFound", err)
	}
}

func appliedVersions(applied []AppliedMigration) []string {
	versions := make([]string, 0, len(applied))
	for _, migration := range applied {
		versions = append(versions, migration.Version)
	}
	return versions
}

func migrationNames(migrations []Migration) []string {
	names := make([]string, 0, len(migrations))
	for _, migration := range migrations {
		names = append(names, migration.Name())
	}
	return names
}

type testMigration struct {
	name string
	up   func(context.Context, *MigrationContext) error
	down func(context.Context, *MigrationContext) error
}

func (m testMigration) Name() string {
	return m.name
}

func (m testMigration) Up(ctx context.Context, migration *MigrationContext) error {
	if m.up == nil {
		return nil
	}
	return m.up(ctx, migration)
}

func (m testMigration) Down(ctx context.Context, migration *MigrationContext) error {
	if m.down == nil {
		return nil
	}
	return m.down(ctx, migration)
}

var (
	fakeSQLDriverOnce sync.Once
	fakeSQLStates     sync.Map
	fakeSQLStateID    atomic.Int64
)

func openFakeSQLDB(t *testing.T, state *fakeSQLState) *sql.DB {
	t.Helper()
	fakeSQLDriverOnce.Do(func() {
		sql.Register("yiimigrate_fake_sql", fakeSQLDriver{})
	})

	name := fmt.Sprintf("state-%d", fakeSQLStateID.Add(1))
	fakeSQLStates.Store(name, state)
	t.Cleanup(func() {
		fakeSQLStates.Delete(name)
	})

	db, err := sql.Open("yiimigrate_fake_sql", name)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

type fakeSQLState struct {
	mu        sync.Mutex
	applied   map[string]int64
	execs     []string
	commits   int
	rollbacks int
}

func newFakeSQLState() *fakeSQLState {
	return &fakeSQLState{applied: map[string]int64{}}
}

func (s *fakeSQLState) appliedVersion(version string) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	applyTime, ok := s.applied[version]
	return applyTime, ok
}

func (s *fakeSQLState) commitCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.commits
}

func (s *fakeSQLState) rollbackCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rollbacks
}

type fakeSQLDriver struct{}

func (fakeSQLDriver) Open(name string) (driver.Conn, error) {
	value, ok := fakeSQLStates.Load(name)
	if !ok {
		return nil, fmt.Errorf("unknown fake SQL state %q", name)
	}
	return &fakeSQLConn{state: value.(*fakeSQLState)}, nil
}

type fakeSQLConn struct {
	state *fakeSQLState
	tx    *fakeSQLTx
}

func (c *fakeSQLConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepared statements are not implemented by fakeSQLConn")
}

func (c *fakeSQLConn) Close() error {
	return nil
}

func (c *fakeSQLConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *fakeSQLConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.tx = &fakeSQLTx{
		conn:          c,
		pendingApply:  map[string]int64{},
		pendingDelete: map[string]struct{}{},
	}
	return c.tx, nil
}

func (c *fakeSQLConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	c.state.execs = append(c.state.execs, query)
	if c.tx != nil {
		c.tx.record(query, args)
		return driver.RowsAffected(1), nil
	}
	recordMigrationExec(c.state.applied, query, args)
	return driver.RowsAffected(1), nil
}

func (c *fakeSQLConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	desc := strings.Contains(strings.ToUpper(query), "DESC")
	values := make([][]driver.Value, 0, len(c.state.applied))
	for version, applyTime := range c.state.applied {
		values = append(values, []driver.Value{version, applyTime})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i][1].(int64) == values[j][1].(int64) {
			if desc {
				return values[i][0].(string) > values[j][0].(string)
			}
			return values[i][0].(string) < values[j][0].(string)
		}
		if desc {
			return values[i][1].(int64) > values[j][1].(int64)
		}
		return values[i][1].(int64) < values[j][1].(int64)
	})
	return &fakeSQLRows{columns: []string{"version", "apply_time"}, values: values}, nil
}

type fakeSQLTx struct {
	conn          *fakeSQLConn
	pendingApply  map[string]int64
	pendingDelete map[string]struct{}
}

func (tx *fakeSQLTx) record(query string, args []driver.NamedValue) {
	recordMigrationExec(tx.pendingApply, query, args)
	if isDeleteMigrationQuery(query) && len(args) > 0 {
		tx.pendingDelete[fmt.Sprint(args[0].Value)] = struct{}{}
	}
}

func (tx *fakeSQLTx) Commit() error {
	tx.conn.state.mu.Lock()
	defer tx.conn.state.mu.Unlock()

	for version, applyTime := range tx.pendingApply {
		tx.conn.state.applied[version] = applyTime
	}
	for version := range tx.pendingDelete {
		delete(tx.conn.state.applied, version)
	}
	tx.conn.state.commits++
	tx.conn.tx = nil
	return nil
}

func (tx *fakeSQLTx) Rollback() error {
	tx.conn.state.mu.Lock()
	defer tx.conn.state.mu.Unlock()

	tx.conn.state.rollbacks++
	tx.conn.tx = nil
	return nil
}

type fakeSQLRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *fakeSQLRows) Columns() []string {
	return r.columns
}

func (r *fakeSQLRows) Close() error {
	return nil
}

func (r *fakeSQLRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func recordMigrationExec(applied map[string]int64, query string, args []driver.NamedValue) {
	if !isInsertMigrationQuery(query) || len(args) < 2 {
		return
	}
	version := fmt.Sprint(args[0].Value)
	applyTime, ok := args[1].Value.(int64)
	if !ok {
		applyTime = 0
	}
	applied[version] = applyTime
}

func isInsertMigrationQuery(query string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "INSERT INTO")
}

func isDeleteMigrationQuery(query string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "DELETE FROM")
}
