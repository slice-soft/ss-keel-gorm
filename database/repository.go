package database

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repository is the generic CRUD contract for database modules.
// It mirrors core.Repository[T, ID] so that this package can be used
// without importing ss-keel-core.
type Repository[T any, ID any] interface {
	FindByID(ctx context.Context, id ID) (*T, error)
	FindAll(ctx context.Context, q PageQuery) (Page[T], error)
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, id ID, entity *T) error
	Delete(ctx context.Context, id ID) error
}

// PageQuery holds pagination parameters.
type PageQuery struct {
	Page  int
	Limit int
}

// Page is a paginated result set.
type Page[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

// NewPage constructs a Page from a slice, total count, and pagination params.
func NewPage[T any](items []T, total, page, limit int) Page[T] {
	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return Page[T]{
		Data:       items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}

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
var _ Repository[any, any] = (*GormRepository[any, any])(nil)

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
func (r *GormRepository[T, ID]) FindAll(ctx context.Context, q PageQuery) (Page[T], error) {
	var items []T
	var total int64

	db := r.db.WithContext(ctx).Model(new(T))

	if err := db.Count(&total).Error; err != nil {
		return Page[T]{}, err
	}

	offset := (q.Page - 1) * q.Limit
	if err := db.Offset(offset).Limit(q.Limit).Find(&items).Error; err != nil {
		return Page[T]{}, err
	}

	return NewPage(items, int(total), q.Page, q.Limit), nil
}

// Create inserts a new entity into the database.
func (r *GormRepository[T, ID]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// Update replaces all fields of the entity identified by id.
// The entity must have its primary key set for GORM to resolve the record.
func (r *GormRepository[T, ID]) Update(ctx context.Context, _ ID, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
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
