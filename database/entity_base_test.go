package database

import (
	"path/filepath"
	"testing"
)

type entityBaseModel struct {
	EntityBase
	Name string
}

func newEntityBaseDB(t *testing.T) *DBinstance {
	t.Helper()

	instance, err := New(Config{
		Engine:   EngineSQLite,
		Database: filepath.Join(t.TempDir(), "entity_base.db"),
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = instance.Close()
	})

	if err := instance.MigrationWithError(&entityBaseModel{}); err != nil {
		t.Fatalf("MigrationWithError returned error: %v", err)
	}

	return instance
}

func TestEntityBase_BeforeCreate_GeneratesUUID(t *testing.T) {
	instance := newEntityBaseDB(t)

	m := &entityBaseModel{Name: "test"}
	if err := instance.DB.Create(m).Error; err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if m.ID == "" {
		t.Fatal("expected ID to be generated, got empty string")
	}
}

func TestEntityBase_BeforeCreate_PreservesExistingID(t *testing.T) {
	instance := newEntityBaseDB(t)

	const customID = "custom-id-123"
	m := &entityBaseModel{EntityBase: EntityBase{ID: customID}, Name: "test"}
	if err := instance.DB.Create(m).Error; err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if m.ID != customID {
		t.Fatalf("expected ID %q, got %q", customID, m.ID)
	}
}

func TestEntityBase_BeforeCreate_UniqueIDsPerRecord(t *testing.T) {
	instance := newEntityBaseDB(t)

	a := &entityBaseModel{Name: "a"}
	b := &entityBaseModel{Name: "b"}

	if err := instance.DB.Create(a).Error; err != nil {
		t.Fatalf("Create a returned error: %v", err)
	}
	if err := instance.DB.Create(b).Error; err != nil {
		t.Fatalf("Create b returned error: %v", err)
	}

	if a.ID == b.ID {
		t.Fatalf("expected unique IDs, both got %q", a.ID)
	}
}

func TestEntityBase_BeforeCreate_SetsTimestamps(t *testing.T) {
	instance := newEntityBaseDB(t)

	m := &entityBaseModel{Name: "test"}
	if err := instance.DB.Create(m).Error; err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if m.CreatedAt == 0 {
		t.Fatal("expected CreatedAt to be set")
	}
	if m.UpdatedAt == 0 {
		t.Fatal("expected UpdatedAt to be set")
	}
}
