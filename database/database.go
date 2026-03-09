package database

import (
	"database/sql"
	"fmt"

	"github.com/slice-soft/ss-keel-core/contracts"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBinstance struct {
	production bool
	DB         *gorm.DB
	sqlDB      *sql.DB
	logger     contracts.Logger
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

	return &DBinstance{
		production: cfg.Production,
		DB:         db,
		sqlDB:      sqlDB,
		logger:     cfg.Logger,
	}, nil
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

func (db *DBinstance) Migration(models ...interface{}) {
	_ = db.DB.AutoMigrate(models...)
}

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
