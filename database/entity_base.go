package database

type EntityBase struct {
	ID        string `json:"id" gorm:"primaryKey"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
}
