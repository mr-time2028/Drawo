package services

import (
	"drawo/internal/modules/room/models"
	"drawo/internal/modules/room/requests"
	"drawo/pkg/errors"
)

type RoomServiceInterface interface {
	CreatePrivateRoom(authHeader string, roomRequest *requests.RoomRequest) (*models.Room, *errors.TypedError)
}
