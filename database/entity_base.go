package database

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EntityBase provides common fields for GORM entities.
// Embed this in any entity to get ID, CreatedAt and UpdatedAt with GORM tags pre-configured.
//
// The generated repository does this automatically when using the official CLI templates.
type EntityBase struct {
	ID        string `json:"id" gorm:"primaryKey"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeCreate is a GORM hook that runs before a new record is inserted into the database.
// It generates a new UUID for the ID field if it is not already set.
func (e *EntityBase) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}
