package models

import (
	roomModel "drawo/internal/modules/room/models"
	userModel "drawo/internal/modules/user/models"
	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	SenderID   string         `gorm:"not null"`
	Sender     userModel.User `gorm:"foreignKey:SenderID;constraint:OnDelete:CASCADE;not null"`
	ReceiverID string         `gorm:"not null"`
	Receiver   userModel.User `gorm:"foreignKey:ReceiverID;constraint:OnDelete:CASCADE;not null"`
	RoomID     string         `gorm:"not null"`
	Room       roomModel.Room `gorm:"foreignKey:RoomID;constraint:OnDelete:CASCADE;not null"`
	Content    string         `gorm:"size:5000;not null"`
}
