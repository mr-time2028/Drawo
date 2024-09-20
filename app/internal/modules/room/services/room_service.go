package services

import (
	roomModel "drawo/internal/modules/room/models"
	roomRepository "drawo/internal/modules/room/repositories"
	"drawo/internal/modules/room/requests"
	userHelper "drawo/internal/modules/user/helpers"
	userModel "drawo/internal/modules/user/models"
	userRepository "drawo/internal/modules/user/repositories"
	"drawo/pkg/errors"
)

type RoomService struct {
	roomRepository roomRepository.RoomRepositoryInterface
	userRepository userRepository.UserRepositoryInterface
}

func New() *RoomService {
	return &RoomService{
		roomRepository: roomRepository.New(),
		userRepository: userRepository.New(),
	}
}

func (roomService *RoomService) CreatePrivateRoom(
	identifier *userModel.User,
	roomRequest *requests.RoomRequest,
) (*roomModel.Room, *errors.TypedError) {
	// hash password
	hashedPassword, err := userHelper.HashPassword(roomRequest.Password)
	if err != nil {
		return nil, &errors.TypedError{
			Message: "cannot hash password",
			Field:   "password",
			Error:   errors.InternalServerErr,
		}
	}

	// create room with user id and hashed password
	room := &roomModel.Room{
		Name:       roomRequest.Name,
		Identifier: identifier,
		Password:   string(hashedPassword),
		Type:       roomModel.Private,
	}
	newRoom, err := roomService.roomRepository.InsertOneRoom(room)
	if err != nil {
		return nil, &errors.TypedError{
			Message: "cannot create room",
			Field:   "room",
			Error:   errors.InternalServerErr,
		}
	}

	return newRoom, nil
}
