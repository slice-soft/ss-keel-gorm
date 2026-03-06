package database

import (
	"testing"
	"time"
)

func TestConfigWithDefaults_AppliesExpectedDefaults(t *testing.T) {
	cfg := Config{}

	cfg.withDefaults()

	if cfg.Engine != EnginePostgres {
		t.Fatalf("expected default engine %q, got %q", EnginePostgres, cfg.Engine)
	}
	if cfg.SSLMode != "disable" {
		t.Fatalf("expected default ssl mode disable, got %q", cfg.SSLMode)
	}
	if cfg.TimeZone != "America/Bogota" {
		t.Fatalf("expected default timezone America/Bogota, got %q", cfg.TimeZone)
	}
	if cfg.Pool.MaxOpenConns != 25 {
		t.Fatalf("expected default max open conns 25, got %d", cfg.Pool.MaxOpenConns)
	}
	if cfg.Pool.MaxIdleConns != 5 {
		t.Fatalf("expected default max idle conns 5, got %d", cfg.Pool.MaxIdleConns)
	}
	if cfg.Pool.ConnMaxLifetime != 30*time.Minute {
		t.Fatalf("expected default conn max lifetime 30m, got %s", cfg.Pool.ConnMaxLifetime)
	}
	if cfg.Pool.ConnMaxIdleTime != 15*time.Minute {
		t.Fatalf("expected default conn max idle time 15m, got %s", cfg.Pool.ConnMaxIdleTime)
	}
}

func TestConfigWithDefaults_PreservesConfiguredValues(t *testing.T) {
	cfg := Config{
		Engine:   EngineMySQL,
		SSLMode:  "require",
		TimeZone: "UTC",
		Pool: PoolConfig{
			MaxOpenConns:    40,
			MaxIdleConns:    10,
			ConnMaxLifetime: 12 * time.Minute,
			ConnMaxIdleTime: 4 * time.Minute,
		},
	}

	cfg.withDefaults()

	if cfg.Engine != EngineMySQL {
		t.Fatalf("expected engine to remain %q, got %q", EngineMySQL, cfg.Engine)
	}
	if cfg.SSLMode != "require" {
		t.Fatalf("expected ssl mode require, got %q", cfg.SSLMode)
	}
	if cfg.TimeZone != "UTC" {
		t.Fatalf("expected timezone UTC, got %q", cfg.TimeZone)
	}
	if cfg.Pool.MaxOpenConns != 40 {
		t.Fatalf("expected max open conns 40, got %d", cfg.Pool.MaxOpenConns)
	}
	if cfg.Pool.MaxIdleConns != 10 {
		t.Fatalf("expected max idle conns 10, got %d", cfg.Pool.MaxIdleConns)
	}
	if cfg.Pool.ConnMaxLifetime != 12*time.Minute {
		t.Fatalf("expected conn max lifetime 12m, got %s", cfg.Pool.ConnMaxLifetime)
	}
	if cfg.Pool.ConnMaxIdleTime != 4*time.Minute {
		t.Fatalf("expected conn max idle time 4m, got %s", cfg.Pool.ConnMaxIdleTime)
	}
}

func TestPoolConfigWithDefaults_NormalizesValues(t *testing.T) {
	pool := PoolConfig{
		MaxOpenConns:    3,
		MaxIdleConns:    -1,
		ConnMaxLifetime: -1,
		ConnMaxIdleTime: 0,
	}

	pool.withDefaults()

	if pool.MaxOpenConns != 3 {
		t.Fatalf("expected max open conns 3, got %d", pool.MaxOpenConns)
	}
	if pool.MaxIdleConns != 3 {
		t.Fatalf("expected max idle conns clamped to 3, got %d", pool.MaxIdleConns)
	}
	if pool.ConnMaxLifetime != 30*time.Minute {
		t.Fatalf("expected default conn max lifetime 30m, got %s", pool.ConnMaxLifetime)
	}
	if pool.ConnMaxIdleTime != 15*time.Minute {
		t.Fatalf("expected default conn max idle time 15m, got %s", pool.ConnMaxIdleTime)
	}
}

