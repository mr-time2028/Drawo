package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID          string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Username    string `gorm:"size:255;not null"`
	Password    string `gorm:"size:60;not null"`
	IsActive    bool   `gorm:"default:false"`
	IsSuperuser bool   `gorm:"default:false"`
}
