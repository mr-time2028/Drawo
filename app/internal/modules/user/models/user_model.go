package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Username    string
	Password    string
	IsActive    bool
	IsSuperuser bool
}
