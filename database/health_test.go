package database

import (
	"context"
	"path/filepath"
	"testing"
)

func newHealthDB(t *testing.T) *DBinstance {
	t.Helper()

	instance, err := New(Config{
		Engine:   EngineSQLite,
		Database: filepath.Join(t.TempDir(), "health.db"),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = instance.Close()
	})

	return instance
}

func TestHealthChecker_NameAndCheck(t *testing.T) {
	instance := newHealthDB(t)
	checker := NewHealthChecker(instance)

	if checker.Name() != "database" {
		t.Fatalf("expected name database, got %q", checker.Name())
	}

	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("expected health check success, got %v", err)
	}
}

func TestHealthChecker_CheckReturnsErrorWhenClosed(t *testing.T) {
	instance := newHealthDB(t)
	checker := NewHealthChecker(instance)

	if err := instance.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if err := checker.Check(context.Background()); err == nil {
		t.Fatal("expected error after closing database")
	}
}

