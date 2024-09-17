package controllers

import (
	"drawo/internal/modules/room/requests"
	"drawo/internal/modules/room/services"
	"drawo/pkg/errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Controller struct {
	RoomService services.RoomServiceInterface
}

func New() *Controller {
	return &Controller{
		RoomService: services.New(),
	}
}

func (controller *Controller) CreatePrivateRoom(c *gin.Context) {
	var roomRequest requests.RoomRequest
	if err := c.ShouldBindJSON(&roomRequest); err != nil {
		status, message := errors.HandleJsonError(err, &roomRequest)
		c.JSON(status, message)
		return
	}

	identifier := c.Request.Header.Get("Authorization")
	newRoom, tErr := controller.RoomService.CreatePrivateRoom(identifier, &roomRequest)
	if tErr != nil {
		status, message := errors.HandleTypedError(tErr)
		c.JSON(status, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("room with id %s created successfully", newRoom.ID)})
}
