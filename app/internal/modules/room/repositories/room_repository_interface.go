package repositories

import "drawo/internal/modules/room/models"

type RoomRepositoryInterface interface {
	InsertOneRoom(room *models.Room) (*models.Room, error)
	GetAllRooms() ([]*models.Room, error)
}
