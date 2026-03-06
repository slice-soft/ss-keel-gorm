package database

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

func TestRegisterDialector_ValidatesInputs(t *testing.T) {
	if err := RegisterDialector("   ", func(Config) (gorm.Dialector, error) { return nil, nil }); err == nil {
		t.Fatal("expected error for empty engine")
	}

	if err := RegisterDialector("custom", nil); err == nil {
		t.Fatal("expected error for nil factory")
	}
}

func TestRegisterDialector_StoresNormalizedEngineAndFactory(t *testing.T) {
	key := Engine("  CUSTom  ")
	wantDialector := sqlite.Open(":memory:")

	if err := RegisterDialector(key, func(cfg Config) (gorm.Dialector, error) {
		return wantDialector, nil
	}); err != nil {
		t.Fatalf("RegisterDialector returned error: %v", err)
	}

	got, err := dialectorFromConfig(Config{Engine: "custom"})
	if err != nil {
		t.Fatalf("dialectorFromConfig returned error: %v", err)
	}
	if got != wantDialector {
		t.Fatal("expected the registered dialector to be returned")
	}
}

func TestDialectorFromConfig_MissingEngine(t *testing.T) {
	_, err := dialectorFromConfig(Config{Engine: "not-registered"})
	if err == nil {
		t.Fatal("expected error for missing engine")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected not registered message, got %q", err.Error())
	}
}

func TestMustRegisterDialector_PanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected mustRegisterDialector to panic")
		}
	}()

	mustRegisterDialector("   ", func(Config) (gorm.Dialector, error) { return nil, nil })
}

func TestNormalizeEngine(t *testing.T) {
	got := normalizeEngine("  MySQL  ")
	if got != EngineMySQL {
		t.Fatalf("expected %q, got %q", EngineMySQL, got)
	}
}

func TestPostgresDialector(t *testing.T) {
	t.Run("requires fields when dsn is empty", func(t *testing.T) {
		_, err := postgresDialector(Config{})
		if err == nil {
			t.Fatal("expected error for missing postgres fields")
		}
	})

	t.Run("uses provided dsn", func(t *testing.T) {
		const dsn = "host=localhost"
		d, err := postgresDialector(Config{DSN: dsn})
		if err != nil {
			t.Fatalf("postgresDialector returned error: %v", err)
		}

		var pd *postgres.Dialector
		switch v := d.(type) {
		case postgres.Dialector:
			pd = &v
		case *postgres.Dialector:
			pd = v
		default:
			t.Fatalf("expected postgres dialector, got %T", d)
		}
		if pd.Config == nil || pd.Config.DSN != dsn {
			t.Fatalf("expected dsn %q, got %#v", dsn, pd.Config)
		}
	})

	t.Run("builds dsn from config", func(t *testing.T) {
		cfg := Config{
			Host:     "127.0.0.1",
			User:     "user",
			Password: "pass",
			Database: "db",
			Port:     5432,
			SSLMode:  "disable",
			TimeZone: "UTC",
		}
		d, err := postgresDialector(cfg)
		if err != nil {
			t.Fatalf("postgresDialector returned error: %v", err)
		}

		var pd *postgres.Dialector
		switch v := d.(type) {
		case postgres.Dialector:
			pd = &v
		case *postgres.Dialector:
			pd = v
		default:
			t.Fatalf("expected postgres dialector, got %T", d)
		}
		if !strings.Contains(pd.Config.DSN, "host=127.0.0.1") ||
			!strings.Contains(pd.Config.DSN, "user=user") ||
			!strings.Contains(pd.Config.DSN, "dbname=db") ||
			!strings.Contains(pd.Config.DSN, "port=5432") {
			t.Fatalf("unexpected postgres dsn: %q", pd.Config.DSN)
		}
	})
}

func TestMySQLDialector(t *testing.T) {
	t.Run("requires fields when dsn is empty", func(t *testing.T) {
		_, err := mysqlDialector(Config{})
		if err == nil {
			t.Fatal("expected error for missing mysql fields")
		}
	})

	t.Run("uses provided dsn", func(t *testing.T) {
		const dsn = "user:pass@tcp(localhost:3306)/db"
		d, err := mysqlDialector(Config{DSN: dsn})
		if err != nil {
			t.Fatalf("mysqlDialector returned error: %v", err)
		}

		var md *mysql.Dialector
		switch v := d.(type) {
		case mysql.Dialector:
			md = &v
		case *mysql.Dialector:
			md = v
		default:
			t.Fatalf("expected mysql dialector, got %T", d)
		}
		if md.Config == nil || md.Config.DSN != dsn {
			t.Fatalf("expected dsn %q, got %#v", dsn, md.Config)
		}
	})

	t.Run("builds dsn from config", func(t *testing.T) {
		cfg := Config{
			Host:     "localhost",
			User:     "user",
			Password: "pass",
			Database: "db",
			Port:     3306,
			TimeZone: "America/Bogota",
		}
		d, err := mysqlDialector(cfg)
		if err != nil {
			t.Fatalf("mysqlDialector returned error: %v", err)
		}

		var md *mysql.Dialector
		switch v := d.(type) {
		case mysql.Dialector:
			md = &v
		case *mysql.Dialector:
			md = v
		default:
			t.Fatalf("expected mysql dialector, got %T", d)
		}
		if !strings.Contains(md.Config.DSN, "user:pass@tcp(localhost:3306)/db") ||
			!strings.Contains(md.Config.DSN, "parseTime=true") ||
			!strings.Contains(md.Config.DSN, "loc=America%2FBogota") {
			t.Fatalf("unexpected mysql dsn: %q", md.Config.DSN)
		}
	})
}

func TestSQLiteDialector(t *testing.T) {
	t.Run("requires dsn or database", func(t *testing.T) {
		_, err := sqliteDialector(Config{})
		if err == nil {
			t.Fatal("expected error for missing sqlite dsn")
		}
	})

	t.Run("uses database field when dsn empty", func(t *testing.T) {
		d, err := sqliteDialector(Config{Database: "test.db"})
		if err != nil {
			t.Fatalf("sqliteDialector returned error: %v", err)
		}

		var sd *sqlite.Dialector
		switch v := d.(type) {
		case sqlite.Dialector:
			sd = &v
		case *sqlite.Dialector:
			sd = v
		default:
			t.Fatalf("expected sqlite dialector, got %T", d)
		}
		if sd.DSN != "test.db" {
			t.Fatalf("expected dsn test.db, got %q", sd.DSN)
		}
	})
}

func TestSQLServerDialector(t *testing.T) {
	t.Run("requires fields when dsn is empty", func(t *testing.T) {
		_, err := sqlServerDialector(Config{})
		if err == nil {
			t.Fatal("expected error for missing sqlserver fields")
		}
	})

	t.Run("uses provided dsn", func(t *testing.T) {
		const dsn = "sqlserver://user:pass@localhost:1433?database=db"
		d, err := sqlServerDialector(Config{DSN: dsn})
		if err != nil {
			t.Fatalf("sqlServerDialector returned error: %v", err)
		}

		var sd *sqlserver.Dialector
		switch v := d.(type) {
		case sqlserver.Dialector:
			sd = &v
		case *sqlserver.Dialector:
			sd = v
		default:
			t.Fatalf("expected sqlserver dialector, got %T", d)
		}
		if sd.Config == nil || sd.Config.DSN != dsn {
			t.Fatalf("expected dsn %q, got %#v", dsn, sd.Config)
		}
	})

	t.Run("builds dsn from config", func(t *testing.T) {
		d, err := sqlServerDialector(Config{
			Host:     "localhost",
			User:     "user",
			Password: "p@ss",
			Database: "db",
			Port:     1433,
		})
		if err != nil {
			t.Fatalf("sqlServerDialector returned error: %v", err)
		}

		var sd *sqlserver.Dialector
		switch v := d.(type) {
		case sqlserver.Dialector:
			sd = &v
		case *sqlserver.Dialector:
			sd = v
		default:
			t.Fatalf("expected sqlserver dialector, got %T", d)
		}
		if !strings.Contains(sd.Config.DSN, "sqlserver://user:p%40ss@localhost:1433?database=db") {
			t.Fatalf("unexpected sqlserver dsn: %q", sd.Config.DSN)
		}
	})
}

func TestRegisterDialector_PropagatesFactoryError(t *testing.T) {
	const engine = Engine("factory-error")
	wantErr := errors.New("factory failed")

	if err := RegisterDialector(engine, func(cfg Config) (gorm.Dialector, error) {
		return nil, wantErr
	}); err != nil {
		t.Fatalf("RegisterDialector returned error: %v", err)
	}

	_, err := dialectorFromConfig(Config{Engine: engine})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected factory error %v, got %v", wantErr, err)
	}
}
