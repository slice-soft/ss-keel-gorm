package database

import (
	"context"
	"errors"

	"github.com/slice-soft/ss-keel-core/contracts"
	"github.com/slice-soft/ss-keel-core/core/httpx"
	"gorm.io/gorm"
)

// GormRepository is a generic CRUD repository backed by GORM.
// It implements Repository[T, ID] and works with any engine
// registered in ss-keel-gorm (Postgres, MySQL, MariaDB, SQLite, SQLServer).
//
// Usage — embed or alias in your domain repository:
//
//	type UserRepository = database.GormRepository[User, string]
//
//	func NewUserRepository(db *database.DBinstance) *UserRepository {
//	    return database.NewGormRepository[User, string](db)
//	}
type GormRepository[T any, ID any] struct {
	db *gorm.DB
}

// Compile-time check: GormRepository implements Repository.
var _ contracts.Repository[any, any, httpx.PageQuery, httpx.Page[any]] = (*GormRepository[any, any])(nil)

// NewGormRepository creates a GormRepository backed by the given DBinstance.
func NewGormRepository[T any, ID any](instance *DBinstance) *GormRepository[T, ID] {
	return &GormRepository[T, ID]{db: instance.DB}
}

// NewGormRepositoryFromDB creates a GormRepository from a raw *gorm.DB.
// Useful when scoping to a transaction or a specific table.
func NewGormRepositoryFromDB[T any, ID any](db *gorm.DB) *GormRepository[T, ID] {
	return &GormRepository[T, ID]{db: db}
}

// FindByID returns the entity with the given ID, or nil if not found.
func (r *GormRepository[T, ID]) FindByID(ctx context.Context, id ID) (*T, error) {
	var entity T
	result := r.db.WithContext(ctx).First(&entity, "id = ?", id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &entity, result.Error
}

// FindAll returns a paginated list of all entities.
func (r *GormRepository[T, ID]) FindAll(ctx context.Context, q httpx.PageQuery) (httpx.Page[T], error) {
	var items []T
	var total int64

	db := r.db.WithContext(ctx).Model(new(T))

	if err := db.Count(&total).Error; err != nil {
		return httpx.Page[T]{}, err
	}

	offset := (q.Page - 1) * q.Limit
	if err := db.Offset(offset).Limit(q.Limit).Find(&items).Error; err != nil {
		return httpx.Page[T]{}, err
	}

	return httpx.NewPage(items, int(total), q.Page, q.Limit), nil
}

// Create inserts a new entity into the database.
func (r *GormRepository[T, ID]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Update replaces ALL fields of the entity (equivalent to HTTP PUT).
// The entity must have its primary key set so GORM can locate the record.
// Use Patch for partial updates (equivalent to HTTP PATCH).
func (r *GormRepository[T, ID]) Update(ctx context.Context, _ ID, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Patch applies a partial update — only the non-zero fields in patch are written
// to the database (equivalent to HTTP PATCH).
// Use Update when you want to replace every field (HTTP PUT semantics).
//
// Caveat: GORM skips zero-value fields (0, "", false, nil). To explicitly set a
// field to its zero value use a map[string]any and r.DB().Model(...).Updates(map).
func (r *GormRepository[T, ID]) Patch(ctx context.Context, id ID, patch *T) error {
	return r.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(patch).Error
}

// Delete removes the entity with the given ID from the database.
// Respects soft-delete if the model embeds gorm.Model or has a DeletedAt field.
func (r *GormRepository[T, ID]) Delete(ctx context.Context, id ID) error {
	return r.db.WithContext(ctx).Delete(new(T), "id = ?", id).Error
}

// DB returns the underlying *gorm.DB for custom queries beyond the standard contract.
func (r *GormRepository[T, ID]) DB() *gorm.DB {
	return r.db
}
