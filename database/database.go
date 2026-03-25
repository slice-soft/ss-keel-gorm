package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/slice-soft/ss-keel-core/contracts"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBinstance struct {
	production bool
	DB         *gorm.DB
	sqlDB      *sql.DB
	logger     contracts.Logger
	events     chan contracts.PanelEvent
}

func New(cfg Config) (*DBinstance, error) {
	cfg.withDefaults()

	dialector, err := dialectorFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	gormConfig := cfg.GormConfig
	if gormConfig == nil {
		gormConfig = &gorm.Config{}
	}

	if gormConfig.Logger == nil {
		levelLoggerDB := logger.Info
		if cfg.Production {
			levelLoggerDB = logger.Error
		}

		gormConfig.Logger = logger.Default.LogMode(levelLoggerDB)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("unable to get sql db: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Pool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Pool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Pool.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.Pool.ConnMaxIdleTime)

	if !cfg.SkipPing {
		if err := sqlDB.Ping(); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("unable to ping database: %w", err)
		}
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("database connected [engine=%s]", cfg.Engine)
	}

	instance := &DBinstance{
		production: cfg.Production,
		DB:         db,
		sqlDB:      sqlDB,
		logger:     cfg.Logger,
		events:     make(chan contracts.PanelEvent, 256),
	}

	instance.registerCallbacks()

	return instance, nil
}

func NewDBinstance(host, user, password, database string, port int, production bool) *DBinstance {
	db, err := New(Config{
		Engine:     EnginePostgres,
		Host:       host,
		User:       user,
		Password:   password,
		Database:   database,
		Port:       port,
		Production: production,
	})
	if err != nil {
		panic(err)
	}

	return db
}

func (db *DBinstance) GetDbInstance() *gorm.DB {
	return db.DB
}

// Deprecated: Migration calls GORM AutoMigrate, which is not recommended in production.
// Keel does not run automatic migrations. Manage schema changes manually.
//
// Options:
//   - Option 1 (recommended): raw SQL files — up.sql / down.sql
//   - Option 2: external tools — goose, atlas, dbmate
//   - Option 3: CI-driven — apply SQL scripts in your pipeline
func (db *DBinstance) Migration(models ...interface{}) {
	_ = db.DB.AutoMigrate(models...)
}

// Deprecated: MigrationWithError calls GORM AutoMigrate, which is not recommended in production.
// Keel does not run automatic migrations. Manage schema changes manually.
func (db *DBinstance) MigrationWithError(models ...interface{}) error {
	return db.DB.AutoMigrate(models...)
}

func (db *DBinstance) SQLDB() *sql.DB {
	return db.sqlDB
}

func (db *DBinstance) Close() error {
	if db == nil || db.sqlDB == nil {
		return nil
	}

	return db.sqlDB.Close()
}

func (d *DBinstance) tryEmit(e contracts.PanelEvent) {
	select {
	case d.events <- e:
	default:
	}
}

func (d *DBinstance) emitGORMEvent(tx *gorm.DB, op string) {
	var elapsed time.Duration
	if v, ok := tx.InstanceGet("keel:start"); ok {
		if start, ok2 := v.(time.Time); ok2 {
			elapsed = time.Since(start)
		}
	}

	level := "info"
	detail := map[string]any{
		"operation":   op,
		"table":       tx.Statement.Table,
		"rows":        tx.Statement.RowsAffected,
		"duration_ms": elapsed.Milliseconds(),
		"slow":        elapsed > 200*time.Millisecond,
	}
	if sql := tx.Statement.SQL.String(); sql != "" {
		detail["sql"] = sql
	}
	if tx.Error != nil {
		detail["error"] = tx.Error.Error()
		level = "error"
	} else if elapsed > 200*time.Millisecond {
		level = "warn"
	}

	d.tryEmit(contracts.PanelEvent{
		Timestamp: time.Now(),
		AddonID:   "gorm",
		Label:     op,
		Detail:    detail,
		Level:     level,
	})
}

func (d *DBinstance) registerCallbacks() {
	d.DB.Callback().Query().Before("gorm:query").Register("keel:before_query", func(tx *gorm.DB) {
		tx.InstanceSet("keel:start", time.Now())
	})
	d.DB.Callback().Query().After("gorm:query").Register("keel:after_query", func(tx *gorm.DB) {
		d.emitGORMEvent(tx, "query")
	})

	d.DB.Callback().Create().Before("gorm:create").Register("keel:before_create", func(tx *gorm.DB) {
		tx.InstanceSet("keel:start", time.Now())
	})
	d.DB.Callback().Create().After("gorm:create").Register("keel:after_create", func(tx *gorm.DB) {
		d.emitGORMEvent(tx, "create")
	})

	d.DB.Callback().Update().Before("gorm:update").Register("keel:before_update", func(tx *gorm.DB) {
		tx.InstanceSet("keel:start", time.Now())
	})
	d.DB.Callback().Update().After("gorm:update").Register("keel:after_update", func(tx *gorm.DB) {
		d.emitGORMEvent(tx, "update")
	})

	d.DB.Callback().Delete().Before("gorm:delete").Register("keel:before_delete", func(tx *gorm.DB) {
		tx.InstanceSet("keel:start", time.Now())
	})
	d.DB.Callback().Delete().After("gorm:delete").Register("keel:after_delete", func(tx *gorm.DB) {
		d.emitGORMEvent(tx, "delete")
	})
}
