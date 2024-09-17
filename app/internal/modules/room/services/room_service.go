package services

import (
	"drawo/internal/modules/room/models"
	roomRepository "drawo/internal/modules/room/repositories"
	"drawo/internal/modules/room/requests"
	tokenHelper "drawo/internal/modules/token/helpers"
	userHelper "drawo/internal/modules/user/helpers"
	userReposiroty "drawo/internal/modules/user/repositories"
	"drawo/pkg/errors"
)

type RoomService struct {
	roomRepository roomRepository.RoomRepositoryInterface
	userRepository userReposiroty.UserRepositoryInterface
}

func New() *RoomService {
	return &RoomService{
		roomRepository: roomRepository.New(),
		userRepository: userReposiroty.New(),
	}
}

func (roomService *RoomService) CreatePrivateRoom(
	authHeader string,
	roomRequest *requests.RoomRequest,
) (*models.Room, *errors.TypedError) {
	// get access token from auth header
	_, claims, err := tokenHelper.VerifyAuthHeaderAccessToken(authHeader)
	if err != nil {
		return nil, &errors.TypedError{
			Message: "invalid auth header",
			Field:   "",
			Error:   errors.ForbiddenErr,
		}
	}

	// get user by user id
	userID := claims.Subject
	user, err := roomService.userRepository.GetUserByID(userID)
	if err != nil {
		return nil, &errors.TypedError{
			Message: "cannot get user from the database",
			Field:   "userID",
			Error:   errors.InternalServerErr,
		}
	}

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
	room := &models.Room{
		Name:       roomRequest.Name,
		Identifier: user,
		Password:   string(hashedPassword),
		Type:       models.Private,
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
