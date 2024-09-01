package repositories

import "drawo/internal/modules/room/models"

type RoomRepositoryInterface interface {
	InsertOneRoom(room *models.Room) (string, error)
	GetAllRooms() ([]*models.Room, error)
}
