package models

import "gorm.io/gorm"

type RoomType string

const (
	Private RoomType = "private"
	Public  RoomType = "public"
)

type Room struct {
	gorm.Model
	ID         string   `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name       string   `gorm:"size:255;not null"`
	Identifier string   `gorm:"size:36;not null"`
	Password   string   `gorm:"size:60"`
	Type       RoomType `gorm:"size:30;not null"`
}
