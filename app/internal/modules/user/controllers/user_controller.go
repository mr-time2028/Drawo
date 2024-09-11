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
		status, message := errors.HandleJsonError(err, &registerRequest)
		c.JSON(status, message)
		return
	}

	user, tErr := controller.UserService.Register(&registerRequest)
	if tErr != nil {
		status, message := errors.HandleTypedError(tErr)
		c.JSON(status, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("user with id %s registered successfully", user.ID)})
}

func (controller *Controller) Login(c *gin.Context) {
	var loginRequest requests.LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		status, message := errors.HandleJsonError(err, &loginRequest)
		c.JSON(status, message)
		return
	}

	tokens, tErr := controller.UserService.Login(&loginRequest)
	if tErr != nil {
		status, message := errors.HandleTypedError(tErr)
		c.JSON(status, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": tokens.AccessToken, "refresh_token": tokens.RefreshToken})
}

func (controller *Controller) GetAccessTokenByRefreshToken(c *gin.Context) {
	var requestBody struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		status, message := errors.HandleJsonError(err, &requestBody)
		c.JSON(status, message)
		return
	}

	accessToken, tErr := controller.UserService.GenerateAccessTokenByRefreshToken(requestBody.RefreshToken)
	if tErr != nil {
		status, message := errors.HandleTypedError(tErr)
		c.JSON(status, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}

func (controller *Controller) VerifyAccessToken(c *gin.Context) {
	var requestBody struct {
		AccessToken string `json:"access_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		status, message := errors.HandleJsonError(err, &requestBody)
		c.JSON(status, message)
		return
	}

	accessToken, tErr := controller.UserService.VerifyAccessToken(requestBody.AccessToken)
	if tErr != nil {
		status, message := errors.HandleTypedError(tErr)
		c.JSON(status, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}
