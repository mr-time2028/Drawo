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
	Username        string `json:"username" validate:"required,min=3,max=20"`
	Password        string `json:"password" validate:"required,min=8,password_uppercase,password_number,password_special"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=8"`
}

// Register handles user account creation.
func (ctrl *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	// 1. Parse JSON body
	if err := c.ShouldBindJSON(&req); err != nil {
		status, body := errors.New(errors.ErrBadRequest, "invalid request body").WithCode("invalid_request_body").Response()
		c.JSON(status, body)
		return
	}

	// 2. Validate input fields
	if errs := validator.Struct(req); errs != nil {
		status, body := errors.ValidationError(errs)
		c.JSON(status, body)
		return
	}

	if req.Password != req.ConfirmPassword {
		status, body := errors.New(errors.ErrBadRequest, "passwords do not match").WithCode("passwords_do_not_match").WithField("confirm_password").Response()
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
		status, body := errors.New(errors.ErrBadRequest, "invalid request body").WithCode("invalid_request_body").Response()
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
		status, body := errors.New(errors.ErrBadRequest, "invalid request body").WithCode("invalid_request_body").Response()
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

// Logout terminates the current session and invalidates tokens.
// This endpoint requires the 'Authorization' header to be present.
func (ctrl *AuthController) Logout(c *gin.Context) {
	// 1. Extract the token from the header.
	// Note: Usually this is done by an Auth middleware, but we can do it here
	// to ensure the service gets the raw string it needs.
	authHeader := c.GetHeader("Authorization")
	tokenStr := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenStr = authHeader[7:]
	}

	if tokenStr == "" {
		c.JSON(http.StatusOK, gin.H{"message": "already logged out"})
		return
	}

	// 2. Call service to delete the session from Redis.
	// If the token is invalid or already expired, the service handles it gracefully.
	_ = ctrl.authSvc.Logout(c.Request.Context(), tokenStr)

	// 3. Return success. We always return 200 OK even if the session was already gone
	// to prevent leaking information about session validity.
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
