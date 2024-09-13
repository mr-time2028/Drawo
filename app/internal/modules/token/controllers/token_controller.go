package controllers

import (
	"drawo/internal/modules/token/requests"
	"drawo/internal/modules/token/services"
	"drawo/pkg/errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Controller struct {
	TokenService services.TokenServiceInterface
}

func New() *Controller {
	return &Controller{
		TokenService: services.New(),
	}
}

func (controller *Controller) GetAccessTokenByRefreshToken(c *gin.Context) {
	var refreshTokenRequest requests.RefreshTokenRequest

	if err := c.ShouldBindJSON(&refreshTokenRequest); err != nil {
		status, message := errors.HandleJsonError(err, &refreshTokenRequest)
		c.JSON(status, message)
		return
	}

	accessToken, tErr := controller.TokenService.GenerateAccessTokenByRefreshToken(refreshTokenRequest.RefreshToken)
	if tErr != nil {
		status, message := errors.HandleTypedError(tErr)
		c.JSON(status, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": gin.H{"access_token": accessToken}})
}

func (controller *Controller) VerifyAccessToken(c *gin.Context) {
	var accessTokenRequest requests.AccessTokenRequest

	if err := c.ShouldBindJSON(&accessTokenRequest); err != nil {
		status, message := errors.HandleJsonError(err, &accessTokenRequest)
		c.JSON(status, message)
		return
	}

	_, tErr := controller.TokenService.VerifyAccessToken(accessTokenRequest.AccessToken)
	if tErr != nil {
		status, message := errors.HandleTypedError(tErr)
		c.JSON(status, message)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "valid access token"})
}
