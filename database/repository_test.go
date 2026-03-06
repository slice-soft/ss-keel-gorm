package database

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"
)

type repoUser struct {
	ID   int `gorm:"primaryKey"`
	Name string
}

func newRepositoryDB(t *testing.T) *DBinstance {
	t.Helper()

	instance, err := New(Config{
		Engine:   EngineSQLite,
		Database: filepath.Join(t.TempDir(), "repo.db"),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = instance.Close()
	})

	if err := instance.MigrationWithError(&repoUser{}); err != nil {
		t.Fatalf("MigrationWithError returned error: %v", err)
	}

	return instance
}

func TestGormRepository_CRUD(t *testing.T) {
	ctx := context.Background()
	instance := newRepositoryDB(t)
	repo := NewGormRepository[repoUser, int](instance)

	if repo.DB() != instance.DB {
		t.Fatal("expected DB() to return underlying gorm db")
	}

	notFound, err := repo.FindByID(ctx, 9999)
	if err != nil {
		t.Fatalf("FindByID unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatalf("expected nil for not found entity, got %+v", notFound)
	}

	alice := &repoUser{Name: "Alice"}
	if err := repo.Create(ctx, alice); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	bob := &repoUser{Name: "Bob"}
	if err := repo.Create(ctx, bob); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	found, err := repo.FindByID(ctx, alice.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if found == nil || found.Name != "Alice" {
		t.Fatalf("expected Alice, got %+v", found)
	}

	alice.Name = "Alice Updated"
	if err := repo.Update(ctx, alice.ID, alice); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	updated, err := repo.FindByID(ctx, alice.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if updated == nil || updated.Name != "Alice Updated" {
		t.Fatalf("expected updated name, got %+v", updated)
	}

	page, err := repo.FindAll(ctx, PageQuery{Page: 1, Limit: 1})
	if err != nil {
		t.Fatalf("FindAll returned error: %v", err)
	}
	if page.Total != 2 || page.Page != 1 || page.Limit != 1 || page.TotalPages != 2 || len(page.Data) != 1 {
		t.Fatalf("unexpected page result: %+v", page)
	}

	if err := repo.Delete(ctx, bob.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	deleted, err := repo.FindByID(ctx, bob.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if deleted != nil {
		t.Fatalf("expected deleted user to be nil, got %+v", deleted)
	}
}

func TestGormRepositoryFromDB(t *testing.T) {
	instance := newRepositoryDB(t)
	repo := NewGormRepositoryFromDB[repoUser, int](instance.DB)
	if repo.DB() != instance.DB {
		t.Fatal("expected repository to keep provided db")
	}
}

func TestGormRepository_FindAllCountError(t *testing.T) {
	ctx := context.Background()
	instance := newRepositoryDB(t)
	db := instance.DB.Session(&gorm.Session{})

	const callbackName = "test:count-error"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*int64); ok {
			tx.AddError(errors.New("forced count error"))
		}
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}
	t.Cleanup(func() {
		db.Callback().Query().Remove(callbackName)
	})

	repo := NewGormRepositoryFromDB[repoUser, int](db)
	_, err := repo.FindAll(ctx, PageQuery{Page: 1, Limit: 10})
	if err == nil {
		t.Fatal("expected count error")
	}
	if !strings.Contains(err.Error(), "forced count error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGormRepository_FindAllQueryError(t *testing.T) {
	ctx := context.Background()
	instance := newRepositoryDB(t)

	if err := instance.DB.Create(&repoUser{Name: "Alice"}).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	db := instance.DB.Session(&gorm.Session{})
	const callbackName = "test:find-error"
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*int64); ok {
			return
		}
		tx.AddError(errors.New("forced find error"))
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}
	t.Cleanup(func() {
		db.Callback().Query().Remove(callbackName)
	})

	repo := NewGormRepositoryFromDB[repoUser, int](db)
	_, err := repo.FindAll(ctx, PageQuery{Page: 1, Limit: 10})
	if err == nil {
		t.Fatal("expected find error")
	}
	if !strings.Contains(err.Error(), "forced find error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

