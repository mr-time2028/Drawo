package models

import (
	"drawo/internal/modules/user/models"
)

type RoomType string

const (
	Private RoomType = "private"
	Public  RoomType = "public"
)

type Room struct {
	ID           string `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name         string `gorm:"size:255;not null"`
	IdentifierID string
	Identifier   *models.User `gorm:"foreignKey:IdentifierID;constraint:OnDelete:CASCADE"`
	Password     string       `gorm:"size:60"`
	Type         RoomType     `gorm:"size:30;not null"`
}
