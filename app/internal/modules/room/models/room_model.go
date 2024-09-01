package models

type RoomType string

const (
	Private RoomType = "private"
	Public  RoomType = "public"
)

type Room struct {
	ID         string   `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name       string   `gorm:"size:255;not null"`
	Identifier string   `gorm:"size:36;not null"`
	Password   string   `gorm:"size:60"`
	Type       RoomType `gorm:"size:30;not null"`
	//Clients    map[string]*websocket.Client `gorm:"-"`
}
