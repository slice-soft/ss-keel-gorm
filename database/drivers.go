package database

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

type DialectorFactory func(cfg Config) (gorm.Dialector, error)

var (
	dialectorRegistryMu sync.RWMutex
	dialectorRegistry   = map[Engine]DialectorFactory{}
)

func init() {
	mustRegisterDialector(EnginePostgres, postgresDialector)
	mustRegisterDialector(EngineMySQL, mysqlDialector)
	mustRegisterDialector(EngineMariaDB, mysqlDialector)
	mustRegisterDialector(EngineSQLite, sqliteDialector)
	mustRegisterDialector(EngineSQLServer, sqlServerDialector)
}

func RegisterDialector(engine Engine, factory DialectorFactory) error {
	if strings.TrimSpace(string(engine)) == "" {
		return errors.New("engine is required")
	}

	if factory == nil {
		return errors.New("factory is required")
	}

	dialectorRegistryMu.Lock()
	defer dialectorRegistryMu.Unlock()

	dialectorRegistry[normalizeEngine(engine)] = factory
	return nil
}

func dialectorFromConfig(cfg Config) (gorm.Dialector, error) {
	engine := normalizeEngine(cfg.Engine)

	dialectorRegistryMu.RLock()
	factory, ok := dialectorRegistry[engine]
	dialectorRegistryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf(
			"engine %q is not registered: call RegisterDialector for custom engines (example: oracle)",
			engine,
		)
	}

	return factory(cfg)
}

func mustRegisterDialector(engine Engine, factory DialectorFactory) {
	if err := RegisterDialector(engine, factory); err != nil {
		panic(err)
	}
}

func normalizeEngine(engine Engine) Engine {
	return Engine(strings.ToLower(strings.TrimSpace(string(engine))))
}

func postgresDialector(cfg Config) (gorm.Dialector, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		if cfg.Host == "" || cfg.User == "" || cfg.Database == "" || cfg.Port == 0 {
			return nil, errors.New("postgres requires host, user, database and port when DSN is empty")
		}

		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
			cfg.Host,
			cfg.User,
			cfg.Password,
			cfg.Database,
			cfg.Port,
			cfg.SSLMode,
			cfg.TimeZone,
		)
	}

	return postgres.Open(dsn), nil
}

func mysqlDialector(cfg Config) (gorm.Dialector, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		if cfg.Host == "" || cfg.User == "" || cfg.Database == "" || cfg.Port == 0 {
			return nil, errors.New("mysql/mariadb requires host, user, database and port when DSN is empty")
		}

		location := url.QueryEscape(cfg.TimeZone)
		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=%s",
			cfg.User,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.Database,
			location,
		)
	}

	return mysql.Open(dsn), nil
}

func sqliteDialector(cfg Config) (gorm.Dialector, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		dsn = strings.TrimSpace(cfg.Database)
	}

	if dsn == "" {
		return nil, errors.New("sqlite requires database path or DSN")
	}

	return sqlite.Open(dsn), nil
}

func sqlServerDialector(cfg Config) (gorm.Dialector, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		if cfg.Host == "" || cfg.User == "" || cfg.Database == "" || cfg.Port == 0 {
			return nil, errors.New("sqlserver requires host, user, database and port when DSN is empty")
		}

		query := url.Values{}
		query.Set("database", cfg.Database)

		dsn = fmt.Sprintf(
			"sqlserver://%s:%s@%s:%d?%s",
			url.QueryEscape(cfg.User),
			url.QueryEscape(cfg.Password),
			cfg.Host,
			cfg.Port,
			query.Encode(),
		)
	}

	return sqlserver.Open(dsn), nil
}
