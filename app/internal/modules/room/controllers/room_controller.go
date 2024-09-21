package controllers

import (
	"drawo/internal/modules/room/requests"
	roomService "drawo/internal/modules/room/services"
	userService "drawo/internal/modules/user/services"
	"drawo/pkg/config"
	"drawo/pkg/errors"
	"drawo/pkg/websocket"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Controller struct {
	RoomService roomService.RoomServiceInterface
	UserService userService.UserServiceInterface
}

func New() *Controller {
	return &Controller{
		RoomService: roomService.New(),
		UserService: userService.New(),
	}
}

func (controller *Controller) CreatePrivateRoom(c *gin.Context) {
	var roomRequest requests.RoomRequest
	if err := c.ShouldBindJSON(&roomRequest); err != nil {
		status, message := errors.HandleJsonError(err, &roomRequest)
		c.JSON(status, message)
		return
	}

	authHeader := c.Request.Header.Get("Authorization")
	user, tErr := controller.UserService.GetUserFromAuthHeader(authHeader)
	if tErr != nil {
		status, message := errors.HandleTypedError(tErr)
		c.JSON(status, message)
		return
	}

	newRoom, tErr := controller.RoomService.CreatePrivateRoom(user, &roomRequest)
	if tErr != nil {
		status, message := errors.HandleTypedError(tErr)
		c.JSON(status, message)
		return
	}

	// get hub
	hub := websocket.GetHub()

	// add new room to hub
	room := &websocket.Room{
		ID:           newRoom.ID,
		Name:         newRoom.Name,
		IdentifierID: user.ID,
		Password:     newRoom.Password,
		Clients:      map[string]*websocket.Client{},
	}

	hub.Rooms[newRoom.ID] = room

	fmt.Println(hub.Rooms)

	config.SetConfig()
	cfg := config.GetConfig()
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s/rooms/join_room?room_id=%s",
		cfg.App.Domain, newRoom.ID)})
}
