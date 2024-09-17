package repositories

import (
	"drawo/internal/modules/room/models"
	"drawo/pkg/database"
	"gorm.io/gorm"
)

type RoomRepository struct {
	DB *gorm.DB
}

func New() *RoomRepository {
	return &RoomRepository{
		DB: database.GetDB(),
	}
}

func (roomRepository *RoomRepository) InsertOneRoom(room *models.Room) (*models.Room, error) {
	result := roomRepository.DB.Create(room)
	if result.Error != nil {
		return nil, result.Error
	}
	return room, nil
}

func (roomRepository *RoomRepository) GetAllRooms() ([]*models.Room, error) {
	var rooms []*models.Room
	result := roomRepository.DB.Find(&rooms)
	if result.Error != nil {
		return nil, result.Error
	}
	return rooms, nil
}
