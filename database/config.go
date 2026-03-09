package database

import (
	"time"

	"github.com/slice-soft/ss-keel-core/contracts"
	"gorm.io/gorm"
)

type Engine string

const (
	EnginePostgres  Engine = "postgres"
	EngineMySQL     Engine = "mysql"
	EngineMariaDB   Engine = "mariadb"
	EngineSQLite    Engine = "sqlite"
	EngineSQLServer Engine = "sqlserver"
	EngineOracle    Engine = "oracle"
)

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type Config struct {
	Engine     Engine
	Host       string
	Port       int
	User       string
	Password   string
	Database   string
	DSN        string
	SSLMode    string
	TimeZone   string
	Production bool
	Pool       PoolConfig
	GormConfig *gorm.Config
	SkipPing   bool
	Logger     contracts.Logger
}

func (cfg *Config) withDefaults() {
	if cfg.Engine == "" {
		cfg.Engine = EnginePostgres
	}

	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	if cfg.TimeZone == "" {
		cfg.TimeZone = "UTC"
	}

	cfg.Pool.withDefaults()
}

func (pool *PoolConfig) withDefaults() {
	if pool.MaxOpenConns <= 0 {
		pool.MaxOpenConns = 25
	}

	if pool.MaxIdleConns < 0 {
		pool.MaxIdleConns = 0
	}

	if pool.MaxIdleConns == 0 {
		pool.MaxIdleConns = 5
	}

	if pool.MaxIdleConns > pool.MaxOpenConns {
		pool.MaxIdleConns = pool.MaxOpenConns
	}

	if pool.ConnMaxLifetime <= 0 {
		pool.ConnMaxLifetime = 30 * time.Minute
	}

	if pool.ConnMaxIdleTime <= 0 {
		pool.ConnMaxIdleTime = 15 * time.Minute
	}
}
