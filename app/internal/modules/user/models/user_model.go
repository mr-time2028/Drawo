package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID          string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Username    string `gorm:"255"`
	Password    string `gorm:"60"`
	IsActive    bool   `gorm:"default:false"`
	IsSuperuser bool   `gorm:"default:false"`
}
