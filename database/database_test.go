package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type testLogger struct {
	infos []string
}

func (l *testLogger) Info(format string, args ...interface{}) {
	l.infos = append(l.infos, fmt.Sprintf(format, args...))
}

func (l *testLogger) Warn(string, ...interface{})  {}
func (l *testLogger) Error(string, ...interface{}) {}
func (l *testLogger) Debug(string, ...interface{}) {}

type testDialector struct {
	name     string
	initErr  error
	connPool gorm.ConnPool
}

func (d testDialector) Name() string {
	if d.name != "" {
		return d.name
	}
	return "test"
}

func (d testDialector) Initialize(db *gorm.DB) error {
	if d.initErr != nil {
		return d.initErr
	}
	if d.connPool != nil {
		db.ConnPool = d.connPool
	}
	return nil
}

func (d testDialector) Migrator(*gorm.DB) gorm.Migrator {
	return nil
}

func (d testDialector) DataTypeOf(*schema.Field) string {
	return ""
}

func (d testDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return nil
}

func (d testDialector) BindVarTo(writer clause.Writer, _ *gorm.Statement, _ interface{}) {
	_ = writer.WriteByte('?')
}

func (d testDialector) QuoteTo(writer clause.Writer, str string) {
	_, _ = writer.WriteString(str)
}

func (d testDialector) Explain(sql string, _ ...interface{}) string {
	return sql
}

type wrappedConnPool struct {
	inner *sql.DB
}

func (w wrappedConnPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return w.inner.PrepareContext(ctx, query)
}

func (w wrappedConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return w.inner.ExecContext(ctx, query, args...)
}

func (w wrappedConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return w.inner.QueryContext(ctx, query, args...)
}

func (w wrappedConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return w.inner.QueryRowContext(ctx, query, args...)
}

var (
	registerPingFailDriverOnce sync.Once
	errPingFailure             = errors.New("ping failure")
)

func ensurePingFailDriverRegistered() {
	registerPingFailDriverOnce.Do(func() {
		sql.Register("pingfail", pingFailDriver{})
	})
}

type pingFailDriver struct{}

func (pingFailDriver) Open(string) (driver.Conn, error) {
	return pingFailConn{}, nil
}

type pingFailConn struct{}

func (pingFailConn) Prepare(string) (driver.Stmt, error) {
	return pingFailStmt{}, nil
}

func (pingFailConn) Close() error {
	return nil
}

func (pingFailConn) Begin() (driver.Tx, error) {
	return pingFailTx{}, nil
}

func (pingFailConn) Ping(context.Context) error {
	return errPingFailure
}

type pingFailStmt struct{}

func (pingFailStmt) Close() error {
	return nil
}

func (pingFailStmt) NumInput() int {
	return -1
}

func (pingFailStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

func (pingFailStmt) Query([]driver.Value) (driver.Rows, error) {
	return pingFailRows{}, nil
}

type pingFailTx struct{}

func (pingFailTx) Commit() error {
	return nil
}

func (pingFailTx) Rollback() error {
	return nil
}

type pingFailRows struct{}

func (pingFailRows) Columns() []string {
	return []string{"ok"}
}

func (pingFailRows) Close() error {
	return nil
}

func (pingFailRows) Next([]driver.Value) error {
	return io.EOF
}

func registerDialectorForTest(t *testing.T, engine Engine, factory DialectorFactory) {
	t.Helper()

	key := normalizeEngine(engine)

	dialectorRegistryMu.Lock()
	previous, hadPrevious := dialectorRegistry[key]
	dialectorRegistry[key] = factory
	dialectorRegistryMu.Unlock()

	t.Cleanup(func() {
		dialectorRegistryMu.Lock()
		defer dialectorRegistryMu.Unlock()

		if hadPrevious {
			dialectorRegistry[key] = previous
			return
		}

		delete(dialectorRegistry, key)
	})
}

func newSQLiteInstance(t *testing.T, production bool, logger Logger) *DBinstance {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	instance, err := New(Config{
		Engine:     EngineSQLite,
		Database:   path,
		Production: production,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	t.Cleanup(func() {
		_ = instance.Close()
	})

	return instance
}

type migrationModel struct {
	ID   int `gorm:"primaryKey"`
	Name string
}

func TestNew_SuccessAndHelpers(t *testing.T) {
	logger := &testLogger{}
	instance := newSQLiteInstance(t, true, logger)

	if instance == nil {
		t.Fatal("expected non-nil DBinstance")
	}
	if instance.GetDbInstance() == nil {
		t.Fatal("expected non-nil gorm DB")
	}
	if instance.SQLDB() == nil {
		t.Fatal("expected non-nil sql.DB")
	}
	if !instance.production {
		t.Fatal("expected production flag to be true")
	}
	if len(logger.infos) == 0 || !strings.Contains(logger.infos[0], "database connected") {
		t.Fatalf("expected connection log, got %v", logger.infos)
	}

	instance.Migration(&migrationModel{})
	if !instance.DB.Migrator().HasTable(&migrationModel{}) {
		t.Fatal("expected migration to create table")
	}

	if err := instance.MigrationWithError(&migrationModel{}); err != nil {
		t.Fatalf("MigrationWithError returned error: %v", err)
	}
}

func TestNew_ReturnsErrorWhenEngineIsNotRegistered(t *testing.T) {
	_, err := New(Config{Engine: "unknown-engine"})
	if err == nil {
		t.Fatal("expected error for unknown engine")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected not registered message, got %q", err.Error())
	}
}

func TestNew_ReturnsErrorWhenGormOpenFails(t *testing.T) {
	const engine = Engine("open-fail")
	registerDialectorForTest(t, engine, func(Config) (gorm.Dialector, error) {
		return testDialector{
			name:    "open-fail",
			initErr: errors.New("init failed"),
		}, nil
	})

	_, err := New(Config{Engine: engine})
	if err == nil {
		t.Fatal("expected error from gorm.Open")
	}
	if !strings.Contains(err.Error(), "unable to open database") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_ReturnsErrorWhenGettingSQLDBFails(t *testing.T) {
	const engine = Engine("db-fail")
	sqliteDB, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "wrapped.db"))
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteDB.Close()
	})

	registerDialectorForTest(t, engine, func(Config) (gorm.Dialector, error) {
		return testDialector{
			name:     "db-fail",
			connPool: wrappedConnPool{inner: sqliteDB},
		}, nil
	})

	_, err = New(Config{Engine: engine})
	if err == nil {
		t.Fatal("expected error from db.DB()")
	}
	if !strings.Contains(err.Error(), "unable to get sql db") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_ReturnsErrorWhenPingFails(t *testing.T) {
	ensurePingFailDriverRegistered()

	sqlDB, err := sql.Open("pingfail", "")
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	const engine = Engine("ping-fail")
	registerDialectorForTest(t, engine, func(Config) (gorm.Dialector, error) {
		return testDialector{
			name:     "ping-fail",
			connPool: sqlDB,
		}, nil
	})

	_, err = New(Config{
		Engine: engine,
		GormConfig: &gorm.Config{
			DisableAutomaticPing: true,
		},
	})
	if err == nil {
		t.Fatal("expected ping error")
	}
	if !strings.Contains(err.Error(), "unable to ping database") {
		t.Fatalf("unexpected error: %v", err)
	}

	if pingErr := sqlDB.Ping(); pingErr == nil {
		t.Fatal("expected sql.DB to be closed after ping failure")
	}
}

func TestNew_RespectsSkipPing(t *testing.T) {
	ensurePingFailDriverRegistered()

	sqlDB, err := sql.Open("pingfail", "")
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	const engine = Engine("skip-ping")
	registerDialectorForTest(t, engine, func(Config) (gorm.Dialector, error) {
		return testDialector{
			name:     "skip-ping",
			connPool: sqlDB,
		}, nil
	})

	instance, err := New(Config{
		Engine:   engine,
		SkipPing: true,
		GormConfig: &gorm.Config{
			DisableAutomaticPing: true,
		},
	})
	if err != nil {
		t.Fatalf("expected success when SkipPing is true, got error: %v", err)
	}
	_ = instance.Close()
}

func TestClose_NilCasesAndRealClose(t *testing.T) {
	var nilInstance *DBinstance
	if err := nilInstance.Close(); err != nil {
		t.Fatalf("expected nil instance close to return nil, got %v", err)
	}

	if err := (&DBinstance{}).Close(); err != nil {
		t.Fatalf("expected instance with nil sqlDB close to return nil, got %v", err)
	}

	instance := newSQLiteInstance(t, false, nil)
	if err := instance.Close(); err != nil {
		t.Fatalf("expected close to succeed, got %v", err)
	}
}

func TestMigrationWithError_ReturnsErrorWhenConnectionIsClosed(t *testing.T) {
	instance := newSQLiteInstance(t, false, nil)
	if err := instance.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if err := instance.MigrationWithError(&migrationModel{}); err == nil {
		t.Fatal("expected migration error after closing db")
	}
}

func TestNewDBinstance_PanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = NewDBinstance("", "", "", "", 0, false)
}

func TestNewDBinstance_Success(t *testing.T) {
	registerDialectorForTest(t, EnginePostgres, func(Config) (gorm.Dialector, error) {
		return sqlite.Open(filepath.Join(t.TempDir(), "newdbinstance.db")), nil
	})

	instance := NewDBinstance("localhost", "user", "pass", "db", 5432, false)
	if instance == nil || instance.DB == nil || instance.SQLDB() == nil {
		t.Fatal("expected NewDBinstance to return a valid instance")
	}
	if err := instance.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}
