package models

import (
	"drawo/internal/modules/user/models"
)

type Room struct {
	ID        string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name      string `gorm:"size:255;not null"`
	OwnerID   string
	Owner     *models.User `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE"`
	Password  string       `gorm:"size:60"`
	IsPrivate bool
}
