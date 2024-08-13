package controllers

import (
	"drawo/internal/modules/user/requests"
	"drawo/internal/modules/user/services"
	"drawo/pkg/errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Controller struct {
	UserService services.UserServiceInterface
}

func New() *Controller {
	return &Controller{
		UserService: services.New(),
	}
}

func (controller *Controller) Register(c *gin.Context) {
	var registerRequest requests.RegisterRequest
	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		errors.Init()
		errors.SetFromErrors(err, &registerRequest)
		c.JSON(http.StatusBadRequest, gin.H{"message": errors.Get()})
		return
	}

	user, err := controller.UserService.Register(&registerRequest)
	if err != nil {
		var statusCode int
		var message gin.H
		switch err.Error {
		case errors.BadRequestErr:
			statusCode = http.StatusBadRequest
			message = gin.H{"message": gin.H{err.Field: []string{err.Message}}}
		case errors.InternalServerErr:
			statusCode = http.StatusInternalServerError
			message = gin.H{"message": "internal server error"}
		}
		c.JSON(statusCode, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("user with id %s registered successfully", user.ID)})
}
