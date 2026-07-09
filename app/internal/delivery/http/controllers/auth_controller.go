// Package controllers handles incoming HTTP requests and returns JSON responses.
package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	svc "drawo/internal/core/ports/services"
	"drawo/pkg/errors"
	"drawo/pkg/validator"
)

// AuthController implements the handlers for authentication routes.
type AuthController struct {
	authSvc svc.AuthService
}

// NewAuthController creates a new controller with the required dependencies.
func NewAuthController(authSvc svc.AuthService) *AuthController {
	return &AuthController{authSvc: authSvc}
}

// RegisterRequest defines the expected JSON body for the registration endpoint.
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=20"`
	Password string `json:"password" validate:"required,min=8"`
}

// Register handles user account creation.
func (ctrl *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	// 1. Parse JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		status, body := errors.New(errors.ErrBadRequest, "invalid request body").Response()
		c.JSON(status, body)
		return
	}

	// 2. Validate input fields
	if errs := validator.Struct(req); errs != nil {
		status, body := errors.ValidationError(errs)
		c.JSON(status, body)
		return
	}

	// 3. Call Service
	user, err := ctrl.authSvc.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		// Our pkg/errors helper automatically maps the error to the correct status code.
		appErr, _ := err.(*errors.AppError)
		status, body := appErr.Response()
		c.JSON(status, body)
		return
	}

	// 4. Return success (Hide password hash in response)
	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

// LoginRequest defines the expected JSON body for the login endpoint.
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Login handles credential verification and session establishment.
func (ctrl *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		status, body := errors.New(errors.ErrBadRequest, "invalid request body").Response()
		c.JSON(status, body)
		return
	}

	// Enforce the logic: Service needs IP and UserAgent for security tracking.
	ip := c.ClientIP()
	ua := c.Request.UserAgent()

	// Call Service (Logic for Single Device Policy is inside here)
	tokens, err := ctrl.authSvc.Login(c.Request.Context(), req.Username, req.Password, ip, ua)
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		status, body := appErr.Response()
		c.JSON(status, body)
		return
	}

	// Return the tokens
	c.JSON(http.StatusOK, tokens)
}

// RefreshRequest defines the expected JSON body for token rotation.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Refresh handles token rotation and security re-verification.
func (ctrl *AuthController) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		status, body := errors.New(errors.ErrBadRequest, "invalid request body").Response()
		c.JSON(status, body)
		return
	}

	tokens, err := ctrl.authSvc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		appErr, _ := err.(*errors.AppError)
		status, body := appErr.Response()
		c.JSON(status, body)
		return
	}

	c.JSON(http.StatusOK, tokens)
}
