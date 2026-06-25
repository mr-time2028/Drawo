package services

import (
	roomModel "drawo/internal/modules/room/models"
	"drawo/internal/modules/room/requests"
	userModel "drawo/internal/modules/user/models"
	"drawo/pkg/errors"
)

type RoomServiceInterface interface {
	CreatePrivateRoom(identifier *userModel.User, roomRequest *requests.RoomRequest) (*roomModel.Room, *errors.TypedError)
}
