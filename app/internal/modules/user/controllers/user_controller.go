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

	user, sErr := controller.UserService.Register(&registerRequest)
	if sErr != nil {
		status, message := errors.HandleServiceError(sErr)
		c.JSON(status, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("user with id %s registered successfully", user.ID)})
}

func (controller *Controller) Login(c *gin.Context) {
	var loginRequest requests.LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		errors.Init()
		errors.SetFromErrors(err, &loginRequest)
		c.JSON(http.StatusBadRequest, gin.H{"message": errors.Get()})
		return
	}

	tokens, sErr := controller.UserService.Login(&loginRequest)
	if sErr != nil {
		status, message := errors.HandleServiceError(sErr)
		c.JSON(status, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": tokens.AccessToken, "refresh_token": tokens.RefreshToken})
}
