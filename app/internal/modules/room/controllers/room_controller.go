package controllers

import (
	"drawo/internal/modules/room/requests"
	roomService "drawo/internal/modules/room/services"
	userService "drawo/internal/modules/user/services"
	"drawo/pkg/errors"
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

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("room with id %s created successfully", newRoom.ID)})
}
